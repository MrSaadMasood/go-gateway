package gateway

import (
	"context"
	"gateway/internal/config"
	glog "gateway/internal/log"
	"gateway/internal/store"
	"log/slog"
)

type Gatewayer interface {
	Start() config.Config
}

type Gateway struct {
	ctx          context.Context
	configLoader config.Loader
	initializer  store.Initializer
	logger       glog.Logger
}

func (g *Gateway) Start() config.Config {
	c, err := g.configLoader.Load()
	if err != nil {
		panic(err)
	}

	err = g.initializer.Initialize()
	if err != nil {
		panic(err)
	}

	g.logger.Log(slog.LevelInfo, "GATEWAY_STARTED", slog.String("message", "gateway started successfully"))
	return c
}

func New(ctx context.Context, cl config.Loader, si store.Initializer, logger glog.Logger) *Gateway {
	return &Gateway{
		ctx:          ctx,
		configLoader: cl,
		initializer:  si,
		logger:       logger,
	}
}
