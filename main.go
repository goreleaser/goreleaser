package main

import (
	"fmt"

	"github.com/goreleaser/releaser/config"
)

func main() {
	config, err := config.Load("goreleaser.yml")
	if err != nil {
		panic(err)
	}
	fmt.Println(config)
}
