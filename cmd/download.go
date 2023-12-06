/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"fmt"
	"io"
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
	"gopkg.in/yaml.v3"
)

// DefaultParallelization set to 50% of available cores.
var DefaultParallelization = int(float32(runtime.NumCPU()) * 0.5)

// MaxRequeues is the number of retries allowed for download reattempts.
const MaxRequeues = 3

// DefaultCatalogRedhatOperators is the default location of the redhat-operators catalog.
const DefaultCatalogRedhatOperators = "registry.redhat.io/redhat/redhat-operator-index"

// DefaultCatalogCertifiedOperators is the default location of the certified-operators catalog.
const DefaultCatalogCertifiedOperators = "registry.redhat.io/redhat/certified-operator-index"

// TestMode indicates whether to create a dummy tarball rather than download the image.
var TestMode = false

// ImageFilters is a list of regexp patterns to apply to the mapping.txt file when downloading.
var ImageFilters []*regexp.Regexp

// splitVersion will split a version string into an X.Y string and a Z string.
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

// ImageFilterPatterns values.
type ImageFilterPatterns struct {
	Patterns []string `yaml:"patterns,omitempty"`
}

// populateImageFilters parses the filter file to generate a list of regexp patterns.
func populateImageFilters(filterFile string) error {
	yfile, err := os.ReadFile(filterFile)
	if err != nil {
		return err
	}

	var data ImageFilterPatterns

	err = yaml.Unmarshal(yfile, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing filter file: %s\n", filterFile)
		return err
	}

	for _, pattern := range data.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed while parsing filter file: %s\n", filterFile)
			return err
		}

		fmt.Fprintf(os.Stdout, "Adding image filter pattern: %s\n", pattern)
		ImageFilters = append(ImageFilters, re)
	}

	return nil
}

// downloadCmd represents the download command.
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
		TestMode, _ = cmd.Flags().GetBool("testmode")
		filterFile, _ := cmd.Flags().GetString("filter")
		catalogRedhatOperators, _ := cmd.Flags().GetString("catalog-redhat-operators")
		catalogCertifiedOperators, _ := cmd.Flags().GetString("catalog-certified-operators")

		// Validate cmdline parameters
		if len(args) > 0 {
			return fmt.Errorf("Unexpected arg(s) on command-line: %s", strings.Join(args, " "))
		}

		versionRE := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

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
			return fmt.Errorf("Invalid mce-version specified. X.Y.Z format expected: %s", mceVersion)
		}

		if duProfile && acmVersion == "" {
			return fmt.Errorf("ACM version must be specified with --acm-version when --du-profile is selected")
		}

		if acmVersion != "" {
			if !versionRE.MatchString(acmVersion) {
				return fmt.Errorf("Invalid acm-version specified. X.Y.Z format expected: %s", acmVersion)
			}
		}

		if filterFile != "" {
			// Read patterns from file
			err := populateImageFilters(filterFile)
			if err != nil {
				return err
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

		err = download(folder, release, url, aiImages, additionalImages, rmStale, generateImageSet,
			duProfile, skipImageSet, acmVersion, mceVersion, maxParallel,
			catalogRedhatOperators, catalogCertifiedOperators)
		if err != nil {
			// Explicitly printing error here rather than returning err to avoid "usage" message in output
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		return nil
	},
	Version: Version,
}

// nolint: errcheck
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
	downloadCmd.Flags().Bool("testmode", false, "Create dummy image files rather than download")
	downloadCmd.Flags().StringP("filter", "", "", "File with list of patterns to use for filtering images by name or tag")
	downloadCmd.Flags().StringP("catalog-redhat-operators", "", "", "Override location and tag of redhat-operator-index catalog")
	downloadCmd.Flags().StringP("catalog-certified-operators", "", "", "Override location and tag of certified-operator-index catalog")
	rootCmd.AddCommand(downloadCmd)
}

// ImageSet contains data passed to imageset template.
type ImageSet struct {
	Channel                   string
	Version                   string
	AdditionalImages          []string
	DuProfile                 bool
	AcmChannel                string
	AcmVersion                string
	MceChannel                string
	MceVersion                string
	CatalogRedhatOperators    string
	CatalogCertifiedOperators string
}

// ImageMapping contains data parsed from mapping file.
type ImageMapping struct {
	Image        string
	ImageMapping string
	Artifact     string
}

// DownloadJob contains image data for download.
type DownloadJob struct {
	Image   ImageMapping
	Attempt int
}

// DownloadResult provides the job result.
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

