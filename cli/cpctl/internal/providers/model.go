package providers

type SecretItem struct {
	Name  string
	Value []byte // NUR im Provider, niemals geloggt
}

type SecretProvider interface {
	List() ([]SecretItem, error)
}
