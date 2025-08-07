.PHONY: all clean ent-install ent

all: ent-install ent

ent-install:
	go install entgo.io/ent/cmd/ent@latest

ent:
	ent generate ./ent

infra:
	./start-infra-compose.sh
