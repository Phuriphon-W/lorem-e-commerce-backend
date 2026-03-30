package database

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	sdkConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func ConnectS3(s3Endpoint, region, accessKey, secretKey string) (*s3.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	sdkConfig, err := sdkConfig.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(sdkConfig, func(opt *s3.Options) {
		opt.BaseEndpoint = aws.String(s3Endpoint)
		opt.UsePathStyle = true
		opt.Region = region
	})

	log.Println("Created a new S3 clients and connected to", s3Endpoint)

	return client, nil
}
