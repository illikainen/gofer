package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/illikainen/gofer/src/gox"
	"github.com/illikainen/gofer/src/metadata"

	"dario.cat/mergo"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Profile   string
	PrivKey   string
	PubKeys   []string
	Sandbox   string
	Verbosity string
	URL       string
	CacheDir  string
	GoPath    string
	GoCache   string
	Profiles  map[string]Config
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
		CacheDir: filepath.Join(cache, metadata.Name()),
		GoPath:   goPath,
		GoCache:  goCache,
	}
	_, err = toml.DecodeFile(path, &c)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
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
