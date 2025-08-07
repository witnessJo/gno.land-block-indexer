.PHONY: all clean ent-install ent

all: ent bs-bin ep-bin rest-bin

bs-bin: 
	go build -o bin/block-synchronizer ./cmd/block-synchronizer

ep-bin:
	go build -o bin/event-processor ./cmd/event-processor

rest-bin:
	go build -o bin/indexer-rest ./cmd/indexer-rest

ent-install:
	go install entgo.io/ent/cmd/ent@latest

ent:
	ent generate ./ent/schema

infra:
	./start-infra-compose.sh
