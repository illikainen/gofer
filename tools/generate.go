//go:build generate

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := writeMetadata("src/metadata/metadata.json")
	if err != nil {
		log.Fatalf("%s", err)
	}

	err = copyFile("go.mod", "src/tools/tools.mod")
	if err != nil {
		log.Fatalf("%s", err)
	}

	err = copyFile("go.sum", "src/tools/tools.sum")
	if err != nil {
		log.Fatalf("%s", err)
	}
}

type metadata struct {
	Name    string
	Version string
	Commit  string
	Branch  string
}

func writeMetadata(file string) (err error) {
	commitCmd := exec.Command("git", "rev-parse", "HEAD") // #nosec G204
	commit, err := commitCmd.Output()
	if err != nil {
		return err
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD") // #nosec G204
	branch, err := branchCmd.Output()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata{
		Name:    "gofer",
		Version: "0.0.0",
		Commit:  strings.Trim(string(commit), "\r\n"),
		Branch:  strings.Trim(string(branch), "\r\n"),
	}, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer errorx.Defer(f.Close, &err)

	data = append(data, '\n')
	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.Errorf("invalid write")
	}

	return nil
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0600)
}
