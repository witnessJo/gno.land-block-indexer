package controller

import (
	"github.com/graphql-go/graphql"
	"gno.land-block-indexer/cmd/block-synchronizer/service"
	"gno.land-block-indexer/repository"
)

type Controller struct {
	schema  *graphql.Schema
	service service.Service
}

type ControllerConfig struct {
	Schema     *graphql.Schema
	repoConfig *repository.RepositoryEntConfig
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Run() error {
	// TODO: Implement controller logic
	// check missing blocks and transactions

	return nil
}
