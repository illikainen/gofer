package h1

import (
	"archive/zip"
	"fmt"
	"sort"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/pkg/errors"
	"golang.org/x/mod/sumdb/dirhash"
)

func VerifyZip(file string, cksum string) error {
	actualCksum, err := HashZip(file)
	if err != nil {
		return err
	}

	if actualCksum != cksum {
		return errors.Errorf("%s: bad checksum: %s != %s", file, actualCksum, cksum)
	}

	return nil
}

func HashZip(file string) (cksum string, err error) {
	z, err := zip.OpenReader(file)
	if err != nil {
		return "", err
	}
	defer errorx.Defer(z.Close, &err)

	if z.Comment != "" {
		return "", errors.Errorf("invalid zip comment")
	}

	files := append([]*zip.File{}, z.File...)
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Name < files[j].Name
	})
	if len(files) <= 0 {
		return "", errors.Errorf("invalid file count")
	}

	lines := []string{}
	for _, elt := range files {
		_, err := validatePath(elt.Name)
		if err != nil {
			return "", err
		}

		if elt.Comment != "" {
			return "", errors.Errorf("invalid comment")
		}

		if elt.NonUTF8 {
			return "", errors.Errorf("invalid filename encoding")
		}

		if elt.Flags != 0 && elt.Flags != 8 {
			return "", errors.Errorf("invalid flags: %d", elt.Flags)
		}

		if elt.Method != 0 && elt.Method != 8 {
			return "", errors.Errorf("invalid compression method: %d", elt.Method)
		}

		if len(elt.Extra) != 0 {
			return "", errors.Errorf("unexpected extra data")
		}

		if elt.UncompressedSize64 > 1024*1024*100 {
			return "", errors.Errorf("invalid size")
		}

		f, err := elt.Open()
		if err != nil {
			return "", err
		}

		cksum, err := hashReader(f, int64(elt.UncompressedSize64))
		if err != nil {
			return "", errorx.Join(err, f.Close())
		}

		err = f.Close()
		if err != nil {
			return "", err
		}

		lines = append(lines, fmt.Sprintf("%s  %s", cksum, elt.Name))
	}

	h1, err := hashLines(lines)
	if err != nil {
		return "", err
	}

	upstreamH1, err := dirhash.HashZip(file, dirhash.Hash1)
	if err != nil {
		return "", err
	}
	if len(h1) != 47 || upstreamH1 != h1 {
		return "", errors.Errorf("bug")
	}

	return h1, nil
}
