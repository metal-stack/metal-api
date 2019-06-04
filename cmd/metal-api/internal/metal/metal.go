package metal

import (
	"time"

	"git.f-i-ts.de/cloud-native/metallib/security"
)

// These are our supported groups.
const (
	ViewAccess  = security.RessourceAccess("k8s_kaas-view")
	EditAccess  = security.RessourceAccess("k8s_kaas-edit")
	AdminAccess = security.RessourceAccess("k8s_kaas-admin")
)

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
	TopicSwitch  NSQTopic = "switch"
)

var (
	// Topics is a list of topics of which the metal-api is a producer.
	// metal-api will make sure these topics exist when it is started.
	Topics = []NSQTopic{
		TopicMachine,
		TopicSwitch,
	}

	getNow = time.Now
)

// Base implements common fields for most basic entity types (not all).
type Base struct {
	ID          string    `rethinkdb:"id,omitempty"`
	Name        string    `rethinkdb:"name"`
	Description string    `rethinkdb:"description"`
	Created     time.Time `rethinkdb:"created"`
	Changed     time.Time `rethinkdb:"changed"`
}

// Entity is an interface that allows metal entities to be created and stored into the database with the generic creation and update functions.
type Entity interface {
	// GetID returns the entity's id
	GetID() string
	// SetID sets the entity's id
	SetID(id string)
	// GetChanged returns the entity's changed time
	GetChanged() time.Time
	// SetChanged sets the entity's changed time
	SetChanged(changed time.Time)
	// GetCreated sets the entity's creation time
	GetCreated() time.Time
	// SetCreated sets the entity's creation time
	SetCreated(created time.Time)
}

// GetID returns the ID of the entity
func (b *Base) GetID() string {
	return b.ID
}

// SetID sets the ID of the entity
func (b *Base) SetID(id string) {
	b.ID = id
}

// GetChanged returns the last changed timestamp of the entity
func (b *Base) GetChanged() time.Time {
	return b.Changed
}

// SetChanged sets the last changed timestamp of the entity
func (b *Base) SetChanged(changed time.Time) {
	b.Changed = changed
}

// GetCreated returns the creation timestamp of the entity
func (b *Base) GetCreated() time.Time {
	return b.Created
}

// SetCreated sets the creation timestamp of the entity
func (b *Base) SetCreated(created time.Time) {
	b.Created = created
}
