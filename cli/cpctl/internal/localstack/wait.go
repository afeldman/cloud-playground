package localstack

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

func WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:4566/_localstack/health")
		if err == nil && resp.StatusCode == 200 {
			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
				resp.Body.Close()
				slog.Info("✓ localstack ready (health endpoint reachable)")
				return nil
			}
			resp.Body.Close()
		}

		time.Sleep(2 * time.Second)
	}

	return errors.New("localstack not ready (timeout)")
}
