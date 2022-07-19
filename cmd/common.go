package cmd

import "os/exec"

func executeCommand(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}
