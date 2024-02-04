package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpinela/Raven/internal/config"
	"github.com/dpinela/Raven/internal/modlinks"
)

const appDirName = "raven-installer"

type modFile struct {
	*os.File
	Size  int64
	IsZIP bool
}

func getModFile(cachedir string, mod *modlinks.Mod) (*modFile, error) {
	expectedSHA, err := hex.DecodeString(mod.SHA256)
	if err != nil {
		return nil, err
	}
	ext := path.Ext(mod.Link)
	cacheEntry := filepath.Join(cachedir, appDirName, mod.Name+ext)
	f, err := os.Open(cacheEntry)
	if os.IsNotExist(err) {
		fmt.Println("=> Installing", mod.Name, "from", mod.Link)
		return downloadLink(cacheEntry, mod.Link, expectedSHA)
	}
	if err != nil {
		return nil, err
	}
	sha := sha256.New()
	size, err := io.Copy(sha, f)
	if err != nil {
		f.Close()
		return nil, err
	}
	if !bytes.Equal(expectedSHA, sha.Sum(make([]byte, 0, sha256.Size))) {
		f.Close()
		fmt.Println("=> Installing", mod.Name, "from", mod.Link)
		return downloadLink(cacheEntry, mod.Link, expectedSHA)
	}
	fmt.Println("=> Installing", mod.Name, "from cache")
	return &modFile{File: f, Size: size, IsZIP: ext == ".zip"}, nil
}

func isatty(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const ansiEraseLine = "\x1b[G\x1b[K"

func downloadLink(localfile string, url string, expectedSHA []byte) (*modFile, error) {
	wrap := func(err error) error { return fmt.Errorf("download %s: %w", url, err) }

	resp, err := http.Get(url)
	if err != nil {
		return nil, wrap(err)
	}
	defer resp.Body.Close()
	if !isHTTPOK(resp.StatusCode) {
		return nil, fmt.Errorf("download %s: response status was %d", url, resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(localfile), 0750); err != nil {
		return nil, wrap(err)
	}
	f, err := os.Create(localfile)
	if err != nil {
		return nil, wrap(err)
	}

	sha := sha256.New()
	r := io.TeeReader(resp.Body, sha)
	if isatty(os.Stdout) {
		defer fmt.Print(ansiEraseLine)
		var counter byteCounter
		counter.updatePeriod = time.Second
		if fullSize := dataSize(resp.ContentLength); fullSize != -1 {
			counter.update = func(n dataSize) { fmt.Printf(ansiEraseLine+"downloading: %s of %s", n, fullSize) }
		} else {
			counter.update = func(n dataSize) { fmt.Printf(ansiEraseLine+"downloading: %s of ???", n) }
		}
		r = io.TeeReader(r, &counter)
	}
	size, err := io.Copy(f, r)
	if err != nil {
		f.Close()
		return nil, wrap(err)
	}
	if !bytes.Equal(sha.Sum(make([]byte, 0, sha256.Size)), expectedSHA) {
		return nil, fmt.Errorf("download %s: sha256 does not match manifest", url)
	}
	return &modFile{File: f, Size: size, IsZIP: path.Ext(url) == ".zip"}, nil
}

type byteCounter struct {
	bytesWritten dataSize
	lastUpdate   time.Time
	updatePeriod time.Duration
	update       func(dataSize)
}

func (bc *byteCounter) Write(p []byte) (int, error) {
	bc.bytesWritten += dataSize(len(p))
	if now := time.Now(); now.Sub(bc.lastUpdate) > bc.updatePeriod {
		bc.lastUpdate = now
		bc.update(bc.bytesWritten)
	}
	return len(p), nil
}

type dataSize int64

func (n dataSize) String() string {
	switch {
	case n < 1_000:
		return fmt.Sprintf("%d bytes", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.1f kB", float64(n)/1_000)
	case n < 1_000_000_000:
		return fmt.Sprintf("%.1f MB", float64(n)/1_000_000)
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/1_000_000_000)
	}
}

func isHTTPOK(code int) bool { return code >= 200 && code < 300 }

func extractZip(zipfile io.ReaderAt, size int64, name, installdir string) error {
	wrap := func(err error) error { return fmt.Errorf("extract %s: %w", name, err) }
	archive, err := zip.NewReader(zipfile, size)
	if err != nil {
		return wrap(err)
	}
	for _, file := range archive.File {
		// Prevent us from accidentally (or not so accidentally, in case of a malicious input)
		// from writing outside the destination directory.
		dest := joinNoEscape(installdir, filepath.FromSlash(file.Name))
		if strings.HasSuffix(file.Name, "/") {
			err = os.MkdirAll(dest, 0750)
		} else {
			err = writeZipFile(dest, file)
		}
		if err != nil {
			return wrap(err)
		}
	}
	return nil
}

func joinNoEscape(parent string, child string) string {
	return filepath.Join(parent, filepath.Join(string(filepath.Separator), child))
}

func writeZipFile(dest string, file *zip.File) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
		return err
	}
	w, err := os.Create(dest)
	if err != nil {
		return err
	}
	r, err := file.Open()
	if err != nil {
		w.Close()
		return err
	}
	_, err = io.Copy(w, r)
	if err != nil {
		r.Close()
		w.Close()
		return err
	}
	if err := r.Close(); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(dest, file.Modified, file.Modified); err != nil {
		fmt.Println("warning:", err)
	}
	return nil
}

