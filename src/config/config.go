package config

import (
	"os"
	"path/filepath"

	"github.com/illikainen/gofer/src/gox"
	"github.com/illikainen/gofer/src/metadata"

	"dario.cat/mergo"
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Config struct {
	Profile   string `toml:"-"`
	PrivKey   string
	PubKeys   []string
	Sandbox   string
	Verbosity string
	URL       string
	CacheDir  string
	GoPath    string
	GoCache   string
	Profiles  map[string]Config `toml:"profile"`
}

func Read(path string, overrides *Config) (*Config, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	goPath, err := gox.GoPath()
	if err != nil {
		return nil, err
	}

	goCache, err := gox.GoCache()
	if err != nil {
		return nil, err
	}

	c := &Config{
		Verbosity: "info",
		CacheDir:  filepath.Join(cache, metadata.Name()),
		GoPath:    goPath,
		GoCache:   goCache,
	}
	_, err = toml.DecodeFile(path, &c)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if overrides.Profile != "" {
		profile, ok := c.Profiles[overrides.Profile]
		if !ok {
			return nil, errors.Errorf("invalid profile: %s", overrides.Profile)
		}

		err = mergo.Merge(c, profile, mergo.WithOverride)
		if err != nil {
			return nil, err
		}
	}

	err = mergo.Merge(c, overrides, mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, metadata.Name()), nil
}

func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.toml"), nil
}
