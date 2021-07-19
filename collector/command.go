package collector

import (
	"context"
	"os/exec"
)

func execVtyshCommand(args ...string) ([]byte, error) {
	var err error
	var output []byte

	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	var a []string
	var executable string

	if vtyshSudo == true {
		a = []string{vtyshPath}
		executable = "/usr/bin/sudo"
	} else {
		a = []string{}
		executable = vtyshPath
	}

	if vtyshPathspace != "" {
		n_opt := []string{"-N", vtyshPathspace}
		a = append(a, n_opt...)
	}

	a = append(a, args...)

	output, err = exec.CommandContext(ctx, executable, a...).Output()

	if err != nil {
		return nil, err
	}

	return output, nil
}
