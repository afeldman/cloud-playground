package exec

import "os/exec"

func Exists(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}
