/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

// partitionCmd represents the partition command
var partitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "Partitions and formats a disk",
	Run: func(cmd *cobra.Command, args []string) {
		device, _ := cmd.Flags().GetString("device")
		size, _ := cmd.Flags().GetInt("size")
		partition(device, size)
	},
	Version: Version,
}

func init() {
	partitionCmd.Flags().StringP("device", "d", "", "Device to be partitioned")
	partitionCmd.MarkFlagRequired("device")
	partitionCmd.Flags().IntP("size", "s", 100, "Partition size in GB")
	rootCmd.AddCommand(partitionCmd)

}

func generatePartitionCommand(device string, size int) *exec.Cmd {
	return exec.Command("sgdisk", "-n", fmt.Sprintf("1:-%dGiB:0", size), device, "-g", "-c:1:data")
}

func generateFormatCommand(device string) *exec.Cmd {
	// if device ends with a number a 'p' is included in the partition name
	val, err := strconv.Atoi(device[len(device)-1:])
	_ = val
	if err != nil {
		fmt.Printf("Partition %s1 is being formatted\n", device)
		return exec.Command("mkfs.xfs", "-f", device+"1")
	} else {
		fmt.Printf("Partition %sp1 is being formatted\n", device)
		return exec.Command("mkfs.xfs", "-f", device+"p1")
	}
}

func partition(device string, size int) {
	cmd := generatePartitionCommand(device, size)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to partition device: %s\n", string(stdout))
		os.Exit(1)
	}

	cmd = generateFormatCommand(device)
	stdout, err = executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to format device: %s\n", string(stdout))
		os.Exit(1)
	}

}
