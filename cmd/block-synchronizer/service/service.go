package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/machinebox/graphql"
	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

const (
	topicBlockWithTransactions = "block_with_transactions"
)

type Service interface {
	// usecases (from controller)
	RestoreMissingBlockAndTransactions(ctx context.Context, blockHeight int) error
	SubscribeAndPush() error

	// repository operations
	GetMissingBlocks() ([]model.Block, error)
	GetBlocksMissingTxCount() ([]model.Block, error)
	GetHighestBlock() (*model.Block, error)

	// poll operations (graphql)
	PollBlocks(offset int, limit int) ([]model.Block, error)
	PollTransactions(blockOffset int, limit int) ([]model.Transaction, error)
	SubscribeLastestBlock(ctx context.Context, ch chan<- model.Block) error
}

type service struct {
	logger            log.Logger
	repo              repository.Repository
	localStack        msgbroker.MsgBroker
	client            *graphql.Client
	websocketEndpoint string
}

type ServiceConfig struct {
	FetchEndpoint     string
	WebSocketEndpoint string
	EntConfig         *repository.RepositoryEntConfig
	LocalStackConfig  *msgbroker.LocalStackConfig
}

func NewService(ctx context.Context, logger log.Logger, config *ServiceConfig) Service {
	repo := repository.NewRepositoryEnt(
		config.EntConfig,
	)
	client := graphql.NewClient(config.FetchEndpoint, graphql.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))

	// Set default endpoints if not provided
	localStack, err := msgbroker.NewMsgBrokerLocalStack(ctx, config.LocalStackConfig)
	if err != nil {
		log.Fatalf("failed to create local stack: %v", err)
	}

	fetchEndpoint := config.FetchEndpoint
	if fetchEndpoint == "" {
		fetchEndpoint = "https://indexer.onbloc.xyz/graphql/query"
	}

	websocketEndpoint := config.WebSocketEndpoint
	if websocketEndpoint == "" {
		websocketEndpoint = "wss://indexer.onbloc.xyz/graphql/query"
	}

	return &service{
		logger:            logger,
		repo:              repo,
		client:            client,
		websocketEndpoint: websocketEndpoint,
		localStack:        localStack,
	}
}

// RestoreMissingBlockAndTransactions implements Service.
func (s *service) RestoreMissingBlockAndTransactions(ctx context.Context, blockHeight int) error {
	panic("unimplemented")
}

// SubscribeAndPush implements Service.
func (s *service) SubscribeAndPush() error {
	ctx := context.Background()
	ch := make(chan model.Block, 100)

	// Start the subscription in a goroutine
	go func() {
		if err := s.SubscribeLastestBlock(ctx, ch); err != nil {
			s.logger.Infof("failed to subscribe to latest block: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("context done, stopping subscription")
			close(ch)
			return ctx.Err()
		case block, ok := <-ch:
			if !ok {
				s.logger.Infof("channel closed, stopping subscription")
				return nil
			}

			go func() {
				transactions, err := s.PollTransactions(block.Height, 1) // Poll transactions for the block
				if err != nil {
					s.logger.Infof("failed to poll transactions for block %d: %v", block.Height, err)
					return
				}
				blockMessage := msgbroker.BlockWithTransactions{
					Block:        &block,
					Transactions: transactions,
				}
				msgBytes, err := json.Marshal(blockMessage) // Ensure the blockMessage is marshaled to JSON
				if err != nil {
					s.logger.Infof("failed to marshal block with transactions: %v", err)
					return
				}

				if err := s.localStack.Publish(topicBlockWithTransactions, msgBytes); err != nil {
					s.logger.Infof("failed to publish block with transactions: %v", err)
					return
				}
			}()
		}
	}
}

// GetHighestBlock implements Service.
func (s *service) GetHighestBlock() (*model.Block, error) {
	ctx := context.Background()
	blocks, err := s.repo.GetBlocks(ctx, 0, 1) // Get only the highest block
	if err != nil {
		s.logger.Infof("failed to get highest block: %v", err)
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("no blocks found in database")
	}

	return blocks[0], nil
}

// PollBlocks implements Service.
func (s *service) PollBlocks(offset int, limit int) ([]model.Block, error) {
	// GraphQL 쿼리
	req := graphql.NewRequest(fmt.Sprintf(`
        query {
            getBlocks(
                where: {
                    height: {
                        gt: %d,
                        lt: %d
                    }
                }
            ) {
                hash
                height
                time
                total_txs
                num_txs
            }
        }
    `, offset, offset+limit))

	// 응답 구조체
	var resp struct {
		GetBlocks []struct {
			Hash     string `json:"hash"`
			Height   int    `json:"height"`
			Time     string `json:"time"`
			TotalTxs int    `json:"total_txs"`
			NumTxs   int    `json:"num_txs"`
		} `json:"getBlocks"`
	}

	// 요청 실행
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}

	// 변환
	blocks := make([]model.Block, 0, len(resp.GetBlocks))
	for _, b := range resp.GetBlocks {
		// fmt.Printf("Block: %s, Height: %d, Time: %s, TotalTxs: %d, NumTxs: %d\n",
		// b.Hash, b.Height, b.Time, b.TotalTxs, b.NumTxs)

		parsedTime, err := time.Parse(time.RFC3339, b.Time)
		if err != nil {
			s.logger.Infof("failed to parse time %s: %v", b.Time, err)
			parsedTime = time.Time{}
		}

		blocks = append(blocks, model.Block{
			Hash:     b.Hash,
			Height:   b.Height,
			Time:     parsedTime,
			TotalTxs: b.TotalTxs,
			NumTxs:   b.NumTxs,
		})
	}

	return blocks, nil
}

