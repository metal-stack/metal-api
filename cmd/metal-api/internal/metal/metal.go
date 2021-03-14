package metal

import (
	"fmt"
	"time"

	"github.com/metal-stack/metal-lib/jwt/sec"

	"github.com/metal-stack/security"
)

// These are our supported groups.
var (
	// View Groupname
	ViewGroups = []security.ResourceAccess{
		security.ResourceAccess("k8s_kaas-view"), // FIXME remove legacy, only for compatibility
		security.ResourceAccess("maas-all-all-view"),
	}

	// Edit Groupname
	EditGroups = []security.ResourceAccess{
		security.ResourceAccess("k8s_kaas-edit"), // FIXME remove legacy, only for compatibility
		security.ResourceAccess("maas-all-all-edit"),
	}

	// Admin Groupname
	AdminGroups = []security.ResourceAccess{
		security.ResourceAccess("k8s_kaas-admin"), // FIXME remove legacy, only for compatibility
		security.ResourceAccess("maas-all-all-admin"),
	}

	// Groups that have view permission
	ViewAccess = sec.MergeResourceAccess(ViewGroups, EditGroups, AdminGroups)
	// Groups that have edit permission
	EditAccess = sec.MergeResourceAccess(EditGroups, AdminGroups)
	// Groups that have admin permission
	AdminAccess = AdminGroups
)

// EventType is the type for event types.
type EventType string

// NSQTopic .
type NSQTopic struct {
	Name              string
	PartitionAgnostic bool
}

// Some enums.
const (
	CREATE  EventType = "create"
	UPDATE  EventType = "update"
	DELETE  EventType = "delete"
	COMMAND EventType = "command"
)

var (
	TopicMachine    = NSQTopic{Name: "machine", PartitionAgnostic: true}
	TopicAllocation = NSQTopic{Name: "allocation", PartitionAgnostic: false}
)

// Topics is a list of topics of which the metal-api is a producer.
// metal-api will make sure these topics exist when it is started.
var Topics = []NSQTopic{
	TopicMachine,
	TopicAllocation,
}

// GetFQN gets the fully qualified name of a NSQTopic
func (t NSQTopic) GetFQN(partitionID string) string {
	if !t.PartitionAgnostic {
		return t.Name
	}
	return fmt.Sprintf("%s-%s", partitionID, t.Name)
}

// Base implements common fields for most basic entity types (not all).
type Base struct {
	ID          string    `rethinkdb:"id,omitempty" json:"id,omitempty"`
	Name        string    `rethinkdb:"name" json:"name"`
	Description string    `rethinkdb:"description" json:"description"`
	Created     time.Time `rethinkdb:"created" json:"created"`
	Changed     time.Time `rethinkdb:"changed" json:"changed"`
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
