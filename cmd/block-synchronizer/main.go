package main

import (
	"gno.land-block-indexer/cmd/block-synchronizer/controller"
)

func main() {
	// run block-synchronization
	controller := controller.NewController()
	controller.Run()
}
