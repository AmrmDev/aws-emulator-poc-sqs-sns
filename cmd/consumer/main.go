package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"errors"

	"aws-emulator-poc-sqs-sns/config"
	"aws-emulator-poc-sqs-sns/internal/messaging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.QueueURL == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	consumer := messaging.NewConsumer(
		cfg.AWS,
		cfg.Endpoint,
		cfg.QueueURL,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("SQS CONSUMER STARTED")
	log.Println("")


	if err := consumer.Consume(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Fatalf("Consumer failed: %v", err)
		}
	}	

	log.Println("SQS CONSUMER STOPPED")
}
