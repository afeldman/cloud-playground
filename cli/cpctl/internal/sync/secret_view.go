package sync

type SecretView struct {
	Name      string
	Namespace string
	Keys      map[string]string // key -> checksum
}
