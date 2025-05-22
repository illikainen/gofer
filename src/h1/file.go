package h1

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/pkg/errors"
)

func hashFile(file string) (cksum string, err error) {
	f, err := os.Open(file) // #nosec G304
	if err != nil {
		return "", err
	}
	defer errorx.Defer(f.Close, &err)

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	return hashReader(f, stat.Size())
}

func hashReader(r io.Reader, size int64) (string, error) {
	hash := sha256.New()
	written, err := io.Copy(hash, r)
	if err != nil {
		return "", err
	}

	if written != size {
		return "", errors.Errorf("invalid write size")
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
