package collector

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	vtyshPath       = kingpin.Flag("frr.vtysh.path", "Path of vtysh.").Default("/usr/bin/vtysh").String()
	vtyshTimeout    = kingpin.Flag("frr.vtysh.timeout", "The timeout when running vtysh commands (default: 20s).").Default("20s").Duration()
	vtyshSudo       = kingpin.Flag("frr.vtysh.sudo", "Enable sudo when executing vtysh commands.").Bool()
	frrVTYSHOptions = kingpin.Flag("frr.vtysh.options", "Additional options passed to vtysh.").Default("").String()
)

func execVtyshCommand(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), *vtyshTimeout)
	defer cancel()

	var a []string
	var executable string

	if *vtyshSudo {
		a = []string{*vtyshPath}
		executable = "/usr/bin/sudo"
	} else {
		a = []string{}
		executable = *vtyshPath
	}

	if *frrVTYSHOptions != "" {
		frrOptions := strings.Split(*frrVTYSHOptions, " ")
		a = append(a, frrOptions...)
	}

	a = append(a, args...)

	cmd := exec.CommandContext(ctx, executable, a...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, strings.Replace(stderr.String(), "\n", " ", -1))
	}

	return stdout.Bytes(), nil
}
