/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		folder, _ := cmd.Flags().GetString("folder")
		release, _ := cmd.Flags().GetString("release")
		download(folder, release)
	},
}

func init() {
	downloadCmd.Flags().StringP("folder", "f", "", "Folder to download artifacts")
	downloadCmd.MarkFlagRequired("folder")
	downloadCmd.Flags().StringP("release", "r", "", "OpenShift release version")
	downloadCmd.MarkFlagRequired("folder")
	rootCmd.AddCommand(downloadCmd)
}

type ImageSet struct {
	Channel string
	Version string
}

func generateOcMirrorCommand(tmpDir string) *exec.Cmd {
	return exec.Command("oc-mirror", "-c", tmpDir+"/imageset.yaml", "file://"+tmpDir+"/mirror", "--ignore-history", "--dry-run")
}

func generateCreateArtifactsCommand(tmpDir string) *exec.Cmd {
	artifactsCmd := "cat " + tmpDir + "/mirror/oc-mirror-workspace/mapping.txt | cut -d \"=\" -f1 > " + tmpDir + "/artifacts.txt"
	return exec.Command("bash", "-c", artifactsCmd)
}

func generateSkopeoCopyCommand(folder, artifact, artifactsFile string) *exec.Cmd {
	return exec.Command("skopeo", "copy", "docker://"+artifactsFile, "dir://"+folder+"/"+artifact, "-q")
}

func generateTarArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("tar", "czvf", folder+"/"+artifact+".tgz", folder+"/"+artifact)
}

func generateRemoveArtifactCommand(folder, artifact string) *exec.Cmd {
	return exec.Command("rm", "-rf", folder+"/"+artifact)
}

func templatizeImageset(release, tmpDir string) {
	t, err := template.New("ImageSet").Parse(imageSetTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse template: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(tmpDir + "/imageset.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse imageset.yaml file: %v\n", err)
		os.Exit(1)
	}

	r := strings.Split(release, ".")
	channel := r[0] + "." + r[1]
	version := channel + "." + "0"

	if len(r) == 3 {
		version = release
	}

	d := ImageSet{channel, version}
	err = t.Execute(f, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to execute template: %v\n", err)
		os.Exit(1)
	}
}

func download(folder, release string) {
	tmpDir, err := ioutil.TempDir("", "fp-cli-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to create temporary directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	templatizeImageset(release, tmpDir)

	cmd := generateOcMirrorCommand(tmpDir)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		os.Exit(1)
	}

	cmd = generateCreateArtifactsCommand(tmpDir)
	stdout, err = executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to run command %s: %s\n", strings.Join(cmd.Args, " "), string(stdout))
		os.Exit(1)
	}

	readFile, err := os.Open(tmpDir + "/artifacts.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to open artifacts.txt file: %v\n", err)
		os.Exit(1)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		splittedArtifact := strings.Split(line, "/")
		artifact := splittedArtifact[len(splittedArtifact)-1]
		artifact = strings.Replace(artifact, ":", "_", 1)
		cmd = generateSkopeoCopyCommand(folder, artifact, line)
		fmt.Println(cmd.Args)
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
