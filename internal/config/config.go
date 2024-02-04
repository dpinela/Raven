package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Settings struct {
	GameLocation string
}

func Get() (Settings, error) {
	path, err := configFilePath()
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	_, err = toml.DecodeFile(path, &s)
	if err != nil {
		return Settings{}, err
	}
	return s, nil
}

func Write(s Settings) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	err = toml.NewEncoder(f).Encode(s)
	if err != nil {
		return err
	}
	return f.Close()
}

func configFilePath() (string, error) {
	cd, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cd, "raven-installer", "config.toml"), nil
}
