package modlinks

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"

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

func (r *Repository) ResolveModName(modName string) (string, error) {
	names := r.ModNames()
	return ResolveModName(names, modName)
}

func (r *Repository) ModNames() []string {
	modFiles, _ := fs.Glob(r.arch, "modlinks-main/mods/*.toml")
	names := make([]string, len(modFiles))
	for i, mf := range modFiles {
		b := path.Base(mf)
		names[i] = strings.TrimSuffix(b, path.Ext(b))
	}
	return names
}

func (r *Repository) TransitiveClosure(leaves []string) ([]Mod, error) {
	resultSet := map[string]Mod{}
	missingModSet := map[string]error{}
	for _, leaf := range leaves {
		r.transitiveClosure(resultSet, missingModSet, leaf)
	}
	result := make([]Mod, 0, len(resultSet))
	for _, mod := range resultSet {
		result = append(result, mod)
	}
	missing := make(missingModsError, 0, len(missingModSet))
	for name := range missingModSet {
		missing = append(missing, name)
	}
	var err error
	if len(missing) > 0 {
		err = missing
	}
	return result, err
}

func (r *Repository) transitiveClosure(resultSet map[string]Mod, missingMods map[string]error, name string) {
	if _, ok := resultSet[name]; ok {
		return
	}
	if _, notOK := missingMods[name]; notOK {
		return
	}
	m, err := r.GetMod(name)
	if err != nil {
		missingMods[name] = err
		return
	}
	resultSet[name] = m
	for _, dep := range m.Dependencies {
		r.transitiveClosure(resultSet, missingMods, dep)
	}
}

type missingModsError []string

func (err missingModsError) Error() string {
	return fmt.Sprintf("required mods do not exist: %s", strings.Join(err, ","))
}

func (r *Repository) get(section, modName string) (Mod, error) {
	var m Mod
	_, err := toml.DecodeFS(r.arch, path.Join("modlinks-main", section, modName+".toml"), &m)
	if err != nil {
		return Mod{}, fmt.Errorf("get mod %q: %w", modName, err)
	}
	return m, nil
}

type unknownModError struct{ requestedName string }

func (err *unknownModError) Error() string {
	return fmt.Sprintf("%q matches no mods", err.requestedName)
}

type ambiguousModError struct {
	requestedName string
	possibilities []string
}

func (err *ambiguousModError) Error() string {
	return fmt.Sprintf("%q is ambiguous: matches %s", err.requestedName, strings.Join(err.possibilities, ", "))
}

type duplicateModError struct {
	requestedName string
	numMatches    int
}

func (err *duplicateModError) Error() string {
	return fmt.Sprintf("%q is ambiguous: %d mods with that exact name exist", err.requestedName, err.numMatches)
}

func ResolveModName(ms []string, requestedName string) (string, error) {
	pattern, err := regexp.Compile("(?i)" + regexp.QuoteMeta(requestedName))
	if err != nil {
		return "", err
	}
	var matches []string
	for _, m := range ms {
		if pattern.MatchString(m) {
			matches = append(matches, m)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", &unknownModError{requestedName}
	}

	fullMatches := matches[:0]
	for _, m := range matches {
		if strings.EqualFold(m, requestedName) {
			fullMatches = append(fullMatches, m)
		}
	}
	switch len(fullMatches) {
	case 1:
		return fullMatches[0], nil
	case 0:
		// If fullMatches is empty, the previous loop never appended to it, so the contents of
		// the matches slice are intact.
		return "", &ambiguousModError{requestedName, matches}
	}

	numExactMatches := 0
	for _, m := range fullMatches {
		if m == requestedName {
			numExactMatches++
		}
	}
	switch numExactMatches {
	case 1:
		return requestedName, nil
	case 0:
		return "", &ambiguousModError{requestedName, fullMatches}
	default:
		return "", &duplicateModError{requestedName, numExactMatches}
	}
}
