package metal

type EventType string

// Some EventType enums.
const (
	CREATE EventType = "create"
	UPDATE EventType = "update"
	DELETE EventType = "delete"
)
