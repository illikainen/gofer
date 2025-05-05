package metadata

import (
	_ "embed"
	"encoding/json"
)

var metadata struct {
	Name    string
	Version string
	Commit  string
	Branch  string
}

//go:embed metadata.json
var metadataBytes []byte

func init() {
	err := json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		panic(err)
	}
}

func Name() string {
	return metadata.Name
}

func Version() string {
	return metadata.Version
}

func Commit() string {
	return metadata.Commit
}

func Branch() string {
	return metadata.Branch
}
