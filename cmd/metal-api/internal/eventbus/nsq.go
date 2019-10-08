package eventbus

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"go.uber.org/zap"
)

// NSQRetryDelay represents the delay that is used for retries in blocking calls.
const NSQRetryDelay = 3 * time.Second

type Publisher func(*zap.Logger, string, string) (bus.Publisher, error)

type NSQDConfig struct {
	TCPAddress     string
	RESTEndpoint   string
	CACertFile     string
	ClientCertFile string
}

// NSQClient is a type to request NSQ related tasks such as creation of topics.
type NSQClient struct {
	logger    *zap.Logger
	config    *NSQDConfig
	Publisher bus.Publisher
	publisher Publisher
}

// NewNSQ create a new NSQClient.
func NewNSQ(nsqdAddress, nsqdRESTEndpoint string, logger *zap.Logger, publisher Publisher) NSQClient {
	return NewTLSNSQ(&NSQDConfig{
		TCPAddress:   nsqdAddress,
		RESTEndpoint: nsqdRESTEndpoint,
	}, logger, publisher)
}

// NewNSQ create a new NSQClient.
func NewTLSNSQ(config *NSQDConfig, logger *zap.Logger, publisher Publisher) NSQClient {
	return NSQClient{
		config:    config,
		logger:    logger,
		publisher: publisher,
	}
}

//WaitForPublisher blocks until the given provider is able to provide a non nil publisher.
func (n *NSQClient) WaitForPublisher() {
	for {
		publisher, err := n.publisher(n.logger, n.config.TCPAddress, n.config.RESTEndpoint)
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
	time.Sleep(NSQRetryDelay)
}
