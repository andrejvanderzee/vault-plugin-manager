package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type Plugin struct {
	Name   string
	Type   string
	SHA256 string
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s/%s", p.Type, p.Name)
}

type PluginSyncSession struct {
	PluginPath              string
	Interval                time.Duration
	AWSSession              *session.Session
	S3Bucket                string
	VaultClient             *api.Client
	VaultAuthPath           string
	VaultAuthRole           string
	VaultMaxRetries         int
	ServiceAccountTokenPath string
}

func newPluginSyncSession(c *cli.Context) (*PluginSyncSession, error) {

	s, err := newAWSSession(c.String("region"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create AWS session")
	}

	maxRetries := c.Int("vault-max-retries")
	address := c.String("vault-addr")
	v, err := newVaultClient(address, maxRetries)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Vault client")
	}

	return &PluginSyncSession{
		PluginPath:              c.String("plugin-path"),
		Interval:                c.Duration("interval"),
		AWSSession:              s,
		S3Bucket:                c.String("s3-bucket"),
		VaultClient:             v,
		VaultAuthPath:           c.String("vault-auth-path"),
		VaultAuthRole:           c.String("vault-auth-role"),
		VaultMaxRetries:         maxRetries,
		ServiceAccountTokenPath: c.String("sa-token-path"),
	}, nil
}

func (s *PluginSyncSession) syncPlugins() error {

	log.Infof("Get list from s3://%s", s.S3Bucket)
	s3Plugs, err := s.s3ListPlugins()
	if err != nil {
		return errors.Wrapf(err, "failed to list plugins from S3 bucket %s", s.S3Bucket)
	}

	log.Info("Log into Vault")
	err = s.vaultLogin()
	if err != nil {
		return errors.Wrap(err, "failed to login Vault")
	}

	for _, s3p := range s3Plugs {

		log.Infof("Sync plugin %s", s3p)
		err := s.syncPlugin(s3p)
		if err != nil {
			log.Errorf("Failed to syn plugin: %s", err)
		}
	}

	return nil
}

func (s *PluginSyncSession) syncPlugin(s3p Plugin) error {

	fsp, err := s.getFSPlugin(s3p)
	if err != nil {
		return errors.Wrapf(err, "failed to get filesystem plugin %s", s3p.Name)
	}

	if fsp == nil || fsp.SHA256 != s3p.SHA256 {

		log.Infof("Downloading s3://%s/%s to %s", s.S3Bucket, s3p.Name, s.PluginPath)
		err := s.s3DownloadPlugin(s3p.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to download plugin %s from bucket %s", s3p, s.S3Bucket)
		}

		path := filepath.Join(s.PluginPath, s3p.Name)
		err = os.Chmod(path, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to chmod %s", path)
		}

		log.Info("Read downloaded plugin from filesystem")
		fsp, err = s.getFSPlugin(s3p)
		if err != nil {
			return errors.Wrapf(err, "failed to get filesystem plugin %s", s3p.Name)
		}

		log.Info("Register plugin at Vault")
		err = s.vaultRegisterPlugin(*fsp)
		if err != nil {
			return errors.Wrapf(err, "failed to register plugin %s", fsp)
		}

		log.Info("Reload plugin in Vault")
		err = s.vaultLoadPlugin(*fsp)
		if err != nil {
			return errors.Wrapf(err, "failed to register plugin %s", fsp)
		}
	} else {

		log.Info("Read plugin info from Vault")
		vp, err := s.vaultReadPluginInfo(*fsp)
		if err != nil {
			return errors.Wrap(err, "failed to read plugin from Vault")
		} else if vp == nil {
			log.Info("Plugin not registered")
		}

		if vp != nil && vp.SHA256 == fsp.SHA256 {
			log.Info("Plugin already registered")
			return nil
		}

		log.Info("Register plugin at Vault")
		err = s.vaultRegisterPlugin(*fsp)
		if err != nil {
			return errors.Wrapf(err, "failed to register plugin %s", fsp)
		}

		log.Info("Reload plugin in Vault")
		err = s.vaultLoadPlugin(*fsp)
		if err != nil {
			return errors.Wrapf(err, "failed to register plugin %s", fsp)
		}
	}

	return nil
}
