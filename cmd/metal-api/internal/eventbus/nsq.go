package eventbus

import (
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

// nsqdRetryDelay represents the delay that is used for retries in blocking calls.
const nsqdRetryDelay = 3 * time.Second

type PublisherProvider func(*zap.Logger, *bus.PublisherConfig) (bus.Publisher, error)

// NSQClient is a type to request NSQ related tasks such as creation of topics.
type NSQClient struct {
	logger            *zap.Logger
	config            *bus.PublisherConfig
	publisherProvider PublisherProvider
	Publisher         bus.Publisher
	Endpoints         *bus.Endpoints
}

// NewNSQ create a new NSQClient.
func NewNSQ(publisherConfig *bus.PublisherConfig, logger *zap.Logger, publisherProvider PublisherProvider) NSQClient {
	return NSQClient{
		config:            publisherConfig,
		logger:            logger,
		publisherProvider: publisherProvider,
	}
}

//WaitForPublisher blocks until the given provider is able to provide a non nil publisher.
func (n *NSQClient) WaitForPublisher() {
	for {
		publisher, err := n.publisherProvider(n.logger, n.config)
		if err != nil {
			n.logger.Sugar().Errorw("cannot create nsq publisher", "error", err)
			n.delay()
			continue
		}
		n.logger.Sugar().Infow("nsq connected", "nsqd", fmt.Sprintf("%+v", n.config))
		n.Publisher = publisher
		break
	}
}

func (n *NSQClient) CreateEndpoints(lookupds ...string) error {
	c, err := bus.NewConsumer(n.logger, n.config.TLS, lookupds...)
	if err != nil {
		return fmt.Errorf("cannot create consumer for endpoints: %w", err)
	}
	// change loglevel to warning, because nsq is very noisy
	c.With(bus.LogLevel(bus.Warning))
	n.Endpoints = bus.NewEndpoints(c, n.Publisher)
	return nil
}

//WaitForTopicsCreated blocks until the topices are created within the given partitions.
func (n NSQClient) WaitForTopicsCreated(partitions metal.Partitions, topics []metal.NSQTopic) {
	for {
		if err := n.createTopics(partitions, topics); err != nil {
			n.logger.Sugar().Errorw("cannot create topics", "error", err)
			n.delay()
			continue
		}
		break
	}
}

//CreateTopic creates a topic for the given partition.
func (n NSQClient) CreateTopic(partitionID, topicFQN string) error {
	if err := n.Publisher.CreateTopic(topicFQN); err != nil {
		n.logger.Sugar().Errorw("cannot create topic", "topic", topicFQN, "partition", partitionID)
		return err
	}
	n.logger.Sugar().Infow("topic created", "partition", partitionID, "topic", topicFQN)
	return nil
}

func (n NSQClient) createTopics(partitions metal.Partitions, topics []metal.NSQTopic) error {
	for _, partition := range partitions {
		for _, topic := range topics {
			topicFQN := topic.GetFQN(partition.GetID())
			if err := n.CreateTopic(partition.GetID(), topicFQN); err != nil {
				n.logger.Sugar().Errorw("cannot create topics", "partition", partition.GetID())
				return err
			}
		}
	}
	return nil
}

func (n NSQClient) delay() {
	time.Sleep(nsqdRetryDelay)
}
