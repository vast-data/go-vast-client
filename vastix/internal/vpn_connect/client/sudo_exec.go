package client

import (
	"io"
	"os/exec"
	"strings"
)

func runSudo(sudoPassword string, writer io.Writer, args ...string) error {
	cmd := exec.Command("sudo", append([]string{"-S"}, args...)...)
	cmd.Stdin = strings.NewReader(sudoPassword + "\n")
	if writer != nil {
		cmd.Stdout = writer
		cmd.Stderr = writer
	}
	return cmd.Run()
}

func runSudoScript(sudoPassword string, writer io.Writer, script string) error {
	bash := platformBash()
	cmd := exec.Command("sudo", "-S", bash, "-c", script)
	cmd.Stdin = strings.NewReader(sudoPassword + "\n")
	if writer != nil {
		cmd.Stdout = writer
		cmd.Stderr = writer
	}
	return cmd.Run()
}

func runWgQuick(sudoPassword string, writer io.Writer, action, target string) error {
	args := wgQuickSudoArgs(action, target)
	return runSudo(sudoPassword, writer, args...)
}
