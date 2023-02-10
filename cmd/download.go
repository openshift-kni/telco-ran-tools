/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

var DefaultParallelization = int(float32(runtime.NumCPU()) * 0.8) // Default to 80% of available cores
const MaxRequeues = 3

// splitVersion will split a version string into an X.Y string and a Z string
func splitVersion(version string) (xy, z string) {
	if version != "" {
		r := strings.Split(version, ".")
		xy = r[0] + "." + r[1]
		z = r[2]
	}
	return
}

func versionAtLeast(version, atleast string) bool {
	versionComponents := strings.Split(version, ".")
	verX, _ := strconv.Atoi(versionComponents[0])
	verY, _ := strconv.Atoi(versionComponents[1])

	atleastComponents := strings.Split(atleast, ".")
	atleastX, _ := strconv.Atoi(atleastComponents[0])
	atleastY, _ := strconv.Atoi(atleastComponents[1])

	if verX < atleastX {
		return false
	} else if verX > atleastX {
		return true
	} else {
		return verY >= atleastY
	}
}

func versionAtMost(version, atmost string) bool {
	versionComponents := strings.Split(version, ".")
	verX, _ := strconv.Atoi(versionComponents[0])
	verY, _ := strconv.Atoi(versionComponents[1])

	atmostComponents := strings.Split(atmost, ".")
	atmostX, _ := strconv.Atoi(atmostComponents[0])
	atmostY, _ := strconv.Atoi(atmostComponents[1])

	if verX > atmostX {
		return false
	} else if verX < atmostX {
		return true
	} else {
		return verY <= atmostY
	}
}

// deprecatedHubVersionToAcmMce translates the hubVersion to acmVersion and mceVersions to provide
// minimal backwards-compatible support for the deprecated --hub-version option, which uses an invalid
// assumption that the Z value of the ACM and MCE Z-stream versions are the same. Because this assumption
// is invalid, the --hub-version is replaced with separate --acm-version and --mce-version options.
// This function uses the previous (invalid) translation in order to support users that have not updated
// to using the new options.
func deprecatedHubVersionToAcmMce(hubVersion string) (acmVersion, mceVersion string) {
	acmVersion = hubVersion

	xy, z := splitVersion(hubVersion)

	if xy == "2.6" {
		mceVersion = "2.1." + z
	} else if xy == "2.5" {
		mceVersion = "2.0." + z
	} else {
		fmt.Fprintf(os.Stderr, "Error: Use of --hub-version is unsupported for %s. Please use --acm-version and --mce-version options\n", hubVersion)
		os.Exit(1)
	}

	return
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Downloads and pre-caches artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get cmdline parameters
		folder, _ := cmd.Flags().GetString("folder")
		release, _ := cmd.Flags().GetString("release")
		url, _ := cmd.Flags().GetString("rootfs-url")
		aiImages, _ := cmd.Flags().GetStringSlice("ai-img")
		additionalImages, _ := cmd.Flags().GetStringSlice("img")
		rmStale, _ := cmd.Flags().GetBool("rm-stale")
		generateImageSet, _ := cmd.Flags().GetBool("generate-imageset")
		duProfile, _ := cmd.Flags().GetBool("du-profile")
		skipImageSet, _ := cmd.Flags().GetBool("skip-imageset")
		hubVersion, _ := cmd.Flags().GetString("hub-version")
		maxParallel, _ := cmd.Flags().GetInt("parallel")
		acmVersion, _ := cmd.Flags().GetString("acm-version")
		mceVersion, _ := cmd.Flags().GetString("mce-version")

		// Validate cmdline parameters
		if len(args) > 0 {
			return fmt.Errorf("Unexpected arg(s) on command-line: %s\n", strings.Join(args, " "))
		}

		versionRE := regexp.MustCompile("^[0-9]+\\.[0-9]+\\.[0-9]+$")

		if !versionRE.MatchString(release) {
			return fmt.Errorf("Invalid release specified. X.Y.Z format expected: %s", release)
		}

		// Either hubVersion or mceVersion must be set
		if (hubVersion != "" && acmVersion != "") || (hubVersion != "" && mceVersion != "") {
			return fmt.Errorf("--hub-version is deprecated, and conflicts with --mce-version and --acm-version options")
		}

		if hubVersion != "" {
			if !versionRE.MatchString(hubVersion) {
				return fmt.Errorf("Invalid hub-version specified. X.Y.Z format expected: %s", hubVersion)
			}

			acmVersion, mceVersion = deprecatedHubVersionToAcmMce(hubVersion)
		}

		if mceVersion == "" {
			return fmt.Errorf("MCE version must be specified with --mce-version")
		}

		if !versionRE.MatchString(mceVersion) {
			return fmt.Errorf("Invalid mce-version specified. X.Y.Z format expected: %s", hubVersion)
		}

		if duProfile && acmVersion == "" {
			return fmt.Errorf("ACM version must be specified with --acm-version when --du-profile is selected")
		}

		if acmVersion != "" {
			if !versionRE.MatchString(acmVersion) {
				return fmt.Errorf("Invalid acm-version specified. X.Y.Z format expected: %s", hubVersion)
			}
		}

		if maxParallel < 1 {
			maxParallel = 1
		}

		err := verifyImagesExist(append(aiImages, additionalImages...))
		if err != nil {
			// Explicitly printing error here rather than returning err to avoid "usage" message in output
			fmt.Fprintln(os.Stderr, "Error: ", err)
			os.Exit(1)
		}

		download(folder, release, url, aiImages, additionalImages, rmStale, generateImageSet, duProfile, skipImageSet, acmVersion, mceVersion, maxParallel)

		return nil
	},
	Version: Version,
}