// verifyImagesExist uses "skopeo inspect" to verify the existence of specified images.
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
		return fmt.Errorf("The following images were explicitly requested, but are not available in remote registry:\n\n%s", strings.Join(invalidImages, "\n"))
	}

	return nil
}

// Validation of operators with specified versions:
// The following types define a subset of the ImageSetConfiguration metadata needed to query the catalog with oc-mirror
// in order to verify whether specified operator versions exist. The imageset.yaml file is parsed prior to querying
// the full image list from oc-mirror, after the user has had the opportunity to modify the imageset.yaml with a private
// registry or additional operators.

// ImageSetData imageset values.
type ImageSetData struct {
	Mirror ImageSetMirror `yaml:"mirror"`
}

// ImageSetMirror imageset values.
type ImageSetMirror struct {
	Operators []ImageSetOperator `yaml:"operators,omitempty"`
}

// ImageSetChannel imageset values.
type ImageSetChannel struct {
	Name       string `yaml:"name"`
	MinVersion string `yaml:"minVersion"`
	MaxVersion string `yaml:"maxVersion"`
}

// ImageSetPackage imageset values.
type ImageSetPackage struct {
	Name     string            `yaml:"name"`
	Channels []ImageSetChannel `yaml:"channels,omitempty"`
}

// ImageSetOperator imageset values.
type ImageSetOperator struct {
	Catalog  string            `yaml:"catalog"`
	Packages []ImageSetPackage `yaml:"packages,omitempty"`
}

// checkOperatorVersion runs an oc-mirror query to get the list of versions for a given operator, grepping the results
// to verify the expected version exists.
func checkOperatorVersion(catalog, name, channel, operatorVersion string) bool {
	shellcmd := fmt.Sprintf("oc-mirror list operators --catalog %s --package %s --channel %s | grep -q '^%s$'",
		catalog, name, channel, regexp.QuoteMeta(operatorVersion))
	cmd := exec.Command("bash", "-c", shellcmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if cmd.Run() != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		os.Stderr.Sync()
		return false
	}
	return true
}

// validateVersions parses the imageset.yaml, calling checkOperatorVersion for each operator with a specified maxVersion,
// and assumes minVersion and maxVersion are set the same in order to specify the required version.
func validateVersions(folder string) error {
	yfile, err := os.ReadFile(path.Join(folder, "imageset.yaml"))
	if err != nil {
		return err
	}

	var data ImageSetData

	err = yaml.Unmarshal(yfile, &data)
	if err != nil {
		return err
	}

	failures := 0
	for _, op := range data.Mirror.Operators {
		for _, pkg := range op.Packages {
			for _, channel := range pkg.Channels {
				if len(channel.MaxVersion) > 0 {
					fmt.Fprintf(os.Stdout, "Checking %s version %s...\n", pkg.Name, channel.MaxVersion)
					if !checkOperatorVersion(op.Catalog, pkg.Name, channel.Name, channel.MaxVersion) {
						fmt.Fprintf(os.Stderr, "%s version %s not found in channel %s: Catalog %s\n\n",
							pkg.Name, channel.MaxVersion, channel.Name, op.Catalog)
						os.Stderr.Sync()
						failures++
					}
				}
			}
		}
	}

	if failures > 0 {
		return fmt.Errorf("Version checks failed for %d operator(s)", failures)
	}
	return nil
}

// runOcMirrorCommand runs the oc-mirror command with retries.
func runOcMirrorCommand(tmpDir, folder string) error {
	max_retries := 3
	for retry := 1; retry <= max_retries; retry++ {
		cmd := generateOcMirrorCommand(tmpDir, folder)
		stdout, err := executeCommand(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))

			if retry == max_retries {
				return err
			}

			fmt.Fprintf(os.Stderr, "Retrying in 10 seconds...\n")
			time.Sleep(time.Second * 10)
			fmt.Fprintf(os.Stdout, "Retrying oc-mirror command...\n")
		} else {
			break
		}
	}
	return nil
}

