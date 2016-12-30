package main

import (
	"log"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/pipeline"
	"github.com/goreleaser/releaser/pipeline/brew"
	"github.com/goreleaser/releaser/pipeline/build"
	"github.com/goreleaser/releaser/pipeline/compress"
	"github.com/goreleaser/releaser/pipeline/release"
)

var version = "master"

func main() {
	config, err := config.Load("goreleaser.yml")
	if err != nil {
		log.Fatalln("Failed to load goreleaser.yml:", err.Error())
	}
	var pipeline = []pipeline.Pipe{
		build.Pipe{},
		compress.Pipe{},
		release.Pipe{},
		brew.Pipe{},
	}
	for _, pipe := range pipeline {
		if err := pipe.Work(config); err != nil {
			log.Fatalln(pipe.Name(), "failed:", err.Error())
		}
	}
}
