package mod

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// The Go cache system replaces capital letters in module names with a '!'
// followed by the downcased letter.
func downcase(str string) string {
	rx := regexp.MustCompile(`[A-Z]`)
	return rx.ReplaceAllStringFunc(str, func(s string) string {
		return "!" + strings.ToLower(s)
	})
}

func validateName(name string) (string, error) {
	matched, err := regexp.MatchString(`^[a-z][a-zA-Z0-9/._-]+$`, name)
	if err != nil {
		return "", err
	}
	if !matched || strings.Contains(name, "..") {
		return "", errors.Errorf("invalid name: %s", name)
	}

	return name, nil
}

func validateVersion(version string) (string, bool, error) {
	mod := false
	if strings.HasSuffix(version, "/go.mod") {
		mod = true
		version = version[:len(version)-len("/go.mod")]
	}

	matched, err := regexp.MatchString(`^v[a-z0-9.-]+$`, version)
	if err != nil {
		return "", false, err
	}
	if !matched || strings.Contains(version, "..") {
		return "", false, errors.Errorf("invalid version: %s", version)
	}

	return version, mod, nil
}

func validateChecksum(cksum string) (string, error) {
	matched, err := regexp.MatchString(`^h1:[a-zA-Z0-9+/=]{44}$`, cksum)
	if err != nil {
		return "", err
	}
	if !matched {
		return "", errors.Errorf("invalid checksum: %s", cksum)
	}

	return cksum, nil
}
