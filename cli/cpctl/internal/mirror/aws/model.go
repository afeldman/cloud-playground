package aws

// Parameter represents a mirrored AWS parameter
// in a filesystem-friendly format.
type Parameter struct {
	Key   string
	Value string
}
