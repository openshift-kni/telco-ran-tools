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
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

// partitionCmd represents the partition command.
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

// nolint: errcheck
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
	part := device + "1"

	// if device ends with a number a 'p' is included in the partition name
	_, err := strconv.Atoi(device[len(device)-1:])
	if err == nil {
		part = device + "p1"
	}

	fmt.Printf("Partition %s is being formatted\n", part)
	return exec.Command("mkfs.xfs", "-f", part)

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
