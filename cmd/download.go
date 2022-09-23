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
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

// Using SHAs for ACM 2.5.1 as defaults
const ACMDefaultAIAgentSHA = "sha256:482618e19dc48990bb53f46e441ce21f574c04a6e0b9ee8fe1103284e15db994"
const ACMDefaultAIInstallerSHA = "sha256:e0dbc04261a5f9d946d05ea40117b6c3ab33c000ae256e062f7c3e38cdf116cc"

// Default AI image locations
const ACMDefaultAIAgentImageLoc = "registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8"
const ACMDefaultAIInstallerImageLoc = "registry.redhat.io/multicluster-engine/assisted-installer-rhel8"
const ACMDefaultAIControllerImageLoc = "registry.redhat.io/multicluster-engine/assisted-reporter-agent-rhel8"

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Downloads and pre-caches artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		folder, _ := cmd.Flags().GetString("folder")
		release, _ := cmd.Flags().GetString("release")
		url, _ := cmd.Flags().GetString("rootfs-url")
		aiInstallerSha, _ := cmd.Flags().GetString("ai-installer-sha")
		aiAgentSha, _ := cmd.Flags().GetString("ai-agent-sha")
		aiControllerSha, _ := cmd.Flags().GetString("ai-controller-sha")
		aiInstallerImage, _ := cmd.Flags().GetString("ai-installer-image")
		aiAgentImage, _ := cmd.Flags().GetString("ai-agent-image")
		aiControllerImage, _ := cmd.Flags().GetString("ai-controller-image")
		additionalImages, _ := cmd.Flags().GetStringSlice("img")
		rmStale, _ := cmd.Flags().GetBool("rm-stale")
		generateImageSet, _ := cmd.Flags().GetBool("generate-imageset")
		skipImageSet, _ := cmd.Flags().GetBool("skip-imageset")
		download(folder, release, url, aiInstallerSha, aiAgentSha, aiControllerSha, aiInstallerImage, aiAgentImage, aiControllerImage, additionalImages,
			rmStale, generateImageSet, skipImageSet)
	},
}

func init() {
	downloadCmd.Flags().StringP("folder", "f", "", "Folder to download artifacts")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("release", "r", "", "OpenShift release version")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("rootfs-url", "u", "", "rootFS URL")
	downloadCmd.Flags().StringP("ai-installer-sha", "i", "", "AI Installer Image SHA (deprecated)")
	downloadCmd.Flags().StringP("ai-agent-sha", "g", "", "AI Agent Image SHA (deprecated)")
	downloadCmd.Flags().StringP("ai-controller-sha", "c", "", "AI Controller Image SHA (deprecated)")
	downloadCmd.Flags().String("ai-installer-image", "", "AI Installer Image")
	downloadCmd.Flags().String("ai-agent-image", "", "AI Agent Image")
	downloadCmd.Flags().String("ai-controller-image", "", "AI Controller Image")
	downloadCmd.Flags().StringSliceP("img", "a", []string{}, "Additional Image(s)")
	downloadCmd.Flags().BoolP("rm-stale", "s", false, "Remove stale images")
	downloadCmd.Flags().Bool("generate-imageset", false, "Generate imageset.yaml only")
	downloadCmd.Flags().Bool("skip-imageset", false, "Skip imageset.yaml generation")
	rootCmd.AddCommand(downloadCmd)
}

type ImageSet struct {
	Channel           string
	Version           string
	AIInstallerImage  string
	AIAgentImage      string
	AIControllerImage string
	AdditionalImages  []string
}

type ImageMapping struct {
	Image        string
	ImageMapping string
	Artifact     string
}

func generateOcMirrorCommand(tmpDir, folder string) *exec.Cmd {
	return exec.Command("oc-mirror", "-c", path.Join(folder, "imageset.yaml"), "file://"+tmpDir+"/mirror", "--ignore-history", "--dry-run")
}

