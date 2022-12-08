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
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

var DefaultParallelization = int(float32(runtime.NumCPU()) * 0.8) // Default to 80% of available cores
const MaxRequeues = 3

// Using SHAs for ACM 2.5.1 as defaults
const ACMDefaultAIAgentImage = "registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:482618e19dc48990bb53f46e441ce21f574c04a6e0b9ee8fe1103284e15db994"
const ACMDefaultAIInstallerImage = "registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:e0dbc04261a5f9d946d05ea40117b6c3ab33c000ae256e062f7c3e38cdf116cc"

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

		// Validate cmdline parameters
		if len(args) > 0 {
			return fmt.Errorf("Unexpected arg(s) on command-line: %s\n", strings.Join(args, " "))
		}

		versionRE := regexp.MustCompile("^[0-9]+\\.[0-9]+\\.[0-9]+$")

		if !versionRE.MatchString(release) {
			return fmt.Errorf("Invalid release specified. X.Y.Z format expected: %s", release)
		}

		if !versionRE.MatchString(hubVersion) {
			return fmt.Errorf("Invalid hub-version specified. X.Y.Z format expected: %s", hubVersion)
		}

		if maxParallel < 1 {
			maxParallel = 1
		}

		download(folder, release, url, aiImages, additionalImages, rmStale, generateImageSet, duProfile, skipImageSet, hubVersion, maxParallel)

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
	downloadCmd.Flags().StringP("hub-version", "", "", "RHACM operator version, in X.Y.Z format")
	downloadCmd.MarkFlagRequired("hub-version")
	downloadCmd.Flags().IntP("parallel", "p", DefaultParallelization, "Maximum parallel downloads")
	rootCmd.AddCommand(downloadCmd)
}

type ImageSet struct {
	Channel          string
	Version          string
	AdditionalImages []string
	DuProfile        bool
	HubVersion       string
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

func generateTarArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("tar", "czvf", path.Join(folder, artifact+".tgz"), "-C", folder, artifact)
}

func generateRemoveArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("rm", "-rf", path.Join(folder, artifact))
}

func generateMoveMappingFileCommand(tmpFolder, folder string) *exec.Cmd {
	return exec.Command("cp", path.Join(tmpFolder, "mirror/oc-mirror-workspace/mapping.txt"), path.Join(folder, "mapping.txt"))
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

func templatizeImageset(release, folder string, aiImages, additionalImages []string, duProfile bool, hubVersion string) {
	t, err := template.New("ImageSet").Parse(imageSetTemplate)
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
	r := strings.Split(release, ".")
	channel := r[0] + "." + r[1]
	version := release

	images := append(aiImages, additionalImages...)
	d := ImageSet{channel, version, images, duProfile, hubVersion}
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
	if contains(aiImages, image) {
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
	hubVersion string,
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

	if len(aiImages) == 0 {
		// No AI images specified, so use defaults
		aiImages = append(aiImages, ACMDefaultAIAgentImage, ACMDefaultAIInstallerImage)
	}

	imagesetFile := path.Join(folder, "imageset.yaml")

	if !skipImageSet {
		templatizeImageset(release, folder, aiImages, additionalImages, duProfile, hubVersion)

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

	summarize(release, hubVersion, duProfile, workers, totalImages, previous, (totalJobs - downloadFailures), downloadFailures, diff)

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

func summarize(release, hubVersion string, duProfile bool, workers int,
	totalImages, skipped, downloaded, failures int, downloadTime time.Duration) {
	fmt.Printf("\nSummary:\n\n")

	fmt.Printf("%-35s %s\n", "Release:", release)
	fmt.Printf("%-35s %s\n", "Hub Version:", hubVersion)
	fmt.Printf("%-35s %s\n", "Include DU Profile:", yesOrNo(duProfile))
	fmt.Printf("%-35s %d\n\n", "Workers:", workers)

	fmt.Printf("%-35s %d\n", "Total Images:", totalImages)
	fmt.Printf("%-35s %d\n", "Downloaded:", downloaded)
	fmt.Printf("%-35s %d\n", "Skipped (Previously Downloaded):", skipped)
	fmt.Printf("%-35s %d\n", "Download Failures:", failures)
	fmt.Printf("%-35s %s\n", "Time for Download:", downloadTime.Truncate(time.Second).String())
}
