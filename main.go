package main

import (
	"fmt"
	"log"

	"github.com/goreleaser/releaser/build"
	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/git"
)

var version = "none"

func main() {
	config, err := config.Load("goreleaser.yml")
	if err != nil {
		panic(err)
	}
	tag, err := git.CurrentTag()
	if err != nil {
		panic(err)
	}
	previousTag, err := git.PreviousTag()
	diff, err := git.Log(previousTag, tag)
	if err != nil {
		panic(err)
	}
	log.Println(diff)
	fmt.Println("Building", tag, "...")
	err = build.Build(tag, config)
	fmt.Println(err)
}
