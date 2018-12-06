package metal

import "time"

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

type Base struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" rethinkdb:"id,omitempty"`
	Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
	Description string    `json:"description,omitempty" description:"a description for this entity" optional:"true" rethinkdb:"description"`
	Created     time.Time `json:"created" description:"the creation time of this entity" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}
