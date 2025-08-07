package main

import (
	"context"

	"gno.land-block-indexer/cmd/block-synchronizer/controller"
)

func main() {
	// run block-synchronization
	ctx := context.Background()
	controller := controller.NewController()
	controller.Run(ctx)
}
