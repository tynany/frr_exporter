package collector

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func execVtyshCommand(args ...string) ([]byte, error) {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	var a []string
	var executable string

	if vtyshSudo {
		a = []string{vtyshPath}
		executable = "/usr/bin/sudo"
	} else {
		a = []string{}
		executable = vtyshPath
	}

	if frrVTYSHOptions != "" {
		frrOptions := strings.Split(frrVTYSHOptions, " ")
		a = append(a, frrOptions...)
	}

	a = append(a, args...)

	cmd := exec.CommandContext(ctx, executable, a...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, strings.Replace(stderr.String(), "\n", " ", -1))
	}

	return stdout.Bytes(), nil
}
