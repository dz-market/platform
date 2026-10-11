package vault

import (
	"fmt"
	"os"
	"strings"
)

const (
	envAddr     = "VAULT_ADDR"
	envRoleID   = "VAULT_ROLE_ID"
	envSecretID = "VAULT_SECRET_ID"
)

func ConfigFromEnv(env func(string) (string, bool)) (Config, error) {
	addr, _ := env(envAddr)

	roleID, err := credential(env, envRoleID)
	if err != nil {
		return Config{}, err
	}

	secretID, err := credential(env, envSecretID)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:     addr,
		RoleID:   roleID,
		SecretID: secretID,
	}, nil
}

func credential(env func(string) (string, bool), name string) (string, error) {
	if v, ok := env(name); ok && v != "" {
		return v, nil
	}

	path, ok := env(name + "_FILE")
	if !ok || path == "" {
		return "", nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s_FILE: %w", name, err)
	}

	return strings.TrimSpace(string(raw)), nil
}
