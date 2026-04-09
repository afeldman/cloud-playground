package aws

import (
	"log/slog"
	"os"
)

func Run(profile, ssmPath string) error {

	slog.Info(
		"mirroring aws config",
		"profile", profile,
		"path", ssmPath,
	)

	// Set AWS profile explicitly before SDK calls
	if profile != "" {
		os.Setenv("AWS_PROFILE", profile)
	}

	// Verify AWS login / SSO session
	if err := CheckLogin(profile); err != nil {
		return err
	}
	slog.Info("✅ AWS login verified")

	// Fetch parameters from SSM
	params, err := fetchSSMParams(ssmPath)
	if err != nil {
		return err
	}

	// Render sanitized Kubernetes Secret
	if err := writeSecretYAML(
		"sanitized/secrets",
		"services",
		"birdy-secrets",
		params,
	); err != nil {
		return err
	}

	slog.Info("✅ mirror completed")
	return nil
}
