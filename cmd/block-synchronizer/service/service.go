package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/machinebox/graphql"

	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

type Service interface {
	GetMissingBlocks() ([]model.Block, error)
	GetBlocksMissingTxCount() ([]model.Block, error)
	PushBlock(block *model.Block) error
	PushTransaction(transactions *model.Transaction) error
	PollBlocks(offset int, limit int) ([]model.Block, error)
	PollTransactions(blockOffset int, limit int) ([]model.Transaction, error)
}

type service struct {
	repo   repository.Repository
	client *http.Client
}

type ServiceConfig struct {
	entConfig repository.RepositoryEntConfig
}

func NewService(config *ServiceConfig) Service {
	repo := repository.NewRepositoryEnt(
		&config.entConfig,
	)

	return &service{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// PollBlocks implements Service.
func (s *service) PollBlocks(offset int, limit int) ([]model.Block, error) {
	client := graphql.NewClient("https://indexer.onbloc.xyz/graphql/query")

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

	if err := client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}

	// 변환
	blocks := make([]model.Block, 0, len(resp.GetBlocks))
	for _, b := range resp.GetBlocks {
		// fmt.Printf("Block: %s, Height: %d, Time: %s, TotalTxs: %d, NumTxs: %d\n",
		// b.Hash, b.Height, b.Time, b.TotalTxs, b.NumTxs)

		parsedTime, err := time.Parse(time.RFC3339, b.Time)
		if err != nil {
			log.Printf("failed to parse time: %v", err)
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
	client := graphql.NewClient("https://indexer.onbloc.xyz/graphql/query")

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
				Route   string      `json:"route"`
				TypeUrl string      `json:"typeUrl"`
				Value   interface{} `json:"value"` // 다양한 타입을 처리하기 위해 interface{} 사용
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

	if err := client.Run(ctx, req, &resp); err != nil {
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

// GetMissingBlocks implements Service.
func (s *service) GetMissingBlocks() ([]model.Block, error) {
	ctx := context.Background()
	blocks, err := s.repo.GetBlocks(ctx, 0, 100)
	if err != nil {
		log.Printf("failed to get missing blocks: %v", err)
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
		log.Printf("failed to get blocks missing tx count: %v", err)
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

// PushBlock implements Service.
func (s *service) PushBlock(block *model.Block) error {
	ctx := context.Background()
	err := s.repo.AddBlock(ctx, block)
	if err != nil {
		log.Printf("failed to push block: %v", err)
		return err
	}
	return nil
}

// PushTransaction implements Service.
func (s *service) PushTransaction(transaction *model.Transaction) error {
	ctx := context.Background()
	err := s.repo.AddTransaction(ctx, transaction.BlockHeight, transaction)
	if err != nil {
		log.Printf("failed to push transaction: %v", err)
		return err
	}
	return nil
}
