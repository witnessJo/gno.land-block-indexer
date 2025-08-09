package msgbroker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"gno.land-block-indexer/lib/log"
)

// Config for LocalStack connection
type LocalStackConfig struct {
	Endpoint string // LocalStack endpoint (default: "http://localhost:4566")
	Region   string // AWS Region (default: "us-east-1")
}

type msgBrokerLocalstack struct {
	ctx    context.Context
	logger log.Logger
	cancel context.CancelFunc
	config LocalStackConfig

	sqsClient *sqs.Client
	snsClient *sns.Client

	mu          sync.RWMutex
	topicArns   map[string]string
	queueUrls   map[string]string
	subscribers map[string][]func(message []byte) error
	pollersWg   sync.WaitGroup
}

// NewMsgBrokerLocalStack creates a new message broker using LocalStack running in Docker
func NewMsgBrokerLocalStack(ctx context.Context, logger log.Logger, cfg *LocalStackConfig) (MsgBroker, error) {
	// Default configuration
	if cfg == nil {
		cfg = &LocalStackConfig{
			Endpoint: "http://localhost:4566",
			Region:   "us-east-1",
		}
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:4566"
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	// AWS SDK configuration for LocalStack
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		},
	)

	// Load AWS config with LocalStack endpoint
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	broker := &msgBrokerLocalstack{
		ctx:         ctx,
		logger:      logger,
		cancel:      cancel,
		config:      *cfg,
		sqsClient:   sqs.NewFromConfig(awsCfg),
		snsClient:   sns.NewFromConfig(awsCfg),
		topicArns:   make(map[string]string),
		queueUrls:   make(map[string]string),
		subscribers: make(map[string][]func(message []byte) error),
	}

	// Test connection
	if err := broker.testConnection(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to LocalStack: %w", err)
	}

	return broker, nil
}

// testConnection verifies LocalStack is accessible
func (m *msgBrokerLocalstack) testConnection() error {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	// Try to list topics to verify SNS is accessible
	_, err := m.snsClient.ListTopics(ctx, &sns.ListTopicsInput{})
	if err != nil {
		return fmt.Errorf("SNS service not accessible: %w", err)
	}

	// Try to list queues to verify SQS is accessible
	_, err = m.sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return fmt.Errorf("SQS service not accessible: %w", err)
	}

	log.Infof("Successfully connected to LocalStack at %s", m.config.Endpoint)
	return nil
}

// Publish implements MsgBroker.
func (m *msgBrokerLocalstack) Publish(topic string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Topic이 없으면 생성
	topicArn, exists := m.topicArns[topic]
	if !exists {
		var err error
		topicArn, err = m.createTopic(topic)
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic, err)
		}
		m.topicArns[topic] = topicArn
		log.Infof("Created SNS topic: %s with ARN: %s", topic, topicArn)
	}

	// 메시지 발행
	output, err := m.snsClient.Publish(m.ctx, &sns.PublishInput{
		TopicArn: aws.String(topicArn),
		Message:  aws.String(string(message)),
	})
	if err != nil {
		return fmt.Errorf("failed to publish message to topic %s: %w", topic, err)
	}

	log.Infof("Published message to topic %s, MessageId: %s", topic, *output.MessageId)
	return nil
}

// Subscribe implements MsgBroker.
func (m *msgBrokerLocalstack) Subscribe(topic string, handler func(message []byte) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Topic이 없으면 생성
	topicArn, exists := m.topicArns[topic]
	if !exists {
		var err error
		topicArn, err = m.createTopic(topic)
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic, err)
		}
		m.topicArns[topic] = topicArn
		log.Infof("Created SNS topic: %s with ARN: %s", topic, topicArn)
	}

	// Queue 생성 (각 구독자마다 고유한 queue)
	queueName := fmt.Sprintf("%s-queue-%d", topic, time.Now().UnixNano())
	queueUrl, err := m.createQueue(queueName)
	if err != nil {
		return fmt.Errorf("failed to create queue for topic %s: %w", topic, err)
	}
	m.queueUrls[queueName] = queueUrl
	log.Infof("Created SQS queue: %s with URL: %s", queueName, queueUrl)

	// Topic을 Queue에 구독
	if err := m.subscribeQueueToTopic(topicArn, queueUrl); err != nil {
		return fmt.Errorf("failed to subscribe queue to topic %s: %w", topic, err)
	}
	log.Infof("Subscribed queue %s to topic %s", queueName, topic)

	// 핸들러 등록
	m.subscribers[topic] = append(m.subscribers[topic], handler)

	// 각 구독자마다 별도의 폴러 시작
	m.pollersWg.Add(1)
	go m.pollMessages(topic, queueUrl, handler)

	return nil
}

func (m *msgBrokerLocalstack) createTopic(topicName string) (string, error) {
	output, err := m.snsClient.CreateTopic(m.ctx, &sns.CreateTopicInput{
		Name: aws.String(topicName),
		Attributes: map[string]string{
			"DisplayName": topicName,
		},
	})
	if err != nil {
		return "", err
	}
	return *output.TopicArn, nil
}

func (m *msgBrokerLocalstack) createQueue(queueName string) (string, error) {
	output, err := m.sqsClient.CreateQueue(m.ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]string{
			"MessageRetentionPeriod":        "3600", // 1 hour
			"VisibilityTimeout":             "30",   // 30 seconds
			"ReceiveMessageWaitTimeSeconds": "20",   // Long polling
		},
	})
	if err != nil {
		return "", err
	}
	return *output.QueueUrl, nil
}

