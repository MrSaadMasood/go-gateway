package gateway

import (
	"context"
	"gateway/internal/config"
	"gateway/internal/store"
)

type Gatewayer interface {
	Start() config.Config
}

type Gateway struct {
	ctx          context.Context
	configLoader config.Loader
	initializer  store.Initializer
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

	return c
}

func NewGateway(ctx context.Context, cl config.Loader, si store.Initializer) *Gateway {
	return &Gateway{
		ctx:          ctx,
		configLoader: cl,
		initializer:  si,
	}
}
