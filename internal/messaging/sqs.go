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
			MaxNumberOfMessages: 5,
			WaitTimeSeconds:     10,
		})
		if err != nil {
			return fmt.Errorf("Receive SQS messages: %w", err)
		}

		for _, message := range result.Messages {
			if err := c.process(ctx, message); err != nil {
				fmt.Printf("Processing failed: %v\n", err)
				continue
			}

			_, err = c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.queueURL),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("Delete message failed: %v\n", err)
				continue
			}
		}
	}
}

func (c *Consumer) process(ctx context.Context, message types.Message) error {
	if message.Body == nil {
		return fmt.Errorf("empty message body")
	}

	body := *message.Body

	var envelope SNSEnvelope

	if err := json.Unmarshal([]byte(body), &envelope); err == nil &&
		envelope.Type == "Notification" {

		fmt.Printf("SNS Event received: %s\n", envelope.MessageID)

		body = envelope.Message
	}

	var event map[string]any

	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return fmt.Errorf("invalid event JSON: %w", err)
	}

	fmt.Printf("Processing event: %+v\n", event)
	fmt.Println("===========================")

	return nil
}