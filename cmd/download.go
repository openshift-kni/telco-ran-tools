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
	Release string
}

func generateOcMirrorCommand(tmpDir string) *exec.Cmd {
	return exec.Command("./oc-mirror", "-c", "imageset.yaml", "file://"+tmpDir+"/mirror", "--ignore-history", "--dry-run")
}

func generateCreateArtifactsCommand(tmpDir string) *exec.Cmd {
	artifactsCmd := "cat " + tmpDir + "/mirror/oc-mirror-workspace/mapping.txt | cut -d \"=\" -f1 > " + tmpDir + "/artifacts.txt"
	return exec.Command("bash", "-c", artifactsCmd)
}

func templatizeImageset(release, tmpDir string) {
	t, err := template.New("ImageSet").Parse(imageSetTemplate)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(tmpDir + "/imageset.yaml")
	if err != nil {
		panic(err)
	}

	d := ImageSet{release}
	err = t.Execute(f, d)
	if err != nil {
		panic(err)
	}
}

func download(folder, release string) {
	tmpDir, err := ioutil.TempDir("", "fp-cli-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	templatizeImageset(release, tmpDir)

	cmd := generateOcMirrorCommand(tmpDir)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	cmd = generateCreateArtifactsCommand(tmpDir)
	stdout, err = executeCommand(cmd)
	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	readFile, err := os.Open(tmpDir + "/artifacts.txt")
	if err != nil {
		panic(err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		splittedArtifact := strings.Split(line, "/")
		artifact := splittedArtifact[len(splittedArtifact)-1]
		artifact = strings.Replace(artifact, ":", "_", 1)
		cmd = exec.Command("skopeo", "copy", "docker://"+line, "dir://"+folder+"/"+artifact, "-q")
		stdout, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(stdout))
			panic(err)
		}
		cmd = exec.Command("tar", "czvf", folder+"/"+artifact+".tgz", folder+"/"+artifact)
		stdout, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(stdout))
			panic(err)
		}
		cmd = exec.Command("rm", "-rf", folder+"/"+artifact)
		stdout, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(stdout))
			panic(err)
		}
	}
}
