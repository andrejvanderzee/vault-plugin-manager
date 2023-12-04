package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func (s *PluginSyncSession) getFSPlugin(plug Plugin) (*Plugin, error) {

	path := filepath.Join(s.PluginPath, plug.Name)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, errors.Wrapf(err, "could not open plugin %s", path)
		}
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrapf(err, "failed to calculate sha256 of executable %s", path)
	}

	return &Plugin{
		Name:   filepath.Base(path),
		Type:   plug.Type,
		SHA256: fmt.Sprintf("%x", h.Sum(nil)),
	}, nil
}
