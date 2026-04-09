package kind

import "os/exec"

func KubectlAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}
