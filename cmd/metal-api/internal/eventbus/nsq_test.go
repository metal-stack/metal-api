package eventbus

import (
	"log/slog"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/stretchr/testify/assert"
)

func TestNewNSQ(t *testing.T) {
	cfg := &bus.PublisherConfig{
		TCPAddress:   "addr",
		HTTPEndpoint: "rest",
	}
	publisher := bus.NewPublisher
	logger := slog.Default()
	actual := NewNSQ(cfg, logger, publisher)

	assert.NotNil(t, actual)
	assert.Equal(t, cfg.TCPAddress, actual.config.TCPAddress)
	assert.Equal(t, cfg.HTTPEndpoint, actual.config.HTTPEndpoint)
	assert.Nil(t, actual.Publisher)
}

func TestNSQ_WaitForPublisher(t *testing.T) {
	cfg := &bus.PublisherConfig{
		TCPAddress:   "addr",
		HTTPEndpoint: "rest",
	}
	publisher := NopPublisher{}

	nsq := NewNSQ(cfg, slog.Default(), func(logger *slog.Logger, config *bus.PublisherConfig) (bus.Publisher, error) {
		assert.Equal(t, cfg.TCPAddress, config.TCPAddress)
		assert.Equal(t, cfg.HTTPEndpoint, config.HTTPEndpoint)
		return publisher, nil
	})
	assert.NotNil(t, nsq)
	assert.Nil(t, nsq.Publisher)

	nsq.WaitForPublisher()
	assert.NotNil(t, nsq.Publisher)
	assert.Equal(t, publisher, nsq.Publisher)
}

func TestNSQ_WaitForTopicsCreated(t *testing.T) {
	topic := metal.NSQTopic{Name: "gopher"}
	partition := metal.Partition{
		Base: metal.Base{ID: "partition-id"},
	}
	publisher := NopPublisher{
		T:     t,
		topic: topic.GetFQN(partition.GetID()),
	}
	nsq := NewNSQ(nil, slog.Default(), func(*slog.Logger, *bus.PublisherConfig) (bus.Publisher, error) {
		return nil, nil
	})
	assert.NotNil(t, nsq)
	nsq.Publisher = publisher

	nsq.WaitForTopicsCreated([]metal.Partition{partition}, []metal.NSQTopic{metal.NSQTopic(topic)})

	// assertions are checked within the NopPublisher stub
}

type NopPublisher struct {
	T     assert.TestingT
	topic string
}

func (p NopPublisher) Publish(topic string, data interface{}) error {
	return nil
}

func (p NopPublisher) CreateTopic(topic string) error {
	assert.Equal(p.T, p.topic, topic)
	return nil
}

func (p NopPublisher) Stop() {}
