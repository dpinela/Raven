package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dpinela/Raven/internal/config"
	"github.com/dpinela/Raven/internal/modlinks"
)

func setup(args []string) error {
	if len(args) < 1 {
		return errors.New("setup: expect game location as argument")
	}

	location := args[0]

	wrap := func(err error) error {
		return fmt.Errorf("setup at %s: %w", args[0], err)
	}

	location, err := normalizeGamePath(args[0])
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
	err = config.Write(config.Settings{GameLocation: location})
	if err != nil {
		return wrap(err)
	}
	return nil
}

const gameExeName = "DeathsDoor.exe"

func normalizeGamePath(location string) (string, error) {
	if filepath.Base(location) != gameExeName {
		info, err := os.Stat(location)
		if err != nil {
			return "", err
		}
		if !info.Mode().IsDir() {
			return "", fmt.Errorf("%s is not a directory", location)
		}
		location = filepath.Join(location, gameExeName)
	}
	info, err := os.Stat(location)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("game not found at %s", location)
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("thing at %s is not a regular file", location)
	}
	return filepath.Dir(location), nil
}
