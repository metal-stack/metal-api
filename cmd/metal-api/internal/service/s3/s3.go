package s3

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"time"
)

type Client struct {
	*s3.Client
	Url    string
	Key    string
	Secret string
}

func NewS3Client(url, key, secret string) *Client {
	return &Client{
		Url:    url,
		Key:    key,
		Secret: secret,
	}
}

func (c *Client) Connect() error {
	if c.Client != nil {
		return nil
	}

	dummyRegion := "dummy" // we don't use AWS S3, we don't need a proper region
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:       "aws",
			URL:               c.Url,
			SigningRegion:     dummyRegion,
			HostnameImmutable: true,
		}, nil
	})
	retryer := func() aws.Retryer {
		r := retry.AddWithMaxAttempts(retry.NewStandard(), 3)
		r = retry.AddWithMaxBackoffDelay(r, 10*time.Second)
		return r
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithEndpointResolver(customResolver), config.WithRetryer(retryer))
	if err != nil {
		return err
	}
	cfg.Region = dummyRegion
	cfg.Credentials = credentials.NewStaticCredentialsProvider(c.Key, c.Secret, "")
	c.Client = s3.NewFromConfig(cfg)
	return nil
}
