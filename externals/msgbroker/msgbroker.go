package msgbroker

import "gno.land-block-indexer/model"

type MsgBroker interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string, handler func(message []byte) error) error
	Close() error
}

type BlockWithTransactions struct {
	Block        *model.Block        `json:"block"`
	Transactions []model.Transaction `json:"transactions"`
}
