package s3

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
	Url            string
	Key            string
	Secret         string
	FirmwareBucket string
}

func NewS3Client(url, key, secret, firmwareBucket string) *Client {
	return &Client{
		Url:            url,
		Key:            key,
		Secret:         secret,
		FirmwareBucket: firmwareBucket,
	}
}

func (c *Client) Connect() error {
	if c.S3 != nil {
		return nil
	}
	s, err := c.NewSession()
	if err != nil {
		return err
	}
	c.S3 = s3.New(s)
	return nil
}

func (c *Client) NewSession() (client.ConfigProvider, error) {
	dummyRegion := "dummy" // we don't use AWS S3, we don't need a proper region
	hostnameImmutable := true
	return session.NewSession(&aws.Config{
		Region:           &dummyRegion,
		Endpoint:         &c.Url,
		Credentials:      credentials.NewStaticCredentials(c.Key, c.Secret, ""),
		S3ForcePathStyle: &hostnameImmutable,
		SleepDelay:       time.Sleep,
		Retryer: client.DefaultRetryer{
			NumMaxRetries: 3,
			MinRetryDelay: 10 * time.Second,
		},
	})
}
