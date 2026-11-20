package main

import (
	"context"
	"log"
	"time"

	"aws-emulator-poc-sqs-sns/config"
	"aws-emulator-poc-sqs-sns/internal/messaging"
)

type PaymentEvent struct {
	PaymentID string    `json:"paymentId"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Method    string    `json:"method"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.TopicARN == "" {
		log.Fatal("SNS_TOPIC_ARN is required")
	}

	publisher := messaging.NewPublisher(
		cfg.AWS,
		cfg.Endpoint,
		cfg.TopicARN,
	)

	event := PaymentEvent{
		PaymentID: "pay-go-001",
		Amount:    150.50,
		Currency:  "BRL",
		Method:    "PIX",
		Status:    "PENDING",
		CreatedAt: time.Now().UTC(),
	}

	if err := publisher.Publish(context.Background(), event); err != nil {
		log.Fatal(err)
	}
}
