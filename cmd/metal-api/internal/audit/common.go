package audit

type AuditOutput interface {
	Emit(msg AuditMessage) error
}

type AuditMessage struct {
	Path   []string
	Method string
	Code   int

	Request  []byte
	Response []byte

	User   string
	EMail  string
	Tenant string

	ScopeJSON string

	Summary string
}
