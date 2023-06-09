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
	"regexp"

	"github.com/spf13/cobra"
)

// partitionCmd represents the partition command.
var partitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "Partitions and formats a disk",
	Run: func(cmd *cobra.Command, args []string) {
		device, _ := cmd.Flags().GetString("device")
		label, _ := cmd.Flags().GetString("label")
		num, _ := cmd.Flags().GetInt("num")
		size, _ := cmd.Flags().GetInt("size")
		partition(device, label, num, size)
	},
	Version: Version,
}

// nolint: errcheck
func init() {
	partitionCmd.Flags().StringP("device", "d", "", "Device to be partitioned")
	partitionCmd.MarkFlagRequired("device")
	partitionCmd.Flags().StringP("label", "l", "data", "Partition label")
	partitionCmd.Flags().IntP("num", "n", 1, "Partition number")
	partitionCmd.Flags().IntP("size", "s", 100, "Partition size in GB")
	rootCmd.AddCommand(partitionCmd)

}

func generatePartitionCommand(device, label string, num, size int) *exec.Cmd {
	return exec.Command("sgdisk", "-n", fmt.Sprintf("%d:-%dGiB:0", num, size), device, "-g", fmt.Sprintf("-c:%d:%s", num, label))
}

func generateFormatCommand(device string, num int) *exec.Cmd {
	var part string
	pattern := regexp.MustCompile(`[0-9]$`)

	if pattern.MatchString(device) {
		// if device ends with a number a 'p' is included in the partition name
		part = fmt.Sprintf("%sp%d", device, num)
	} else {
		part = fmt.Sprintf("%s%d", device, num)
	}

	fmt.Printf("Partition %s is being formatted\n", part)
	return exec.Command("mkfs.xfs", "-f", part)

}

func partition(device, label string, num, size int) {
	cmd := generatePartitionCommand(device, label, num, size)
	stdout, err := executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to partition device: %s\n", string(stdout))
		os.Exit(1)
	}

	cmd = generateFormatCommand(device, num)
	stdout, err = executeCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to format device: %s\n", string(stdout))
		os.Exit(1)
	}

}
