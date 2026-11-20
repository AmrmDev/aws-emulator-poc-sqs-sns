package config

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	AWS aws.Config

	Endpoint string
	TopicARN string
	QueueURL string
}

func Load() (*Config, error) {
	region := env("AWS_REGION", "us-east-1")
	endpoint := env("AWS_ENDPOINT_URL", "https://127.0.0.1:9297")

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &Config{
		AWS:      awsCfg,
		Endpoint: endpoint,
		TopicARN: os.Getenv("SNS_TOPIC_ARN"),
		QueueURL: os.Getenv("SQS_QUEUE_URL"),
	}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
