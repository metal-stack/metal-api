package audit

import "go.uber.org/zap"

type AuditLogOutput struct {
	log *zap.SugaredLogger
}

func NewAuditLogOutput(log *zap.SugaredLogger) AuditOutput {
	return &AuditLogOutput{
		log: log,
	}
}

func (a *AuditLogOutput) Emit(msg AuditMessage) error {
	a.log.Infow("emit audit log", "msg", msg)
	return nil
}
