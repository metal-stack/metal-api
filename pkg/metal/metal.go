package metal

type EventType string

type NSQTopic string

// Some enums.
const (
	CREATE EventType = "create"
	UPDATE EventType = "update"
	DELETE EventType = "delete"

	TopicDevice NSQTopic = "device"
)

var (
	Topics = []NSQTopic{
		TopicDevice,
	}
)
