package modlinks

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/BurntSushi/toml"
)

const modlinksURL = "https://github.com/dd-modding/modlinks/archive/refs/heads/main.zip"

type Mod struct {
	Name         string
	Description  string
	Repository   string
	Dependencies []string
	Integrations []string
	Link         string
	SHA256       string
}

type Repository struct {
	arch *zip.Reader
}

func Get() (*Repository, error) {
	wrap := func(err error) error {
		return fmt.Errorf("get modlinks: %w", err)
	}

	resp, err := http.Get(modlinksURL)
	if err != nil {
		return nil, wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get modlinks: got HTTP status %s, expected 200 OK", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, wrap(err)
	}
	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, wrap(err)
	}
	return &Repository{z}, nil
}

func (r *Repository) GetBase(thingName string) (Mod, error) {
	return r.get("base", thingName)
}

func (r *Repository) GetMod(modName string) (Mod, error) {
	return r.get("mods", modName)
}

func (r *Repository) get(section, modName string) (Mod, error) {
	var m Mod
	_, err := toml.DecodeFS(r.arch, path.Join("modlinks-main", section, modName + ".toml"), &m)
	if err != nil {
		return Mod{}, fmt.Errorf("get mod %q: %w", modName, err)
	}
	return m, nil
}
