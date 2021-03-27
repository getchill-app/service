package service

import (
	"path/filepath"
)

func exeDir() string {
	exe, err := executablePath()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(exe)
}