func init() {
	downloadCmd.Flags().StringP("folder", "f", "", "Folder to download artifacts")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("release", "r", "", "OpenShift release version, in X.Y.Z format")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("rootfs-url", "u", "", "rootFS URL")
	downloadCmd.Flags().StringSliceP("ai-img", "i", []string{}, "Assisted Installer Image(s)")
	downloadCmd.Flags().StringSliceP("img", "a", []string{}, "Additional Image(s)")
	downloadCmd.Flags().BoolP("rm-stale", "s", false, "Remove stale images")
	downloadCmd.Flags().Bool("generate-imageset", false, "Generate imageset.yaml only")
	downloadCmd.Flags().Bool("du-profile", false, "Pre-cache telco 5G DU operators")
	downloadCmd.Flags().Bool("skip-imageset", false, "Skip imageset.yaml generation")
	downloadCmd.Flags().StringP("hub-version", "", "", "(deprecated) RHACM operator version, in X.Y.Z format")
	downloadCmd.Flags().MarkDeprecated("hub-version", "please use separate --acm-version and --mce-version options")
	downloadCmd.Flags().StringP("acm-version", "", "", "Advanced Cluster Management operator version, in X.Y.Z format")
	downloadCmd.Flags().StringP("mce-version", "", "", "MultiCluster Engine operator version, in X.Y.Z format")
	downloadCmd.Flags().IntP("parallel", "p", DefaultParallelization, "Maximum parallel downloads")
	rootCmd.AddCommand(downloadCmd)
}

type ImageSet struct {
	Channel          string
	Version          string
	AdditionalImages []string
	DuProfile        bool
	AcmChannel       string
	AcmVersion       string
	MceChannel       string
	MceVersion       string
}

type ImageMapping struct {
	Image        string
	ImageMapping string
	Artifact     string
}

type DownloadJob struct {
	Image   ImageMapping
	Attempt int
}

type DownloadResult struct {
	Image   ImageMapping
	Success bool
}

var downloadFailures int = 0

func generateOcMirrorCommand(tmpDir, folder string) *exec.Cmd {
	return exec.Command("oc-mirror", "-c", path.Join(folder, "imageset.yaml"), "file://"+tmpDir+"/mirror", "--ignore-history", "--dry-run")
}

func generateSkopeoCopyCommand(folder, artifact, artifactsFile string) *exec.Cmd {
	// copies container images from registry to local disk. Notice --all arg copies all architectures for a multi-arch container image
	return exec.Command("skopeo", "copy", "--all", "docker://"+artifactsFile, "dir://"+folder+"/"+artifact, "-q", "--retry-times", "10")
}

func generateSkopeoInspectCommand(img string) *exec.Cmd {
	// Use "skopeo inspect" to verify existence of specified image in registry
	// On each retry in case of failure, skopeo doubles the delay between attempts, so only retry 5 times here
	// 5 retries = 31 seconds of delays
	// 10 retries = 17 minutes of delays
	return exec.Command("skopeo", "inspect", "--no-tags", "docker://"+img, "--retry-times", "5")
}

func generateTarArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("tar", "czvf", path.Join(folder, artifact+".tgz"), "-C", folder, artifact)
}

func generateRemoveArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("rm", "-rf", path.Join(folder, artifact))
}

func generateMoveMappingFileCommand(tmpFolder, folder string) *exec.Cmd {
	return exec.Command("cp", path.Join(tmpFolder, "mirror/oc-mirror-workspace/mapping.txt"), path.Join(folder, "mapping.txt"))
}

