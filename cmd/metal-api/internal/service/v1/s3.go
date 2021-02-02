package v1

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	*s3.Client
	Region            string
	Url               string
	Key               string
	Secret            string
	HostnameImmutable bool // https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/#immutable-endpoints
}

func NewS3Client(region, url, key, secret string, hostnameImmutable bool) *S3Client {
	return &S3Client{
		Region:            region,
		Url:               url,
		Key:               key,
		Secret:            secret,
		HostnameImmutable: hostnameImmutable,
	}
}

func (c *S3Client) Connect() error {
	if c.Client != nil {
		return nil
	}

	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if c.Region == region {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               c.Url,
				SigningRegion:     region,
				HostnameImmutable: c.HostnameImmutable,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithEndpointResolver(customResolver))
	if err != nil {
		return err
	}
	cfg.Region = c.Region
	cfg.Credentials = credentials.NewStaticCredentialsProvider(c.Key, c.Secret, "")
	c.Client = s3.NewFromConfig(cfg)
	return nil
}
