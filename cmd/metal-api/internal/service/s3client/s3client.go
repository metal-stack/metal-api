package s3client

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"time"
)

type Client struct {
	*s3.S3
	Session        client.ConfigProvider
	Url            string
	Key            string
	Secret         string
	FirmwareBucket string
}

func New(url, key, secret, firmwareBucket string) (*Client, error) {
	c := &Client{
		Url:            url,
		Key:            key,
		Secret:         secret,
		FirmwareBucket: firmwareBucket,
	}
	s, err := c.newSession()
	if err != nil {
		return nil, err
	}
	c.S3 = s3.New(s)
	c.Session = s
	return c, nil
}

func (c *Client) newSession() (client.ConfigProvider, error) {
	dummyRegion := "dummy" // we don't use AWS S3, we don't need a proper region
	hostnameImmutable := true
	return session.NewSession(&aws.Config{
		Region:           &dummyRegion,
		Endpoint:         &c.Url,
		Credentials:      credentials.NewStaticCredentials(c.Key, c.Secret, ""),
		S3ForcePathStyle: &hostnameImmutable,
		Retryer: client.DefaultRetryer{
			NumMaxRetries: 3,
			MinRetryDelay: 10 * time.Second,
		},
	})
}
