package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Capture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// stderr anhängen für Debugbarkeit
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunQuiet runs a command without printing output. On failure, stderr is
// included in the returned error.
func RunQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func RunWithStdin(input string, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = bytes.NewBufferString(input)

	return cmd.Run()
}
