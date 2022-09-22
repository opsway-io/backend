package asynq

import (
	"context"
	"log"

	"github.com/hibiken/asynq"
)

type Config struct {
	Addr string `required:"true"`
}

func NewClient(ctx context.Context, conf Config) *asynq.Client {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: conf.Addr})

	return client
}

func NewHandler(pattern string, handler func(context.Context, *asynq.Task) error, conf Config) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: conf.Addr},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(pattern, handler)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