// verifyImagesExist uses "skopeo inspect" to verify the existence of specified images
func verifyImagesExist(images []string) error {
	var invalidImages []string

	for _, img := range images {
		fmt.Fprintf(os.Stdout, "Verifying image exists: %s\n", img)
		cmd := generateSkopeoInspectCommand(img)
		output, err := cmd.CombinedOutput()
		if err != nil {
			invalidImages = append(invalidImages, img)
			fmt.Fprintf(os.Stderr, "Failure to verify image: %s\nCommand output:\n%s\n\n", img, output)
		}
	}

	if len(invalidImages) > 0 {
		return fmt.Errorf("The following images were explicitly requested, but are not available in remote registry:\n\n%s\n", strings.Join(invalidImages, "\n"))
	}

	return nil
}

// imageDownload handles the image download job
func imageDownload(workerId int, image ImageMapping, folder string) error {
	artifactTar := image.Artifact + ".tgz"

	fmt.Fprintf(os.Stdout, "Downloading: %s\n", image.Image)

	scratchdir := path.Join(folder, fmt.Sprintf("scratch-%03d", workerId))
	_, err := os.Stat(scratchdir)
	if err == nil {
		err = os.RemoveAll(scratchdir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to remove directory %s: %e\n", path.Join(folder, image.Artifact), err)
			os.Exit(1)
		}
	}

	err = os.Mkdir(scratchdir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: Failed to mkdir %s: %s\n", scratchdir, err)
		return err
	}

	cmd := generateSkopeoCopyCommand(scratchdir, image.Artifact, image.Image)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		return err
	}

	cmd = generateTarArtifactCommand(scratchdir, image.Artifact)
	stdout, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		return err
	}

	err = os.Rename(path.Join(scratchdir, artifactTar), path.Join(folder, artifactTar))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to move file %s: %e\n", path.Join(scratchdir, artifactTar), err)
		return err
	}

	err = os.RemoveAll(scratchdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to remove directory %s: %e\n", scratchdir, err)
		// Image has been downloaded, so we won't return an error here
	}

	return nil
}

// imageDownloader: Worker for processing image download jobs
func imageDownloader(wg *sync.WaitGroup, workerId int, jobs chan DownloadJob, results chan<- DownloadResult, folder string) {
	defer wg.Done()

	for job := range jobs {
		err := imageDownload(workerId, job.Image, folder)
		if err != nil && job.Attempt < MaxRequeues {
			fmt.Fprintf(os.Stderr, "Requeueing to try again: %s\n", job.Image.Image)
			job.Attempt++
			jobs <- job
		} else {
			results <- DownloadResult{job.Image, (err == nil)}
		}

		if len(jobs) == 0 {
			// Job queue is empty, so we can exit the worker
			return
		}
	}
}

// imageDownloaderResults processes the download job results channel
func imageDownloaderResults(wg *sync.WaitGroup, results <-chan DownloadResult, totalImages int, folder string, aiImages []string, aiImagesFile *os.File, ocpImagesFile *os.File) {
	defer wg.Done()

	counter := 0
	for result := range results {
		counter++
		if result.Success {
			fmt.Fprintf(os.Stdout, "Downloaded artifact [%d/%d]: %s\n", counter, totalImages, result.Image.Artifact)
			saveToImagesFile(result.Image.Image, result.Image.ImageMapping, aiImages, aiImagesFile, ocpImagesFile)
		} else {
			fmt.Fprintf(os.Stdout, "Failed download artifact [%d/%d]: %s\n", counter, totalImages, result.Image.Artifact)
			downloadFailures++
		}
	}
}

func templatizeImageset(release, folder string, aiImages, additionalImages []string, duProfile bool, acmVersion, mceVersion string) {
	t, err := template.New("ImageSet").Funcs(template.FuncMap{
		"VersionAtLeast": versionAtLeast,
		"VersionAtMost":  versionAtMost,
	}).Parse(imageSetTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse template: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path.Join(folder, "imageset.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse imageset.yaml file: %v\n", err)
		os.Exit(1)
	}

	// TODO: Validate the release version is valid with something like oc-mirror list releases --channel=stable-<channel

	channel, _ := splitVersion(release)
	acmChannel, _ := splitVersion(acmVersion)
	mceChannel, _ := splitVersion(mceVersion)

	images := append(aiImages, additionalImages...)
	d := ImageSet{channel, release, images, duProfile, acmChannel, acmVersion, mceChannel, mceVersion}
	err = t.Execute(f, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to execute template: %v\n", err)
		os.Exit(1)
	}
}

