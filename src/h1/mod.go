package h1

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/mod/sumdb/dirhash"
)

func VerifyMod(file string, cksum string) error {
	actualCksum, err := HashMod(file)
	if err != nil {
		return err
	}

	if actualCksum != cksum {
		return errors.Errorf("%s: bad checksum: %s != %s", file, actualCksum, cksum)
	}

	return nil
}

func HashMod(file string) (h1 string, err error) {
	cksum, err := hashFile(file)
	if err != nil {
		return "", err
	}

	h1, err = hashLines([]string{fmt.Sprintf("%s  go.mod", cksum)})
	if err != nil {
		return "", err
	}

	upstreamH1, err := dirhash.Hash1([]string{"go.mod"}, func(_ string) (io.ReadCloser, error) {
		return os.Open(file) // #nosec G304
	})
	if err != nil {
		return "", err
	}
	if len(h1) != 47 || upstreamH1 != h1 {
		return "", errors.Errorf("bug")
	}

	return h1, nil
}
