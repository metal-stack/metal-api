package eventbus

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
)

// nsqdRetryDelay represents the delay that is used for retries in blocking calls.
const nsqdRetryDelay = 3 * time.Second

type PublisherProvider func(*slog.Logger, *bus.PublisherConfig) (bus.Publisher, error)

// NSQClient is a type to request NSQ related tasks such as creation of topics.
type NSQClient struct {
	logger            *slog.Logger
	config            *bus.PublisherConfig
	publisherProvider PublisherProvider
	Publisher         bus.Publisher
	Endpoints         *bus.Endpoints
}

// NewNSQ create a new NSQClient.
func NewNSQ(publisherConfig *bus.PublisherConfig, logger *slog.Logger, publisherProvider PublisherProvider) NSQClient {
	return NSQClient{
		config:            publisherConfig,
		logger:            logger,
		publisherProvider: publisherProvider,
	}
}

// WaitForPublisher blocks until the given provider is able to provide a non nil publisher.
func (n *NSQClient) WaitForPublisher() {
	for {
		publisher, err := n.publisherProvider(n.logger, n.config)
		if err != nil {
			n.logger.Error("cannot create nsq publisher", "error", err)
			n.delay()
			continue
		}
		n.logger.Info("nsq connected", "nsqd", fmt.Sprintf("%+v", n.config))
		n.Publisher = publisher
		break
	}
}

func (n *NSQClient) CreateEndpoints(lookupds ...string) error {
	c, err := bus.NewConsumer(n.logger, n.config.NSQ, lookupds...)
	if err != nil {
		return fmt.Errorf("cannot create consumer for endpoints: %w", err)
	}
	// change loglevel to warning, because nsq is very noisy
	c.With(bus.LogLevel(bus.Warning))
	n.Endpoints = bus.NewEndpoints(c, n.Publisher)
	return nil
}

// WaitForTopicsCreated blocks until the topices are created within the given partitions.
func (n *NSQClient) WaitForTopicsCreated(partitions metal.Partitions, topics []metal.NSQTopic) {
	for {
		if err := n.createTopics(partitions, topics); err != nil {
			n.logger.Error("cannot create topics", "error", err)
			n.delay()
			continue
		}
		break
	}
}

// CreateTopic creates a topic with given name.
func (n *NSQClient) CreateTopic(name string) error {
	if err := n.Publisher.CreateTopic(name); err != nil {
		n.logger.Error("cannot create topic", "topic", name)
		return err
	}
	n.logger.Info("topic created", "topic", name)
	return nil
}

func (n *NSQClient) createTopics(partitions metal.Partitions, topics []metal.NSQTopic) error {
	for _, topic := range topics {
		if topic.PartitionAgnostic {
			continue
		}
		if err := n.CreateTopic(topic.Name); err != nil {
			n.logger.Error("cannot create topic", "topic", topic.Name)
			return err
		}
	}

	for _, partition := range partitions {
		for _, topic := range topics {
			if !topic.PartitionAgnostic {
				continue
			}
			topicFQN := topic.GetFQN(partition.GetID())
			if err := n.CreateTopic(topicFQN); err != nil {
				n.logger.Error("cannot create topic", "topic", topicFQN, "partition", partition.GetID())
				return err
			}
		}
	}
	return nil
}

func (n *NSQClient) delay() {
	time.Sleep(nsqdRetryDelay)
}
