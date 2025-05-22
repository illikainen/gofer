package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/illikainen/gofer/src/metadata"

	"github.com/BurntSushi/toml"
)

type Root struct {
	Settings
	Profiles map[string]Settings
}

type Settings struct {
	PrivKey   string
	PubKeys   []string
	Verbosity string
	URL       string
	CacheDir  string
}

type Config struct {
	Root
	Path string
}

func Read(path string) (*Config, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	c := &Config{
		Path: path,
		Root: Root{
			Settings: Settings{
				CacheDir: filepath.Join(cache, metadata.Name()),
			},
		},
	}
	_, err = toml.DecodeFile(path, &c.Settings)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
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