func downloadRootFsFile(release, folder, url string) error {
	r := strings.Split(release, ".")
	channel := r[0] + "." + r[1]

	out, err := os.Create(path.Join(folder, "rhcos-live-rootfs.x86_64.img"))
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get("https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/" + channel + "/latest/rhcos-live-rootfs.x86_64.img")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Once upgraded to GO 1.18, this function can be replaced by slices.Contains
func contains(stringlist []string, s string) bool {
	for _, item := range stringlist {
		if item == s {
			return true
		}
	}
	return false
}

func saveToImagesFile(image, imageMapping string, aiImages []string, aiImagesFile *os.File, ocpImagesFile *os.File) {
	splittedImageMapping := strings.Split(imageMapping, ":")
	if contains(aiImages, image) || strings.HasPrefix(splittedImageMapping[0], "multicluster-engine/assisted-installer") {
		aiImagesFile.WriteString(image + "\n")
		if strings.Contains(splittedImageMapping[0], "assisted-installer-reporter") {
			ocpImagesFile.WriteString(image + "\n")
		}
	} else if splittedImageMapping[0] == "openshift/release-images" {
		aiImagesFile.WriteString(image + "\n")
		ocpImagesFile.WriteString(image + "\n")
	} else if splittedImageMapping[0] == "openshift/release" {
		splittedImageMapping = strings.SplitN(splittedImageMapping[1], "-", 3)
		containerName := splittedImageMapping[2]
		if containerName == "must-gather" ||
			containerName == "hyperkube" ||
			containerName == "cloud-credential-operator" ||
			containerName == "cluster-policy-controller" ||
			containerName == "pod" ||
			containerName == "cluster-config-operator" ||
			containerName == "cluster-etcd-operator" ||
			containerName == "cluster-kube-controller-manager-operator" ||
			containerName == "cluster-kube-scheduler-operator" ||
			containerName == "machine-config-operator" ||
			containerName == "etcd" ||
			containerName == "cluster-bootstrap" ||
			containerName == "cluster-ingress-operator" ||
			containerName == "cluster-kube-apiserver-operator" {
			aiImagesFile.WriteString(image + "\n")
		}
		ocpImagesFile.WriteString(image + "\n")
	} else {
		ocpImagesFile.WriteString(image + "\n")
	}
}

func download(folder, release, url string,
	aiImages, additionalImages []string,
	rmStale, generateImageSet, duProfile, skipImageSet bool,
	acmVersion, mceVersion string,
	maxParallel int) {
	tmpDir, err := ioutil.TempDir("", "fp-cli-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to create temporary directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	var rootFsFilename string
	if url != "" {
		rootFsFilename = path.Base(url)
	}

	if rootFsFilename != "" {
		_, err = os.Stat(path.Join(folder, rootFsFilename))
		if err != nil {
			fmt.Fprintf(os.Stdout, "Downloading release rootFS image %s...\n", url)
			err = downloadRootFsFile(release, folder, url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to download rootFS image: %v\n", err)
				os.Exit(1)
			}
		}
	}

	imagesetFile := path.Join(folder, "imageset.yaml")

	if !skipImageSet {
		templatizeImageset(release, folder, aiImages, additionalImages, duProfile, acmVersion, mceVersion)

		fmt.Printf("Generated %s\n", imagesetFile)
		if generateImageSet {
			os.Exit(0)
		}
	} else {
		_, err = os.Stat(imagesetFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--skip-imageset specified, but %s not found", imagesetFile)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stdout, "Generating list of pre-cached artifacts...\n")
	cmd := generateOcMirrorCommand(tmpDir, folder)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		os.Exit(1)
	}

	cmd = generateMoveMappingFileCommand(tmpDir, folder)
	stdout, err = executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		os.Exit(1)
	}

	mappingFile, err := os.Open(path.Join(folder, "mapping.txt"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open mapping.txt file: %v\n", err)
		os.Exit(1)
	}
	defer mappingFile.Close()

	aiImagesFile, err := os.OpenFile(path.Join(folder, "ai-images.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open ai-images.txt file: %v\n", err)
		os.Exit(1)
	}
	defer aiImagesFile.Close()

	ocpImagesFile, err := os.OpenFile(path.Join(folder, "ocp-images.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open ocp-images.txt file: %v\n", err)
		os.Exit(1)
	}
	defer ocpImagesFile.Close()

	fileScanner := bufio.NewScanner(mappingFile)
	fileScanner.Split(bufio.ScanLines)

	var images []ImageMapping

	for fileScanner.Scan() {
		line := fileScanner.Text()
		splittedLine := strings.Split(line, "=")
		image := splittedLine[0]
		imageMapping := splittedLine[1]
		splittedImage := strings.Split(image, "/")
		artifact := splittedImage[len(splittedImage)-1]
		artifact = strings.Replace(artifact, ":", "_", 1)

		images = append(images, ImageMapping{image, imageMapping, artifact})
	}

	if rmStale {
		expectedTarfiles := map[string]bool{}
		for _, image := range images {
			expectedTarfiles[image.Artifact+".tgz"] = true
		}

		files, err := ioutil.ReadDir(folder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to readdir: %s\n", folder)
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".tgz") && !expectedTarfiles[file.Name()] {
				fmt.Printf("Deleting stale image: %s\n", file.Name())
				fpath := path.Join(folder, file.Name())
				err = os.Remove(fpath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to delete %s: %s\n", fpath, err)
					os.Exit(1)
				}
			}
		}
	}

	totalImages := len(images)
	totalJobs := 0
	previous := 0

	// Setup channels for jobs and results
	jobs := make(chan DownloadJob, totalImages)
	results := make(chan DownloadResult, totalImages)

	// Process complete image list to determine images that haven't already been downloaded
	for _, image := range images {
		artifactTar := image.Artifact + ".tgz"
		_, err = os.Stat(path.Join(folder, artifactTar))
		if err == nil {
			// File exists, thus it's been already downloaded and tarballed, moving on...
			fmt.Fprintf(os.Stdout, "Exists: %s\n", artifactTar)
			saveToImagesFile(image.Image, image.ImageMapping, aiImages, aiImagesFile, ocpImagesFile)
			previous++
		} else {
			// Add download jobs to channel to be processed
			jobs <- DownloadJob{image, 0}
			totalJobs++
		}
	}

	fmt.Println()

	if totalJobs == 0 {
		fmt.Fprintf(os.Stdout, "All %d images previously downloaded.\n", totalImages)
		os.Exit(0)
	}

	if previous > 0 {
		fmt.Fprintf(os.Stdout, "%d images previously downloaded.\n", previous)
	}

	workers := maxParallel
	if totalJobs < workers {
		workers = totalJobs
	}

	fmt.Fprintf(os.Stdout, "Queueing %d of %d images for download, with %d workers.\n\n", totalJobs, totalImages, workers)
	startTime := time.Now()

	wg := sync.WaitGroup{}

	// Create imageDownloader workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go imageDownloader(&wg, i, jobs, results, folder)
	}

	// Create results processor
	wgResp := sync.WaitGroup{}
	wgResp.Add(1)
	go imageDownloaderResults(&wgResp, results, totalJobs, folder, aiImages, aiImagesFile, ocpImagesFile)

	// Wait for imageDownloader workers to finish
	wg.Wait()

	// Close the jobs channel to complete the task list
	close(jobs)

	// Close results channel
	close(results)
	wgResp.Wait()

	endTime := time.Now()
	diff := endTime.Sub(startTime)

	summarize(release, acmVersion, mceVersion, duProfile, workers, totalImages, previous, (totalJobs - downloadFailures), downloadFailures, diff)

	if downloadFailures > 0 {
		os.Exit(1)
	}
}

func yesOrNo(b bool) string {
	if b {
		return "Yes"
	} else {
		return "No"
	}
}

func summarize(release, acmVersion, mceVersion string, duProfile bool, workers int,
	totalImages, skipped, downloaded, failures int, downloadTime time.Duration) {
	fmt.Printf("\nSummary:\n\n")

	fmt.Printf("%-35s %s\n", "Release:", release)
	if acmVersion != "" {
		fmt.Printf("%-35s %s\n", "ACM Version:", acmVersion)
	}
	fmt.Printf("%-35s %s\n", "MCE Version:", mceVersion)
	fmt.Printf("%-35s %s\n", "Include DU Profile:", yesOrNo(duProfile))
	fmt.Printf("%-35s %d\n\n", "Workers:", workers)

	fmt.Printf("%-35s %d\n", "Total Images:", totalImages)
	fmt.Printf("%-35s %d\n", "Downloaded:", downloaded)
	fmt.Printf("%-35s %d\n", "Skipped (Previously Downloaded):", skipped)
	fmt.Printf("%-35s %d\n", "Download Failures:", failures)
	fmt.Printf("%-35s %s\n", "Time for Download:", downloadTime.Truncate(time.Second).String())
}
