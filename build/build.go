package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/goreleaser/releaser/config"
)

func Build(version string, config config.ProjectConfig) error {
	for _, bos := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			fmt.Println("Building", bos+"/"+arch+"...")
			cmd := exec.Command(
				"go",
				"build",
				"-ldflags=-s -w -X main.version="+version,
				"-o", target(bos, arch, config.BinaryName),
				config.Main,
			)
			cmd.Env = append(
				cmd.Env,
				"GOOS="+bos,
				"GOARCH="+arch,
				"GOROOT="+os.Getenv("GOROOT"),
				"GOPATH="+os.Getenv("GOPATH"),
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