func (m *msgBrokerLocalstack) subscribeQueueToTopic(topicArn, queueUrl string) error {
	// Queue ARN 가져오기
	queueAttrs, err := m.sqsClient.GetQueueAttributes(m.ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueUrl),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return fmt.Errorf("failed to get queue attributes: %w", err)
	}

	queueArn := queueAttrs.Attributes[string(types.QueueAttributeNameQueueArn)]

	// SQS Queue 정책 설정 (SNS가 메시지를 전달할 수 있도록)
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {
				"Service": "sns.amazonaws.com"
			},
			"Action": "sqs:SendMessage",
			"Resource": "%s",
			"Condition": {
				"ArnEquals": {
					"aws:SourceArn": "%s"
				}
			}
		}]
	}`, queueArn, topicArn)

	_, err = m.sqsClient.SetQueueAttributes(m.ctx, &sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(queueUrl),
		Attributes: map[string]string{
			string(types.QueueAttributeNamePolicy): policy,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set queue policy: %w", err)
	}

	// SNS Topic을 SQS Queue에 구독
	subscribeOutput, err := m.snsClient.Subscribe(m.ctx, &sns.SubscribeInput{
		TopicArn: aws.String(topicArn),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(queueArn),
		Attributes: map[string]string{
			"RawMessageDelivery": "false", // SNS 메타데이터 포함
		},
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	log.Infof("Created subscription: %s", *subscribeOutput.SubscriptionArn)
	return nil
}

func (m *msgBrokerLocalstack) pollMessages(topic, queueUrl string, handler func(message []byte) error) {
	defer m.pollersWg.Done()
	log.Infof("Starting message poller for topic: %s, queue: %s", topic, queueUrl)

	for {
		select {
		case <-m.ctx.Done():
			log.Infof("Stopping message poller for topic: %s", topic)
			return
		default:
			// SQS에서 메시지 수신 (Long Polling 사용)
			output, err := m.sqsClient.ReceiveMessage(m.ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueUrl),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20, // Long polling
				VisibilityTimeout:   30,
			})
			if err != nil {
				if m.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Infof("Error receiving messages from queue %s: %v", queueUrl, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(output.Messages) > 0 {
				log.Infof("Received %d messages from queue for topic %s", len(output.Messages), topic)
			}

			for _, msg := range output.Messages {
				m.processMessage(topic, queueUrl, msg, handler)
			}
		}
	}
}

func (m *msgBrokerLocalstack) processMessage(topic, queueUrl string, msg types.Message, handler func(message []byte) error) {
	// SNS 메시지는 JSON으로 래핑되어 있음
	var snsMessage struct {
		Type             string `json:"Type"`
		MessageId        string `json:"MessageId"`
		TopicArn         string `json:"TopicArn"`
		Message          string `json:"Message"`
		Timestamp        string `json:"Timestamp"`
		SignatureVersion string `json:"SignatureVersion"`
		Signature        string `json:"Signature"`
		SigningCertURL   string `json:"SigningCertURL"`
		UnsubscribeURL   string `json:"UnsubscribeURL"`
	}

	messageBody := ""
	if err := json.Unmarshal([]byte(*msg.Body), &snsMessage); err != nil {
		// SNS 메시지가 아닌 경우 직접 사용
		log.Infof("Message is not SNS formatted, using raw body")
		messageBody = *msg.Body
	} else {
		messageBody = snsMessage.Message
		log.Infof("Processing SNS message: Type=%s, MessageId=%s", snsMessage.Type, snsMessage.MessageId)
	}

	// 핸들러 실행
	if err := handler([]byte(messageBody)); err != nil {
		log.Infof("Error handling message for topic %s: %v", topic, err)
		// 에러 발생 시 메시지를 삭제하지 않고 리턴 (재시도를 위해)
		return
	}

	// 성공적으로 처리된 메시지 삭제
	_, err := m.sqsClient.DeleteMessage(m.ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueUrl),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Infof("Error deleting message from queue %s: %v", queueUrl, err)
	} else {
		log.Infof("Successfully processed and deleted message for topic %s", topic)
	}
}

// Close gracefully shuts down the message broker
func (m *msgBrokerLocalstack) Close() error {
	log.Infof("Closing message broker...")

	// Cancel context to stop all pollers
	m.cancel()

	// Wait for all pollers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.pollersWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Infof("All message pollers stopped successfully")
	case <-time.After(30 * time.Second):
		log.Infof("Timeout waiting for pollers to stop")
	}

	// Clean up resources (optional: delete queues and topics)
	// This is commented out as you might want to keep them for debugging
	// m.cleanup()

	log.Infof("Message broker closed")
	return nil
}

// Optional: cleanup method to remove created resources
func (m *msgBrokerLocalstack) cleanup() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Delete all created queues
	for queueName, queueUrl := range m.queueUrls {
		if _, err := m.sqsClient.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{
			QueueUrl: aws.String(queueUrl),
		}); err != nil {
			log.Infof("Failed to delete queue %s: %v", queueName, err)
		} else {
			log.Infof("Deleted queue: %s", queueName)
		}
	}

	// Delete all created topics
	for topicName, topicArn := range m.topicArns {
		if _, err := m.snsClient.DeleteTopic(context.Background(), &sns.DeleteTopicInput{
			TopicArn: aws.String(topicArn),
		}); err != nil {
			log.Infof("Failed to delete topic %s: %v", topicName, err)
		} else {
			log.Infof("Deleted topic: %s", topicName)
		}
	}
}
