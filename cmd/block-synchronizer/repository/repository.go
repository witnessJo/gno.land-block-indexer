package repository

import "context"

type RepositoryBs interface {
	GetNotSequentialBlockNum(ctx context.Context, limit int) (int, error)
}

type RepositoryBsEntConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}
