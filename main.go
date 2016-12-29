package main

import (
	"fmt"
	"log"

	"github.com/goreleaser/releaser/build"
	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/git"
	"github.com/goreleaser/releaser/compress"
	"github.com/goreleaser/releaser/release"
)

var version = "none"

func main() {
	config, err := config.Load("goreleaser.yml")
	if err != nil {
		log.Fatalln("Failed to load goreleaser.yml", err.Error())
	}
	tag, err := git.CurrentTag()
	if err != nil {
		log.Fatalln("Failed to get current tag name", err.Error())
	}
	fmt.Println("Building", tag, "...")
	err = build.Build(tag, config)
	if err != nil {
		log.Fatalln("Failed to diff current and previous tags", err.Error())
	}
	err = compress.ArchiveAll(version, config)
	if err != nil {
		log.Fatalln("Failed to create archives", err.Error())
	}
	previousTag, err := git.PreviousTag()
	if err != nil {
		log.Fatalln("Failed to get previous tag name", err.Error())
	}
	diff, err := git.Log(previousTag, tag)
	if err != nil {
		log.Fatalln("Failed to diff current and previous tags", err.Error())
	}
	err = release.Release(tag, diff, config)
	if err != nil {
		log.Fatalln("Failed to create the GitHub release", err.Error())
	}
	if config.Brew.Repo != "" {
		// release to brew
	}
}
