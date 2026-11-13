package vault

import (
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type Config struct {
	Addr string
}

type Client struct {
	api *api.Client
}

func New(cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("vault: addr is required")
	}

	apiCfg := api.DefaultConfig()
	if apiCfg.Error != nil {
		return nil, fmt.Errorf("read vault config: %w", apiCfg.Error)
	}

	apiCfg.Address = cfg.Addr

	client, err := api.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	return &Client{api: client}, nil
}
