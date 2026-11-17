package vault

import (
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type Client struct {
	api *api.Client
}

func NewWithAddr(addr string) (*Client, error) {
	if addr == "" {
		return nil, errors.New("address is required")
	}

	cfg := api.DefaultConfig()
	cfg.Address = addr
	cfg.AgentAddress = ""

	return New(cfg)
}

func New(cfg *api.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if cfg.Error != nil {
		return nil, fmt.Errorf("configure vault client: %w", cfg.Error)
	}

	if cfg.Address == "" {
		return nil, errors.New("address is required")
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	return &Client{
		api: client,
	}, nil
}
