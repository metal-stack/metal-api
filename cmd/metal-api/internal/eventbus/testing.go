package eventbus

import (
	"testing"
)

type noopPublisher struct{}

func (n *noopPublisher) Publish(topic string, data interface{}) error {
	return nil
}
func (n *noopPublisher) CreateTopic(topic string) error {
	return nil
}
func (n *noopPublisher) Stop() {
}

func InitTestPublisher(t *testing.T) *NSQClient {
	pub := &noopPublisher{}
	nsq := &NSQClient{
		Publisher: pub,
	}
	return nsq
}
