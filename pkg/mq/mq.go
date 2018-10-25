package mq

import (
	"encoding/json"
	"fmt"
	"github.com/nsqio/go-nsq"
)

type Client struct {
	cfg *nsq.Config
}

func NewClient(lookupds []string) *Client {
	cfg := nsq.NewConfig()
	return &Client{cfg: cfg}
}

func (c *Client) Producer(nsqd string) (*Publisher, error) {
	p, err := nsq.NewProducer(nsqd, c.cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create producer with nsqd=%q: %v", nsqd, err)
	}
	return &Publisher{producer: p}, nil
}

type Publisher struct {
	producer *nsq.Producer
}

func (p *Publisher) Publish(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal data to json: %v", err)
	}
	return p.producer.Publish(topic, b)
}
