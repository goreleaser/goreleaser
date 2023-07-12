package main

import (
	"context"
	"fmt"

	"dagger.io/dagger"
)

func main() {
	ctx := context.Background()

	// create a Dagger client
	fmt.Println("Initializing Dagger...")
	client, err := dagger.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	src := client.Host().Directory(".") // TODO: excludes

	fmt.Println("Executing CI...")
	builder, err := builder(client, src).Sync(ctx)
	if err != nil {
		fmt.Printf("Error creating CI environment: %+v\n", err)
		panic(err)
	}

	builder = builder.WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build"}).
		WithExec([]string{"go", "test", "./..."})

	out, err := builder.Stdout(ctx)
	if err != nil {
		fmt.Printf("Error in CI pipeline: %+v\n", err)
		panic(err)
	}
	fmt.Println(out)

	// Export built binary
	_, err = builder.File("./goreleaser").Export(ctx, "./goreleaser")
	if err != nil {
		fmt.Printf("Error in CI pipeline: %+v\n", err)
		panic(err)
	}
}