// PollTransactions implements Service.
func (s *service) PollTransactions(blockOffset int, limit int) ([]model.Transaction, error) {
	// GraphQL 쿼리 - inline fragments 사용
	req := graphql.NewRequest(fmt.Sprintf(`
        query {
              getTransactions(
                where: {
                  block_height: {
                    gt: %d,
                    lt: %d
                  }
                }
              ) {
                index
                hash
                success
                block_height
                gas_wanted
                gas_used
                memo
                gas_fee {
                  amount
                  denom
                }
                messages {
                  route
                  typeUrl
                  value {
                    ... on BankMsgSend {
                      from_address
                      to_address
                      amount
                    }
                    ... on MsgAddPackage {
                      creator
                      deposit
                      package {
                        name
                        path
                        files {
                          name
                          body
                        }
                      }
                    }
                    ... on MsgCall {
                      pkg_path
                      func
                      send
                      caller
                      args
                    }
                    ... on MsgRun {
                      caller
                      send
                      package {
                        name
                        path
                        files {
                          name
                          body
                        }
                      }
                    }
                  }
                }
                response {
                  log
                  info
                  error
                  data
                  events {
                    ... on GnoEvent {
                      type
                      func
                      pkg_path
                      attrs {
                        key
                        value
                      }
                    }
                  }
                }
              }
		}`, blockOffset, blockOffset+limit))

	// 응답 구조체 - interface{} 사용으로 유연하게 처리
	// {
	//    "index": 0,
	//    "hash": "z/NdyOYfV1ux1H1yw4cuVvntYjOy8IxefIWx5DAPbRo=",
	//    "success": true,
	//    "block_height": 45405,
	//    "gas_wanted": 100000,
	//    "gas_used": 45061,
	//    "memo": "",
	//    "gas_fee": {
	//      "amount": 1000000,
	//      "denom": "ugnot"
	//    },
	//    "messages": [
	//      {
	//        "route": "bank",
	//        "typeUrl": "send",
	//        "value": {
	//          "from_address": "g1qhuef2450xh7g7na8s865nreu2xw8j84kgkvt5",
	//          "to_address": "g189falrnshcsk7z7xc2wdwef6684ngtaf6k45lv",
	//          "amount": "1000000ugnot"
	//        }
	//      }
	//    ],
	//    "response": {
	//      "log": "msg:0,success:true,log:,events:[]",
	//      "info": "",
	//      "error": "",
	//      "data": "",
	//      "events": []
	//    }
	//  }
	var resp struct {
		GetTransactions []struct {
			Index       int          `json:"index"`
			Hash        string       `json:"hash"`
			Success     bool         `json:"success"`
			BlockHeight int          `json:"block_height"`
			GasWanted   float64      `json:"gas_wanted"`
			GasUsed     float64      `json:"gas_used"`
			Memo        string       `json:"memo"`
			GasFee      model.GasFee `json:"gas_fee"`
			Messages    []struct {
				Route   string `json:"route"`
				TypeUrl string `json:"typeUrl"`
				Value   any    `json:"value"` // 다양한 타입을 처리하기 위해 interface{} 사용
			} `json:"messages"`
			Response struct {
				Log    string `json:"log"`
				Info   string `json:"info"`
				Error  string `json:"error"`
				Data   string `json:"data"`
				Events []struct {
					Type    string `json:"type"`
					Func    string `json:"func"`
					PkgPath string `json:"pkg_path"`
					Attrs   []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"attrs"`
				} `json:"events"`
			} `json:"response"`
		} `json:"getTransactions"`
	}

	// 요청 실행
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}

	// 변환
	transactions := make([]model.Transaction, 0, len(resp.GetTransactions))
	for _, t := range resp.GetTransactions {
		// GasFee 변환
		gasFee := model.GasFee{
			Amount: t.GasFee.Amount,
			Denom:  t.GasFee.Denom,
		}

		// Messages 변환 - Value를 JSON 문자열로 저장
		messages := make([]model.Message, len(t.Messages))
		for i, msg := range t.Messages {
			valueJSON, _ := json.Marshal(msg.Value)
			messages[i] = model.Message{
				Route:   msg.Route,
				TypeUrl: msg.TypeUrl,
				Value:   string(valueJSON), // JSON 문자열로 저장
			}
		}

		// Events를 JSON 문자열 배열로 변환
		events := make([]string, len(t.Response.Events))
		for i, event := range t.Response.Events {
			eventJSON, _ := json.Marshal(event)
			events[i] = string(eventJSON)
		}

		// Response 변환
		response := model.Response{
			Log:    t.Response.Log,
			Info:   t.Response.Info,
			Error:  t.Response.Error,
			Data:   t.Response.Data,
			Events: events,
		}

		transactions = append(transactions, model.Transaction{
			Index:       t.Index,
			Hash:        t.Hash,
			Success:     t.Success,
			BlockHeight: t.BlockHeight,
			GasWanted:   t.GasWanted,
			GasUsed:     t.GasUsed,
			Memo:        t.Memo,
			GasFee:      gasFee,
			Messages:    messages,
			Response:    response,
		})
	}

	return transactions, nil
}

func (s *service) SubscribeLastestBlock(ctx context.Context, ch chan<- model.Block) error {
	fmt.Println("Connecting to WebSocket endpoint:", s.websocketEndpoint)
	u, err := url.Parse(s.websocketEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Use graphql-transport-ws protocol (Apollo's newer protocol)
	dialer := websocket.Dialer{
		Subprotocols: []string{"graphql-transport-ws"},
	}

	// Add headers to match browser behavior
	header := http.Header{}
	header.Add("Origin", "https://indexer.onbloc.xyz")
	header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	ws, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer ws.Close()

	// Check which protocol was accepted
	actualProtocol := resp.Header.Get("Sec-WebSocket-Protocol")
	s.logger.Infof("WebSocket connection established with protocol: %s", actualProtocol)

	// Send connection init message (graphql-transport-ws doesn't require payload)
	initMsg := map[string]any{
		"type": "connection_init",
	}

	if err := ws.WriteJSON(initMsg); err != nil {
		return fmt.Errorf("failed to send connection init: %w", err)
	}
	s.logger.Infof("Sent connection_init message")

	// Wait for connection_ack
	var ackMsg map[string]any
	if err := ws.ReadJSON(&ackMsg); err != nil {
		return fmt.Errorf("failed to read connection ack: %w", err)
	}

	if ackMsg["type"] != "connection_ack" {
		return fmt.Errorf("expected connection_ack, got %v", ackMsg["type"])
	}
	s.logger.Infof("Received connection_ack message")

	// Generate unique subscription ID
	subscriptionID := uuid.New().String()

	// GraphQL subscription query (exact format from your working example)
	subscription := `subscription{
  getBlocks(
    where: {}
  ) {
    hash
    height
    time
    total_txs
    num_txs
  }
}`

	// Build subscription message for graphql-transport-ws
	subscribeMsg := map[string]any{
		"id":   subscriptionID,
		"type": "subscribe", // graphql-transport-ws uses "subscribe"
		"payload": map[string]any{
			"query": subscription,
		},
	}

	data, _ := json.MarshalIndent(subscribeMsg, "", "  ")
	s.logger.Infof("Sending subscription message: %s", string(data))

	if err := ws.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}
	s.logger.Infof("Sent subscription message with ID: %s", subscriptionID)
	// Statistics tracking
	startTime := time.Now()
	messageCount := 0
	lastMessageTime := time.Now()

	// Handle messages in a loop
	for {
		select {
		case <-ctx.Done():
			// Send complete message before closing
			completeMsg := map[string]any{
				"id":   subscriptionID,
				"type": "complete",
			}
			ws.WriteJSON(completeMsg)
			return ctx.Err()

		default:
			// Set a longer read deadline to handle slow block generation
			ws.SetReadDeadline(time.Now().Add(120 * time.Second))

			var msg map[string]any
			if err := ws.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return nil
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					timeSinceLastMessage := time.Since(lastMessageTime)
					s.logger.Infof("Read timeout after %v, total time: %v, message count: %d",
						timeSinceLastMessage, time.Since(startTime), messageCount)
					// Don't return on timeout, just continue waiting
					continue
				}
				return fmt.Errorf("failed to read WebSocket message: %w", err)
			}

			messageCount++
			lastMessageTime = time.Now()

			// Log full message for debugging
			msgBytes, _ := json.Marshal(msg)
			s.logger.Infof("Received message: %s", string(msgBytes))

			// Handle different message types
			msgType, ok := msg["type"].(string)
			if !ok {
				s.logger.Infof("Received message without type: %v", msg)
				continue
			}

			switch msgType {
			case "ping":
				// Respond to ping with pong
				pongMsg := map[string]any{"type": "pong"}
				if err := ws.WriteJSON(pongMsg); err != nil {
					return s.logger.Errorf("failed to send pong: %v", err)
				}
				s.logger.Infof("Received ping, sent pong")
				continue

			case "next": // graphql-transport-ws uses "next" for data
				payload, ok := msg["payload"].(map[string]any)
				if !ok {
					s.logger.Infof("Received next message without payload: %v", msg)
					continue
				}

				// Check for errors first
				if errors, ok := payload["errors"]; ok {
					s.logger.Infof("Received errors in next message: %v", errors)
					continue
				}

				data, ok := payload["data"].(map[string]any)
				if !ok {
					s.logger.Infof("Received next message without data: %v", payload)
					continue
				}

				// getBlocks should be a single object
				getBlocks, ok := data["getBlocks"].(map[string]any)
				if !ok {
					s.logger.Infof("Received next message without getBlocks: %v", data)
					continue
				}

				// Parse time
				timeStr, ok := getBlocks["time"].(string)
				var parsedTime time.Time
				if ok {
					parsedTime, err = time.Parse(time.RFC3339, timeStr)
					if err != nil {
						// Try other time formats
						parsedTime, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
						if err != nil {
							s.logger.Infof("Failed to parse time %s: %v", timeStr, err)
							parsedTime = time.Time{} // Use zero value if parsing fails
						} else {
							s.logger.Infof("Parsed time: %v", parsedTime)
						}
					}
				}

				block := model.Block{
					Hash:     getString(getBlocks, "hash"),
					Height:   getInt(getBlocks, "height"),
					Time:     parsedTime,
					TotalTxs: getInt(getBlocks, "total_txs"),
					NumTxs:   getInt(getBlocks, "num_txs"),
				}

				// Send block to channel
				select {
				case ch <- block:
					s.logger.Infof("Received block: Height=%d, Hash=%s, Time=%s",
						block.Height, block.Hash, block.Time)
					s.logger.Infof("Total messages received: %d, Time elapsed: %v",
						messageCount, time.Since(startTime))
				case <-ctx.Done():
					return ctx.Err()
				default:
					s.logger.Infof("Channel is full, dropping block: Height=%d, Hash=%s, Time=%s",
						block.Height, block.Hash, block.Time)
				}

			case "error":
				s.logger.Errorf("Received error message: %v", msg)
				errPayload := msg["payload"]
				if id, ok := msg["id"]; ok && id == subscriptionID {
					return fmt.Errorf("subscription error: %v", errPayload)
				}

			case "complete":
				s.logger.Infof("Subscription complete for ID: %s", subscriptionID)
				return nil

			default:
				s.logger.Infof("Received unknown message type: %s, content: %v", msgType, msg)
			}
		}
	}
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

// GetMissingBlocks implements Service.
func (s *service) GetMissingBlocks() ([]model.Block, error) {
	ctx := context.Background()
	blocks, err := s.repo.GetBlocks(ctx, 0, 100)
	if err != nil {
		s.logger.Infof("failed to get blocks: %v", err)
		return nil, err
	}

	result := make([]model.Block, len(blocks))
	for i, block := range blocks {
		result[i] = *block
	}
	return result, nil
}

// GetBlocksMissingTxCount implements Service.
func (s *service) GetBlocksMissingTxCount() ([]model.Block, error) {
	ctx := context.Background()
	blocks, err := s.repo.GetBlocks(ctx, 0, 100)
	if err != nil {
		s.logger.Infof("failed to get blocks: %v", err)
		return nil, err
	}

	var missingTxBlocks []model.Block
	for _, block := range blocks {
		if block.NumTxs == 0 && block.TotalTxs > 0 {
			missingTxBlocks = append(missingTxBlocks, *block)
		}
	}
	return missingTxBlocks, nil
}
