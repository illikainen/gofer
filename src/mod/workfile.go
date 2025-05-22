package mod

import (
	"github.com/illikainen/go-utils/src/iofs"
	"golang.org/x/mod/modfile"
)

func ParseWork(file string) (*modfile.WorkFile, error) {
	data, err := iofs.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return modfile.ParseWork(file, data, nil)
}
