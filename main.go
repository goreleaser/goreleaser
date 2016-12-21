package main

import (
	"fmt"

	"github.com/goreleaser/releaser/build"
	"github.com/goreleaser/releaser/config"
)

var version = "none"

func main() {
	config, err := config.Load("goreleaser.yml")
	if err != nil {
		panic(err)
	}
	err = build.Build("master", config)
	fmt.Println(err)
}
