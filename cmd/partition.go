/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// partitionCmd represents the partition command
var partitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		device, _ := cmd.Flags().GetString("device")
		size, _ := cmd.Flags().GetInt("size")
		folder, _ := cmd.Flags().GetString("folder")
		partition(device, size, folder)
	},
}

func init() {
	partitionCmd.Flags().StringP("device", "d", "", "Device to be partitioned")
	partitionCmd.MarkFlagRequired("device")
	partitionCmd.Flags().IntP("size", "s", 100, "Partition size in GB")
	partitionCmd.Flags().StringP("folder", "f", "", "Folder to mount the device")
	partitionCmd.MarkFlagRequired("folder")
	rootCmd.AddCommand(partitionCmd)

}

func isPartitionSizeTooBig(deviceSize, desiredSize float64) bool {
	return desiredSize > deviceSize
}

func generateGetDeviceSizeCommand(device string) *exec.Cmd {
	return exec.Command("lsblk", device, "-osize", "-dn")
}

func generatePartitionCommand(device string, size int) *exec.Cmd {
	return exec.Command("sgdisk", "-n", fmt.Sprintf("1:-%dGiB:0", size), device, "-g", "-c:1:data")
}

func generateFormatCommand(device string) *exec.Cmd {
	return exec.Command("mkfs.xfs", "-f", device+"1")
}

func generateMountCommand(device, folder string) *exec.Cmd {
	return exec.Command("mount", device+"1", folder)
}

func partition(device string, size int, folder string) {
	cmd := generateGetDeviceSizeCommand(device)
	stdout, err := executeCommand(cmd)

	if err != nil {
		panic(err)
	}

	deviceSizeStr := strings.TrimSpace(string(stdout))
	deviceSizeStr = deviceSizeStr[:len(deviceSizeStr)-1]
	deviceSize, err := strconv.ParseFloat(deviceSizeStr, 64)
	if err != nil {
		panic(err)
	}
	if isPartitionSizeTooBig(deviceSize, float64(size)) {
		panic(fmt.Errorf("partition size too big"))
	}

	cmd = generatePartitionCommand(device, size)
	stdout, err = executeCommand(cmd)

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	cmd = generateFormatCommand(device)
	stdout, err = executeCommand(cmd)

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	cmd = generateMountCommand(device, folder)
	stdout, err = executeCommand(cmd)

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}
}
