package service

import (
	"os/exec"
	"runtime"

	"github.com/pkg/errors"
)

func checkSupportedOS() error {
	switch rt := runtime.GOOS; rt {
	case "darwin", "windows", "linux":
		return nil
	default:
		return errors.Errorf("%s is not currently supported", rt)
	}
}

func checkCodesigned() error {
	exe, exeErr := executablePath()
	if exeErr != nil {
		return exeErr
	}
	cmd := exec.Command("/usr/bin/codesign", "-v", exe) // #nosec
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "%s is not codesigned", exe)
	}
	return nil
}
