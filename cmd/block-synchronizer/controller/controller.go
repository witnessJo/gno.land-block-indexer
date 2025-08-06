package controller

import "gno.land-block-indexer/cmd/block-synchronizer/service"

type Controller struct {
	service service.Service
}

func NewController() *Controller {
	return &Controller{}
}
