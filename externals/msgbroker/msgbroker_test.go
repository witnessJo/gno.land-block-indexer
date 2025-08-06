package msgbroker

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestMsgBroker(t *testing.T) {
	ctx := context.Background()

	// LocalStack 설정 (Docker에서 실행 중)
	config := &LocalStackConfig{
		Endpoint: "http://localhost:4566", // Docker 기본 포트
		Region:   "us-east-1",
	}

	// 메시지 브로커 생성
	broker, err := NewMsgBrokerLocalStack(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create message broker: %v", err)
	}
	defer broker.Close()

	// Topic 이름
	topicName := "test-topic"

	// 구독자 1
	err = broker.Subscribe(topicName, func(message []byte) error {
		fmt.Printf("[Subscriber 1] Received: %s\n", string(message))
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// 구독자 2
	err = broker.Subscribe(topicName, func(message []byte) error {
		fmt.Printf("[Subscriber 2] Received: %s\n", string(message))
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// 메시지 발행
	messages := []string{
		"Hello LocalStack!",
		"Testing SNS-SQS integration",
		"Message broker is working",
	}

	for i, msg := range messages {
		err = broker.Publish(topicName, []byte(msg))
		if err != nil {
			log.Printf("Failed to publish message %d: %v", i+1, err)
		} else {
			log.Printf("Published message %d: %s", i+1, msg)
		}
		time.Sleep(1 * time.Second)
	}

	// 메시지 처리 대기
	time.Sleep(5 * time.Second)

	fmt.Println("Done!")
}
