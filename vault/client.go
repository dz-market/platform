package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
)

type Config struct {
	Addr     string
	RoleID   string
	SecretID string
}

type Client struct {
	api *api.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid vault config: %w", err)
	}

	apiCfg := api.DefaultConfig()
	if apiCfg.Error != nil {
		return nil, fmt.Errorf("read vault config: %w", apiCfg.Error)
	}

	apiCfg.Address = cfg.Addr

	c, err := api.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	auth, err := approle.NewAppRoleAuth(cfg.RoleID, &approle.SecretID{FromString: cfg.SecretID})
	if err != nil {
		return nil, fmt.Errorf("create approle auth: %w", err)
	}

	if _, err := c.Auth().Login(ctx, auth); err != nil {
		return nil, fmt.Errorf("approle login: %w", err)
	}

	return &Client{api: c}, nil
}

func (c Config) validate() error {
	var errs []error

	if c.Addr == "" {
		errs = append(errs, errors.New("addr is required"))
	}

	if c.RoleID == "" {
		errs = append(errs, errors.New("role id is required"))
	}

	if c.SecretID == "" {
		errs = append(errs, errors.New("secret id is required"))
	}

	return errors.Join(errs...)
}