func generateSkopeoCopyCommand(folder, artifact, artifactsFile string) *exec.Cmd {
	return exec.Command("skopeo", "copy", "docker://"+artifactsFile, "dir://"+folder+"/"+artifact, "-q", "--retry-times", "10")
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

func templatizeImageset(release, folder, aiInstallerImage, aiAgentImage, aiControllerImage string, additionalImages []string) {
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

	// If we support ACM 2.6, then there should be logic to add ACM as a param to the CLI and then a map to hace ACM to AI SHAs
	d := ImageSet{channel, version, aiInstallerImage, aiAgentImage, aiControllerImage, additionalImages}
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

func saveToImagesFile(image, imageMapping, aiInstallerImage, aiAgentImage, aiControllerImage string, aiImagesFile *os.File, ocpImagesFile *os.File) {
	splittedImageMapping := strings.Split(imageMapping, ":")
	if strings.HasPrefix(splittedImageMapping[0], "multicluster-engine") || image == aiInstallerImage || image == aiAgentImage || image == aiControllerImage {
		aiImagesFile.WriteString(image + "\n")
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

func download(folder, release, url, aiInstallerSha, aiAgentSha, aiControllerSha, aiInstallerImage, aiAgentImage, aiControllerImage string,
	additionalImages []string,
	rmStale, generateImageSet, skipImageSet bool) {
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

	if aiInstallerImage == "" {
		if aiInstallerSha == "" {
			aiInstallerSha = ACMDefaultAIInstallerSHA
		}
		aiInstallerImage = ACMDefaultAIInstallerImageLoc + "@" + aiInstallerSha
	}

	if aiAgentImage == "" {
		if aiAgentSha == "" {
			aiAgentSha = ACMDefaultAIAgentSHA
		}
		aiAgentImage = ACMDefaultAIAgentImageLoc + "@" + aiAgentSha
	}

	if aiControllerImage == "" && aiControllerSha != "" {
		aiControllerImage = ACMDefaultAIControllerImageLoc + "@" + aiControllerSha
	}

	imagesetFile := path.Join(folder, "imageset.yaml")

	if !skipImageSet {
		templatizeImageset(release, folder, aiInstallerImage, aiAgentImage, aiControllerImage, additionalImages)

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

	for _, image := range images {
		fmt.Fprintf(os.Stdout, "Processing artifact %s\n", image.Artifact)
		artifactTar := image.Artifact + ".tgz"
		_, err = os.Stat(path.Join(folder, artifactTar))
		if err == nil {
			// File exists, thus it's been already downloaded and tarballed, moving on...
			fmt.Fprintf(os.Stdout, "File %s.tgz already exists, skipping...\n", image.Artifact)
		} else {
			scratchdir := path.Join(folder, "scratch")
			_, err = os.Stat(scratchdir)
			if err == nil {
				err = os.RemoveAll(scratchdir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: unable to remove directory %s: %e\n", path.Join(folder, image.Artifact), err)
					os.Exit(1)
				}
			}

			err = os.Mkdir(scratchdir, 0755)

			cmd := generateSkopeoCopyCommand(scratchdir, image.Artifact, image.Image)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: Failed to mkdir %s: %s", scratchdir, err)
				os.Exit(1)
			}

			stdout, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
				os.Exit(1)
			}

			cmd = generateTarArtifactCommand(scratchdir, image.Artifact)
			stdout, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
				os.Exit(1)
			}

			err = os.Rename(path.Join(scratchdir, artifactTar), path.Join(folder, artifactTar))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to move file %s: %e\n", path.Join(scratchdir, artifactTar), err)
				os.Exit(1)
			}

			err = os.RemoveAll(scratchdir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to remove directory %s: %e\n", scratchdir, err)
				os.Exit(1)
			}
		}

		saveToImagesFile(image.Image, image.ImageMapping, aiInstallerImage, aiAgentImage, aiControllerImage, aiImagesFile, ocpImagesFile)
	}
}
