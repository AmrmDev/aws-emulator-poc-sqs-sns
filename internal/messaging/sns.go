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

	fmt.Println("========== SNS PUBLISHER ==========")
	fmt.Printf("Topic ARN: %s\n", p.topicARN)
	fmt.Printf("Message ID: %s\n", aws.ToString(result.MessageId))
	fmt.Printf("Payload: %s\n", string(payload))
	fmt.Println("Message published successfully!")
	fmt.Println("===================================")

	return nil
}
