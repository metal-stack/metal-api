package metal

import "time"

// EventType is the type for event types.
type EventType string

// NSQTopic .
type NSQTopic string

// Some enums.
const (
	CREATE  EventType = "create"
	UPDATE  EventType = "update"
	DELETE  EventType = "delete"
	COMMAND EventType = "command"

	TopicMachine NSQTopic = "machine"
)

var (
	// Topics is a list of topics of which the metal-api is a producer.
	// metal-api will make sure these topics exist when it is started.
	Topics = []NSQTopic{
		TopicMachine,
	}
)

// Base implements common fields for most basic entity types (not all).
type Base struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" rethinkdb:"id,omitempty"`
	Name        string    `json:"name" description:"the readable name" optional:"true" rethinkdb:"name"`
	Description string    `json:"description,omitempty" description:"a description for this entity" optional:"true" rethinkdb:"description"`
	Created     time.Time `json:"created" description:"the creation time of this entity" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}
