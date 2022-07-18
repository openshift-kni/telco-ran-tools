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
		partition(device, size)
	},
}

func init() {
	partitionCmd.Flags().StringP("device", "d", "", "Device to be partitioned")
	partitionCmd.MarkFlagRequired("device")
	partitionCmd.Flags().IntP("size", "s", 100, "Partition size in GB")
	rootCmd.AddCommand(partitionCmd)

}

func isPartitionSizeTooBig(deviceSize, desiredSize float64) bool {
	return deviceSize >= desiredSize
}

func partition(device string, size int) {
	cmd := exec.Command("lsblk", device, "-osize", "-dn")
	stdout, err := cmd.Output()

	if err != nil {
		panic(err)
	}

	deviceSizeStr := strings.TrimSpace(string(stdout))
	deviceSizeStr = deviceSizeStr[:len(deviceSizeStr)-1]
	deviceSize, err := strconv.ParseFloat(deviceSizeStr, 64)
	if err != nil {
		panic(err)
	}
	if !isPartitionSizeTooBig(deviceSize, float64(size)) {
		panic(fmt.Errorf("partition size too big"))
	}

	cmd = exec.Command("sgdisk", "-n", fmt.Sprintf("1:-%dGiB:0", size), device, "-g")
	stdout, err = cmd.CombinedOutput()

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	fmt.Println(string(stdout))

	cmd = exec.Command("sgdisk", "-c:1:data", device)
	stdout, err = cmd.CombinedOutput()

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	fmt.Println(string(stdout))

	cmd = exec.Command("mkfs.xfs", device+"1")
	stdout, err = cmd.CombinedOutput()

	if err != nil {
		fmt.Println(string(stdout))
		panic(err)
	}

	fmt.Println(string(stdout))
}
