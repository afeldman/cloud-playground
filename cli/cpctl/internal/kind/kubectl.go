package kind

import (
	"log/slog"
	"os"
	"os/exec"
)

func Kubectl(args ...string) error {
	slog.Info("→ kubectl", slog.Any("args", args))

	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