func install(args []string) error {
	settings, err := config.Get()
	if err != nil {
		return err
	}
	if settings.GameLocation == "" {
		return errors.New("setup not done yet")
	}

	cachedir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("cache directory not available: %w", err)
	}

	repo, err := modlinks.Get()
	if err != nil {
		return err
	}
	resolvedMods := make([]string, 0, len(args))
	for _, requestedName := range args {
		mod, err := repo.ResolveModName(requestedName)
		if err != nil {
			fmt.Println(err)
			continue
		}
		resolvedMods = append(resolvedMods, mod)
	}

	downloads, err := repo.TransitiveClosure(resolvedMods)
	if err != nil {
		return err
	}
	for _, dl := range downloads {
		// There's no way we can reasonably install a mod whose name contains a path separator.
		// This also avoids any path traversal vulnerabilities from mod names.
		if strings.ContainsRune(dl.Name, filepath.Separator) {
			fmt.Printf("cannot install %s: contains path separator\n", dl.Name)
			continue
		}
		if strings.ContainsRune(path.Base(dl.Link), filepath.Separator) {
			fmt.Printf("cannot install %s: filename contains path separator\n", dl.Name)
			continue
		}
		file, err := getModFile(cachedir, &dl)
		if err != nil {
			fmt.Printf("cannot install %s: %v\n", dl.Name, err)
			continue
		}
		installdir := filepath.Join(settings.GameLocation, "BepInEx", "plugins", dl.Name)
		if err := removePreviousVersion(dl.Name, installdir); err != nil {
			fmt.Printf("cannot install %s: %v\n", dl.Name, err)
			file.Close()
			continue
		}
		if file.IsZIP {
			err = extractZip(file, file.Size, dl.Name, installdir)
		} else {
			err = extractModDLL(file, path.Base(dl.Link), installdir)
		}
		file.Close()
		if err != nil {
			fmt.Printf("cannot install %s: %v\n", dl.Name, err)
		}
	}
	return nil
}

func extractModDLL(dllfile io.ReadSeeker, filename, installdir string) error {
	wrap := func(err error) error { return fmt.Errorf("extract %s: %w", filename, err) }
	dest := joinNoEscape(installdir, filename)
	if err := os.MkdirAll(installdir, 0750); err != nil {
		return wrap(err)
	}
	if _, err := dllfile.Seek(0, io.SeekStart); err != nil {
		return wrap(err)
	}
	w, err := os.Create(dest)
	if err != nil {
		return wrap(err)
	}
	_, err = io.Copy(w, dllfile)
	if err != nil {
		w.Close()
		return wrap(err)
	}
	if err := w.Close(); err != nil {
		return wrap(err)
	}
	return nil
}

func removePreviousVersion(name, installdir string) error {
	err := os.RemoveAll(installdir)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("yeet installed version of %s: %w", name, err)
}
