package service

import (
	"github.com/graphql-go/graphql"
	"gno.land-block-indexer/cmd/block-synchronizer/model"
)

type Service interface {
	PushBlock(block *model.Block) error
	PushTransaction(transactions *model.Transaction) error
}

type service struct {
	missingIndexes []int
	schema         *graphql.Schema
}

// PushBlock implements Service.
func (s *service) PushBlock(block *model.Block) error {
	panic("unimplemented")
}

// PushTransaction implements Service.
func (s *service) PushTransaction(transactions *model.Transaction) error {
	panic("unimplemented")
}

func NewService(schema *graphql.Schema) Service {
	return &service{
		schema: schema,
	}
}
