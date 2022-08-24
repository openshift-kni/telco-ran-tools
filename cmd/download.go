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

// TODO: Add SHAs for 2.6 if we support it as ACM/AI
const ACM25AssistedInstallerAgentSHA = "sha256:482618e19dc48990bb53f46e441ce21f574c04a6e0b9ee8fe1103284e15db994"
const ACM25AssistedInstallerSHA = "sha256:e0dbc04261a5f9d946d05ea40117b6c3ab33c000ae256e062f7c3e38cdf116cc"

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Downloads and pre-caches artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		folder, _ := cmd.Flags().GetString("folder")
		release, _ := cmd.Flags().GetString("release")
		url, _ := cmd.Flags().GetString("rootfs-url")
		download(folder, release, url)
	},
}

func init() {
	downloadCmd.Flags().StringP("folder", "f", "", "Folder to download artifacts")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("release", "r", "", "OpenShift release version")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("rootfs-url", "u", "", "rootFS URL")
	rootCmd.AddCommand(downloadCmd)
}

type ImageSet struct {
	Channel                   string
	Version                   string
	AssistedInstallerAgentSHA string
	AssistedInstallerSHA      string
}

func generateOcMirrorCommand(tmpDir, folder string) *exec.Cmd {
	return exec.Command("oc-mirror", "-c", folder+"/imageset.yaml", "file://"+tmpDir+"/mirror", "--ignore-history", "--dry-run")
}

func generateSkopeoCopyCommand(folder, artifact, artifactsFile string) *exec.Cmd {
	return exec.Command("skopeo", "copy", "docker://"+artifactsFile, "dir://"+folder+"/"+artifact, "-q", "--retry-times", "10")
}

func generateTarArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("tar", "czvf", folder+"/"+artifact+".tgz", "-C", folder, artifact)
}

func generateRemoveArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("rm", "-rf", folder+"/"+artifact)
}

func generateMoveMappingFileCommand(tmpFolder, folder string) *exec.Cmd {
	return exec.Command("cp", tmpFolder+"/mirror/oc-mirror-workspace/mapping.txt", folder+"/mapping.txt")
}

func templatizeImageset(release, folder string) {
	t, err := template.New("ImageSet").Parse(imageSetTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse template: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(folder + "/imageset.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse imageset.yaml file: %v\n", err)
		os.Exit(1)
	}

	// TODO: Validate the release version is valid with something like oc-mirror list releases --channel=stable-<channel
	r := strings.Split(release, ".")
	channel := r[0] + "." + r[1]
	version := release

	// If we support ACM 2.6, then there should be logic to add ACM as a param to the CLI and then a map to hace ACM to AI SHAs
	d := ImageSet{channel, version, ACM25AssistedInstallerAgentSHA, ACM25AssistedInstallerSHA}
	err = t.Execute(f, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to execute template: %v\n", err)
		os.Exit(1)
	}
}

func downloadRootFsFile(release, folder, url string) error {
	r := strings.Split(release, ".")
	channel := r[0] + "." + r[1]

	out, err := os.Create(folder + "/rhcos-live-rootfs.x86_64.img")
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

func saveToImagesFile(image string, imageMapping string, aiImagesFile *os.File, ocpImagesFile *os.File) {
	splittedImageMapping := strings.Split(imageMapping, ":")
	if strings.HasPrefix(splittedImageMapping[0], "multicluster-engine") {
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

func download(folder, release, url string) {
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
		_, err = os.Stat(folder + rootFsFilename)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Downloading release rootFS image %s...\n", url)
			err = downloadRootFsFile(release, folder, url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unable to download rootFS image: %v\n", err)
				os.Exit(1)
			}
		}
	}

	templatizeImageset(release, folder)

	_, err = os.Stat(folder + "/mapping.txt")
	if err != nil {
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
	}

	mappingFile, err := os.Open(folder + "/mapping.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open mapping.txt file: %v\n", err)
		os.Exit(1)
	}
	defer mappingFile.Close()

	aiImagesFile, err := os.OpenFile(folder+"/ai-images.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open ai-images.txt file: %v\n", err)
		os.Exit(1)
	}
	defer aiImagesFile.Close()

	ocpImagesFile, err := os.OpenFile(folder+"/ocp-images.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open ocp-images.txt file: %v\n", err)
		os.Exit(1)
	}
	defer ocpImagesFile.Close()

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
		fmt.Fprintf(os.Stdout, "Processing artifact %s\n", artifact)
		_, err = os.Stat(folder + "/" + artifact + ".tgz")
		if err == nil {
			// File exists, thus it's been already downloaded and tarballed, moving on...
			fmt.Fprintf(os.Stdout, "File %s.tgz already exists, skipping...\n", artifact)
			continue
		}
		saveToImagesFile(image, imageMapping, aiImagesFile, ocpImagesFile)
		cmd := generateSkopeoCopyCommand(folder, artifact, image)
		stdout, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
			os.Exit(1)
		}
		cmd = generateTarArtifactCommand(folder, artifact)
		stdout, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
			os.Exit(1)
		}
		cmd = generateRemoveArtifactCommand(folder, artifact)
		stdout, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
			os.Exit(1)
		}
	}
}
