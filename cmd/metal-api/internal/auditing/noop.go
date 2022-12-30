package auditing

type noop struct {
}

// Index implements Auditing
func (*noop) Index(...any) error {
	return nil
}
