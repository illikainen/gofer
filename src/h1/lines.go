package h1

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
)

func hashLines(lines []string) (string, error) {
	hash := sha256.New()
	str := strings.Join(lines, "\n") + "\n"

	written, err := hash.Write([]byte(str))
	if err != nil {
		return "", err
	}

	if len(str) != written {
		return "", errors.Errorf("invalid write count")
	}

	sum := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return "h1:" + sum, nil
}
