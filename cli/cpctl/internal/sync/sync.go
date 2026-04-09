package sync

import (
	"log/slog"
)

func Run(input SyncInput) (*DiffResult, error) {
	slog.Info("🔄 syncing into namespace:", slog.String("namespace", input.Namespace))

	result := &DiffResult{}

	if err := Preflight(input); err != nil {
		return nil, err
	}

	if len(input.Config) == 0 && len(input.Secrets) == 0 {
		slog.Info("⚠️  nothing to sync (config & secrets empty)")
		return nil, nil
	}

	switch input.Only {
	case "config":
		input.Secrets = nil
	case "secrets":
		input.Config = nil
	}

	if err := EnsureNamespace(input.Namespace); err != nil {
		return nil, err
	}

	// --------------------
	// ConfigMap
	// --------------------
	if len(input.Config) > 0 {
		rendered, err := RenderConfigMapYAML(input.Namespace, input.Config)
		if err != nil {
			return nil, err
		}

		if input.Diff {
			live, err := BuildConfigMapViewFromCluster(input.Namespace, "birdy-config")
			if err != nil {
				slog.Info("🆕 ConfigMap birdy-config does not exist")
			} else {
				desired := BuildConfigMapViewFromRendered(input.Config)
				diff := DiffConfigMapViews(live, &desired)
				PrintConfigMapDiff("birdy-config", diff)

				if len(diff.Added)+len(diff.Changed)+len(diff.Removed) > 0 {
					result.HasChanges = true
				}
			}
		} else if input.DryRun {
			slog.Info("🧪 dry-run: ConfigMap birdy-config")
			slog.Info(rendered)
		} else {
			if err := ApplyConfigMap(input); err != nil {
				return nil, err
			}
		}
	}

	// --------------------
	// Secrets
	// --------------------
	if len(input.Secrets) > 0 {

		if input.Diff {
			live, err := BuildSecretViewFromCluster(input.Namespace, "birdy-secrets")
			if err != nil {
				slog.Info("🆕 Secret birdy-secrets does not exist")
			} else {
				desired := BuildSecretViewFromRendered(input.Secrets)
				diff := DiffSecretViews(live, &desired)
				PrintSecretDiff("birdy-secrets", diff)

				if len(diff.Added)+len(diff.Changed)+len(diff.Removed) > 0 {
					result.HasChanges = true
				}
			}
		} else if input.DryRun {
			slog.Info("🧪 dry-run: secrets present (values hidden)")
		} else {
			if err := ApplySecrets(input.Namespace, input.Secrets); err != nil {
				return nil, err
			}
		}
	}

	slog.Info("✅ sync completed")
	return result, nil
}
