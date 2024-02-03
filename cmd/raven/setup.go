package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/dpinela/Raven/internal/modlinks"
)

type settings struct {
	GameLocation string
}

func configFilePath() (string, error) {
	cd, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cd, appDirName, "config.toml"), nil
}

func setup(args []string) error {
	if len(args) < 1 {
		return errors.New("setup: expect game location as argument")
	}

	location := args[0]

	wrap := func(err error) error {
		return fmt.Errorf("setup at %s: %w", location, err)
	}

	if err := checkGameAtPath(location); err != nil {
		return wrap(err)
	}

	location = filepath.Dir(location)

	cfPath, err := configFilePath()
	if err != nil {
		return wrap(err)
	}

	cachedir, err := os.UserCacheDir()
	if err != nil {
		return wrap(err)
	}

	r, err := modlinks.Get()
	if err != nil {
		return wrap(err)
	}
	bie, err := r.GetBase("BepInEx")
	if err != nil {
		return wrap(err)
	}
	f, err := getModFile(cachedir, &bie)
	if err != nil {
		return wrap(err)
	}
	err = extractZip(f, f.Size, bie.Name, location)
	f.Close()
	if err != nil {
		return wrap(err)
	}
	err = writeSettings(cfPath, settings{GameLocation: location})
	if err != nil {
		return wrap(err)
	}
	return nil
}

const gameExeName = "DeathsDoor.exe"

func checkGameAtPath(location string) error {
	if filepath.Base(location) != gameExeName {
		info, err := os.Stat(location)
		if err != nil {
			return err
		}
		if !info.Mode().IsDir() {
			return fmt.Errorf("%s is not a directory", location)
		}
		location = filepath.Join(location, gameExeName)
	}
	info, err := os.Stat(location)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("game not found at %s", location)
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("thing at %s is not a regular file", location)
	}
	return nil
}

func writeSettings(path string, s settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
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
