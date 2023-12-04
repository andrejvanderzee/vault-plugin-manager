package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func newVaultClient(maxRetries int) (*api.Client, error) {

	c := api.DefaultConfig()
	if c == nil {
		return nil, errors.New("could not create/read default configuration")
	}
	if c.Error != nil {
		return nil, errors.Wrapf(c.Error, "error encountered setting up default configuration")
	}

	// Vault is in booting phase so let's always retry (not only for 5xx response codes).
	c.CheckRetry = alwaysRetry
	c.MaxRetries = maxRetries

	clt, err := api.NewClient(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vault client")
	}

	return clt, nil
}

func alwaysRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {

	if err != nil {
		log.Error(err)
	}

	if resp == nil {
		return true, fmt.Errorf("nil response")
	}

	switch resp.StatusCode {
	case 200, 201, 202, 204:
		// Success.
		return false, nil
	}

	// Unexpected, retry just in case.
	return true, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func (s *PluginSyncSession) vaultLogin() error {

	jwt, err := os.ReadFile(s.ServiceAccountTokenPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read service account token")
	}

	data := map[string]interface{}{
		"role": s.VaultAuthRole,
		"jwt":  string(jwt),
	}

	vaultPath := path.Join(s.VaultAuthPath, "login")
	secret, err := s.VaultClient.Logical().Write(vaultPath, data)
	if err != nil {
		return errors.Wrapf(err, "failed to write vault-path: %s", vaultPath)
	}

	s.VaultClient.SetToken(secret.Auth.ClientToken)
	return nil
}

func (s *PluginSyncSession) vaultRevoke() error {

	vaultPath := "/auth/token/revoke-self"
	_, err := s.VaultClient.Logical().Write(vaultPath, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to write vault-path: %s", vaultPath)
	}

	return nil
}

func (s *PluginSyncSession) vaultReadPlugin(plug Plugin) (*Plugin, error) {

	vaultPath := path.Join("sys/plugins/catalog", plug.Type, plug.Name)
	secret, err := s.VaultClient.Logical().Read(vaultPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read vault-path: %s", vaultPath)
	}

	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	return &Plugin{
		Name:   plug.Name,
		Type:   plug.Type,
		SHA256: secret.Data["sha256"].(string),
	}, nil
}

func (s *PluginSyncSession) vaultRegisterPlugin(plug Plugin) error {

	data := map[string]interface{}{
		"sha256":  plug.SHA256,
		"command": plug.Name,
	}

	vaultPath := path.Join("sys/plugins/catalog", plug.Type, plug.Name)
	_, err := s.VaultClient.Logical().Write(vaultPath, data)
	if err != nil {
		return errors.Wrapf(err, "failed to write vault-path: %s", vaultPath)
	}

	return nil
}

func (s *PluginSyncSession) vaultLoadPlugin(plug Plugin) error {

	data := map[string]interface{}{
		"plugin": plug.Name,
	}

	vaultPath := "/sys/plugins/reload/backend"
	_, err := s.VaultClient.Logical().Write(vaultPath, data)
	if err != nil {
		return errors.Wrapf(err, "failed to write vault-path: %s", vaultPath)
	}

	return nil
}
