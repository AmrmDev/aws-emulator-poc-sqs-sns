package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type Publisher struct {
	client   *sns.Client
	topicARN string
}

func NewPublisher(cfg aws.Config, endpoint, topicARN string) *Publisher {
	client := sns.NewFromConfig(cfg, func(o *sns.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Publisher{
		client:   client,
		topicARN: topicARN,
	}
}

func (p *Publisher) Publish(ctx context.Context, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	result, err := p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicARN),
		Message:  aws.String(string(payload)),
	})
	if err != nil {
		return fmt.Errorf("publish SNS message: %w", err)
	}

	fmt.Printf("Message published! ID: %s\n", aws.ToString(result.MessageId))

	return nil
}
