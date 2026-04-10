package moto

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

func WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:4566/moto-api/data.json")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				slog.Info("✓ moto ready (health endpoint reachable)")
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return errors.New("moto not ready (timeout)")
}
