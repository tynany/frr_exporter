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

	if vtyshUsername != "root" {
		output, err = exec.CommandContext(ctx, "/usr/bin/sudo", args...).Output()
	} else {
		output, err = exec.CommandContext(ctx, vtyshPath, args...).Output()
	}

	if err != nil {
		return nil, err
	}

	return output, nil
}
