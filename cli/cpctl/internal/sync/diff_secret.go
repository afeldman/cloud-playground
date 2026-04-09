package sync

import (
	"log/slog"
	"sort"
)

type SecretDiff struct {
	Added   []string
	Removed []string
	Changed []string
	Same    []string
}

func DiffSecretViews(live, desired *SecretView) SecretDiff {
	diff := SecretDiff{}
	seen := map[string]bool{}

	for key, liveSum := range live.Keys {
		seen[key] = true

		desiredSum, ok := desired.Keys[key]
		if !ok {
			diff.Removed = append(diff.Removed, key)
			continue
		}

		if liveSum != desiredSum {
			diff.Changed = append(diff.Changed, key)
		} else {
			diff.Same = append(diff.Same, key)
		}
	}

	for key := range desired.Keys {
		if !seen[key] {
			diff.Added = append(diff.Added, key)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Changed)
	sort.Strings(diff.Same)

	return diff
}

func PrintSecretDiff(name string, diff SecretDiff) {
	slog.Info("🔐 Secret", slog.String("name", name))

	if len(diff.Added) == 0 &&
		len(diff.Removed) == 0 &&
		len(diff.Changed) == 0 {
		slog.Info("  ✅ no changes")
		return
	}

	for _, k := range diff.Added {
		slog.Info("  + " + k)
	}
	for _, k := range diff.Changed {
		slog.Info("  ~ " + k)
	}
	for _, k := range diff.Removed {
		slog.Info("  - " + k)
	}

}
