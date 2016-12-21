package build

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/goreleaser/releaser/config"
	"github.com/kardianos/osext"
)

func Build(version string, config config.ProjectConfig) error {
	currentPath, err := osext.ExecutableFolder()
	if err != nil {
		return err
	}
	fmt.Println(currentPath)
	for _, os := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			fmt.Println("Building", os, arch)
			cmd := exec.Command(
				"go", "build",
				"-ldflags=\"-s -w -X main.version="+version+"\"",
				"-o", target(os, arch, config.BinaryName),
				config.Main,
			)
			cmd.Env = append(
				cmd.Env,
				"GOOS="+os,
				"GOARCH="+arch,
				// "GOPATH="+
			)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stdout
			err := cmd.Run()
			fmt.Println(stdout.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func target(os, arch, binary string) string {
	return "dist/" + binary + "_" + os + "_" + arch + "/" + binary
}
