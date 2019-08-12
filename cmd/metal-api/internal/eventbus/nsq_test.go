package eventbus

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestNewNSQ(t *testing.T) {
	nsqAddr := "addr"
	nsqRESTAddr := "rest"
	provider := bus.NewPublisher
	logger := zap.NewNop()

	actual := NewNSQ(nsqAddr, nsqRESTAddr, logger, provider)

	assert := assert.New(t)
	assert.NotNil(actual)
	assert.Equal(nsqAddr, actual.nsqAddress)
	assert.Equal(nsqRESTAddr, actual.nsqRESTEndpoint)
	assert.Nil(actual.Publisher)
}

func TestNSQ_WaitForPublisher(t *testing.T) {
	nsqAddr := "addr"
	nsqRESTAddr := "rest"
	publisher := NopPublisher{}
	assert := assert.New(t)

	nsq := NewNSQ(nsqAddr, nsqRESTAddr, zap.NewNop(), func(logger *zap.Logger, s string,
		s2 string) (bus.Publisher, error) {
		assert.Equal(nsqAddr, s)
		assert.Equal(nsqRESTAddr, s2)
		return publisher, nil
	})
	assert.NotNil(nsq)
	assert.Nil(nsq.Publisher)

	nsq.WaitForPublisher()
	assert.NotNil(nsq.Publisher)
	assert.Equal(publisher, nsq.Publisher)
}

func TestNSQ_WaitForTopicsCreated(t *testing.T) {
	assert := assert.New(t)
	topic := metal.NSQTopic("gopher")
	partition := metal.Partition{
		Base: metal.Base{ID: "partition-id"},
	}
	publisher := NopPublisher{
		T: t,
		topic: topic.GetFQN(partition.GetID()),
	}
	nsq := NewNSQ("", "", zap.NewNop(), func(*zap.Logger, string, string) (bus.Publisher, error) {
		return nil, nil
	})
	assert.NotNil(nsq)
	nsq.Publisher = publisher

	nsq.WaitForTopicsCreated([]metal.Partition{partition}, []metal.NSQTopic{metal.NSQTopic(topic)})

	// assertions are checked within the NopPublisher stub
}

type NopPublisher struct {
	T assert.TestingT
	topic string
}

func (p NopPublisher) Publish(topic string, data interface{}) error {
	return nil
}

func (p NopPublisher) CreateTopic(topic string) error {
	assert.Equal(p.T, p.topic, topic)
	return nil
}