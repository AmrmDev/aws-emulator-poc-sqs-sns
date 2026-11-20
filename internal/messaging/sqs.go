package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Consumer struct {
	client   *sqs.Client
	queueURL string
}

type SNSEnvelope struct {
	Type      string `json:"Type"`
	MessageID string `json:"MessageId"`
	TopicARN  string `json:"TopicArn"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
}

func NewConsumer(cfg aws.Config, endpoint, queueURL string) *Consumer {
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Consumer{
		client:   client,
		queueURL: queueURL,
	}
}

func (c *Consumer) Consume(ctx context.Context) error {
	for {
		result, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			return fmt.Errorf("receive SQS messages: %w", err)
		}

		for _, message := range result.Messages {
			if err := c.process(ctx, message); err != nil {
				fmt.Printf("Processing failed: %v\n", err)
				continue
			}

			_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.queueURL),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("Delete message failed: %v\n", err)
				continue
			}

			fmt.Println("Message deleted successfully!")
			fmt.Println("===================================")
		}
	}
}

func (c *Consumer) process(ctx context.Context, message types.Message) error {
	if message.Body == nil || *message.Body == "" {
		return fmt.Errorf("empty message body")
	}

	var envelope SNSEnvelope

	if err := json.Unmarshal([]byte(*message.Body), &envelope); err != nil {
		return fmt.Errorf("invalid SNS envelope: %w", err)
	}

	if envelope.Type != "Notification" {
		return fmt.Errorf("unexpected SNS message type: %s", envelope.Type)
	}

	if envelope.Message == "" {
		return fmt.Errorf("empty SNS message payload")
	}

	fmt.Println("========== SQS CONSUMER ==========")
	fmt.Println("SNS Notification received!")
	fmt.Printf("SNS Message ID: %s\n", envelope.MessageID)
	fmt.Printf("SNS Topic ARN: %s\n", envelope.TopicARN)
	fmt.Printf("SNS Timestamp: %s\n", envelope.Timestamp)

	var event map[string]any

	if err := json.Unmarshal([]byte(envelope.Message), &event); err != nil {
		return fmt.Errorf("invalid payment event JSON: %w", err)
	}

	fmt.Println("-----------------------------------")
	fmt.Println("Payment event processed:")
	fmt.Printf("%+v\n", event)
	fmt.Println("-----------------------------------")

	return nil
}
