package h1

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/mod/zip"
)

func VerifyDir(dir string, name string, version string, cksum string) error {
	actualCksum, err := HashDir(dir, name, version)
	if err != nil {
		return err
	}

	if actualCksum != cksum {
		return errors.Errorf("%s: bad checksum: %s != %s", dir, actualCksum, cksum)
	}

	check, err := zip.CheckDir(dir)
	if err != nil {
		return err
	}

	if len(check.Omitted) != 0 || len(check.Invalid) != 0 || check.SizeError != nil {
		return errors.Errorf("%s: invalid content", dir)
	}
	return nil
}

func HashDir(dir string, name string, version string) (string, error) {
	stat, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", errors.Errorf("%s must be a directory", dir)
	}

	files, err := getDirFiles(dir, name, version)
	if err != nil {
		return "", err
	}

	lines := []string{}
	for _, f := range files {
		cksum, err := hashFile(f.path)
		if err != nil {
			return "", err
		}

		lines = append(lines, fmt.Sprintf("%s  %s", cksum, f.hashPath))
	}

	h1, err := hashLines(lines)
	if err != nil {
		return "", err
	}

	upstreamH1, err := dirhash.HashDir(dir, name+"@"+version, dirhash.Hash1)
	if err != nil {
		return "", err
	}
	if len(h1) != 47 || upstreamH1 != h1 {
		return "", errors.Errorf("bug")
	}

	return h1, nil
}

type file struct {
	path     string
	hashPath string
}

func getDirFiles(dir string, name string, version string) ([]*file, error) {
	files := []*file{}
	dirs := []string{}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			dirs = append(dirs, path)
		} else {
			stat, err := d.Info()
			if err != nil {
				return err
			}
			if stat.Mode()&os.ModeType != 0 {
				return errors.Errorf("unsupported file type for %s", path)
			}

			hashPath, err := validatePath(fmt.Sprintf("%s@%s%s", name, version, path[len(dir):]))
			if err != nil {
				return err
			}
			files = append(files, &file{
				path:     path,
				hashPath: hashPath,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Forbid dangling directories because they're not considered by the Go
	// H1 algorithm.
	for _, d := range dirs {
		used := false
		for _, f := range files {
			sep := string(os.PathSeparator)
			if strings.HasPrefix(f.path+sep, strings.TrimRight(d, sep)+sep) {
				used = true
			}
		}

		if !used {
			return nil, errors.Errorf("invalid subdir: %s", d)
		}
	}

	if len(files) <= 0 {
		return nil, errors.Errorf("invalid length")
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].path < files[j].path
	})
	return files, err
}

func validatePath(path string) (string, error) {
	matched, err := regexp.MatchString(`^[a-z_][a-zA-Z0-9 @!/._-]+$`, path)
	if err != nil {
		return "", err
	}
	if !matched || strings.Contains(path, "..") {
		return "", errors.Errorf("invalid path: %s", path)
	}

	return path, nil
}
