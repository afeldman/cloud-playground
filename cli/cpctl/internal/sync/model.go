package sync

type SyncInput struct {
	Namespace string
	Config    map[string]string
	Secrets   map[string]string

	Only   string
	DryRun bool
	Diff   bool
}