// imageDownload handles the image download job.
func imageDownload(workerID int, image ImageMapping, folder string) error {
	artifactTar := image.Artifact + ".tgz"

	if TestMode {
		fmt.Fprintf(os.Stdout, "TestMode: Creating dummy image file: %s\n", artifactTar)

		time.Sleep(time.Second * 1) // Sleep 1 second before creating file

		out, err := os.OpenFile(path.Join(folder, artifactTar), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()
		return nil
	}

	fmt.Fprintf(os.Stdout, "Downloading: %s\n", image.Image)

	scratchdir := path.Join(folder, fmt.Sprintf("scratch-%03d", workerID))
	_, err := os.Stat(scratchdir)
	if err == nil {
		err = os.RemoveAll(scratchdir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to remove directory %s\n", path.Join(folder, image.Artifact))
			return err
		}
	}

	err = os.Mkdir(scratchdir, 0o755)
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

// imageDownloader: Worker for processing image download jobs.
func imageDownloader(wg *sync.WaitGroup, workerID int, jobs chan DownloadJob, results chan<- DownloadResult, folder string) {
	defer wg.Done()

	for job := range jobs {
		err := imageDownload(workerID, job.Image, folder)
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

// imageDownloaderResults processes the download job results channel.
func imageDownloaderResults(wg *sync.WaitGroup, results <-chan DownloadResult, totalImages int, aiImages []string, aiImagesFile, ocpImagesFile *os.File) {
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

func templatizeImageset(
	release, folder string,
	aiImages, additionalImages []string,
	duProfile bool,
	acmVersion, mceVersion, catalogRedhatOperators, catalogCertifiedOperators string) error {
	t, err := template.New("ImageSet").Funcs(template.FuncMap{
		"VersionAtLeast": versionAtLeast,
		"VersionAtMost":  versionAtMost,
	}).Parse(imageSetTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse template:\n")
		return err
	}

	f, err := os.Create(path.Join(folder, "imageset.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse imageset.yaml file:\n")
		return err
	}

	channel, _ := splitVersion(release)
	acmChannel, _ := splitVersion(acmVersion)
	mceChannel, _ := splitVersion(mceVersion)

	catalogRedhatOperatorsLocation := catalogRedhatOperators
	if catalogRedhatOperators == "" {
		catalogRedhatOperatorsLocation = fmt.Sprintf("%s:v%s", DefaultCatalogRedhatOperators, channel)
	}

	catalogCertifiedOperatorsLocation := catalogCertifiedOperators
	if catalogRedhatOperators == "" {
		catalogCertifiedOperatorsLocation = fmt.Sprintf("%s:v%s", DefaultCatalogCertifiedOperators, channel)
	}

	images := append(aiImages, additionalImages...)
	d := ImageSet{channel, release, images, duProfile, acmChannel, acmVersion, mceChannel, mceVersion, catalogRedhatOperatorsLocation, catalogCertifiedOperatorsLocation}
	err = t.Execute(f, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to execute template:\n")
		return err
	}

	return nil
}

func downloadRootFsFile(folder, url string) error {
	rootFsFilename := path.Join(folder, path.Base(url))

	_, err := os.Stat(rootFsFilename)
	if err == nil {
		fmt.Fprintf(os.Stdout, "Exists: Skipping download of %s\n", url)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Downloading requested rootFS image: %s\n", url)

	resp, err := http.Get(url) // nolint: noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unable to download %s, status %s", url, resp.Status)
	}

	out, err := os.Create(rootFsFilename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(rootFsFilename)
		return err
	}

	fmt.Fprintf(os.Stdout, "Download rootFS complete\n")

	return nil
}

// Once upgraded to GO 1.18, this function can be replaced by slices.Contains.
func contains(stringlist []string, s string) bool {
	for _, item := range stringlist {
		if item == s {
			return true
		}
	}
	return false
}

func writeImageToFile(imageFile *os.File, image string) {
	_, err := imageFile.WriteString(image + "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to image list: %v", err)
		os.Exit(1)
	}
}

func saveToImagesFile(image, imageMapping string, aiImages []string, aiImagesFile, ocpImagesFile *os.File) {
	splittedImageMapping := strings.Split(imageMapping, ":")
	if contains(aiImages, image) || strings.HasPrefix(splittedImageMapping[0], "multicluster-engine/assisted-installer") {
		writeImageToFile(aiImagesFile, image)
		if strings.Contains(splittedImageMapping[0], "assisted-installer-reporter") {
			writeImageToFile(ocpImagesFile, image)
		}
	} else if splittedImageMapping[0] == "openshift/release-images" {
		writeImageToFile(aiImagesFile, image)
		writeImageToFile(ocpImagesFile, image)
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
			writeImageToFile(aiImagesFile, image)
		}
		writeImageToFile(ocpImagesFile, image)
	} else {
		writeImageToFile(ocpImagesFile, image)
	}
}

// filterImage checks an image name and tag against the compiled filter patterns,
// returning true if an image should be ignored.
func filterImage(line, image, imageMapping string) bool {
	for _, re := range ImageFilters {
		if re.MatchString(image) {
			fmt.Fprintf(os.Stdout, "Skipping image due to filter (%s): Matched image name: %s\n", re.String(), line)
			return true
		}
		if re.MatchString(imageMapping) {
			fmt.Fprintf(os.Stdout, "Skipping image due to filter (%s): Matched image tag: %s\n", re.String(), line)
			return true
		}
	}

	return false
}

// parseMappingFile will parse the mapping.txt file returned by the oc-mirror command, returning
// the ImageMapping list of images to be downloaded.
func parseMappingFile(mappingFile, ignoredImagesFile *os.File) (images []ImageMapping) {
	fileScanner := bufio.NewScanner(mappingFile)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		splittedLine := strings.Split(line, "=")
		image := splittedLine[0]
		imageMapping := splittedLine[1]
		splittedImage := strings.Split(image, "/")
		artifact := splittedImage[len(splittedImage)-1]
		artifact = strings.Replace(artifact, ":", "_", 1)

		// Check image against filters before adding to list
		if !filterImage(line, image, imageMapping) {
			images = append(images, ImageMapping{image, imageMapping, artifact})
		} else {
			writeImageToFile(ignoredImagesFile, line)
		}
	}

	return images
}

// handleStaleFiles compares the existing tar files against the ImageMapping list to determine
// whether there are stale files, deleting any that are found.
func handleStaleFiles(folder string, images []ImageMapping) error {
	expectedTarfiles := map[string]bool{}
	for _, image := range images {
		expectedTarfiles[image.Artifact+".tgz"] = true
	}

	files, err := os.ReadDir(folder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to readdir: %s\n", folder)
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tgz") && !expectedTarfiles[file.Name()] {
			fmt.Printf("Deleting stale image: %s\n", file.Name())
			fpath := path.Join(folder, file.Name())
			err = os.Remove(fpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete %s:\n", fpath)
				return err
			}
		}
	}

	return nil
}

func download(folder, release, url string,
	aiImages, additionalImages []string,
	rmStale, generateImageSet, duProfile, skipImageSet bool,
	acmVersion, mceVersion string,
	maxParallel int,
	catalogRedhatOperators, catalogCertifiedOperators string) error {
	tmpDir, err := os.MkdirTemp("", "fp-cli-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to create temporary directory:\n")
		return err
	}
	defer os.RemoveAll(tmpDir)

	imagesetFile := path.Join(folder, "imageset.yaml")

	if !skipImageSet {
		err = templatizeImageset(release, folder, aiImages, additionalImages, duProfile, acmVersion, mceVersion, catalogRedhatOperators, catalogCertifiedOperators)
		if err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", imagesetFile)
		if generateImageSet {
			return nil
		}
	} else {
		_, err = os.Stat(imagesetFile)
		if err != nil {
			return fmt.Errorf("--skip-imageset specified, but %s not found", imagesetFile)
		}
	}

	if url != "" {
		err = downloadRootFsFile(folder, url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to download rootFS image:\n")
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Validating operator versions:\n")
	err = validateVersions(folder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Operator version validation failed:\n")
		return err
	}

	fmt.Fprintf(os.Stdout, "Generating list of pre-cached artifacts...\n")
	err = runOcMirrorCommand(tmpDir, folder)
	if err != nil {
		return err
	}

	cmd := generateMoveMappingFileCommand(tmpDir, folder)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		return err
	}

	mappingFile, err := os.Open(path.Join(folder, "mapping.txt"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to open mapping.txt file:\n")
		return err
	}
	defer mappingFile.Close()

	aiImagesFile, err := os.OpenFile(path.Join(folder, "ai-images.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to open ai-images.txt file:\n")
		return err
	}
	defer aiImagesFile.Close()

	ocpImagesFile, err := os.OpenFile(path.Join(folder, "ocp-images.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open ocp-images.txt file:\n")
		return err
	}
	defer ocpImagesFile.Close()

	ignoredImagesFile, err := os.OpenFile(path.Join(folder, "ignored-images.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to open ignored-images.txt file:\n")
		return err
	}
	defer ignoredImagesFile.Close()

	images := parseMappingFile(mappingFile, ignoredImagesFile)

	if rmStale {
		err = handleStaleFiles(folder, images)
		if err != nil {
			return err
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
		return nil
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
	go imageDownloaderResults(&wgResp, results, totalJobs, aiImages, aiImagesFile, ocpImagesFile)

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
		return fmt.Errorf("%d download failures occurred", downloadFailures)
	}

	return nil
}

func yesOrNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
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
