// Package source provides pipes to take care of using the correct source files.
// For the releasing process we need the files of the tag we are releasing.
package source

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

	"github.com/goreleaser/goreleaser/context"
)

// Pipe to use the latest Git tag as source.
type Pipe struct {
	dirty       bool
	wrongBranch bool
}

// Description of the pipe
func (p *Pipe) Description() string {
	return "Using source from latest tag..."
}

// Run uses the latest tag as source.
// Uncommited changes are stashed.
func (p *Pipe) Run(ctx *context.Context) error {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	dirty := err != nil

	if dirty {
		log.Println("Stashing changes...")
		cmd = exec.Command("git", "stash", "--include-untracked", "--quiet")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed stashing changes: %s", stdout.String())
		}
	}

	p.dirty = dirty

	cmd = exec.Command("git", "describe", "--exact-match", "--match", ctx.Git.CurrentTag)
	err = cmd.Run()
	wrongBranch := err != nil

	if wrongBranch {
		log.Println("Checking out tag...")
		cmd = exec.Command("git", "checkout", ctx.Git.CurrentTag)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("failed changing branch: %s", stdout.String())
		}
	}

	p.wrongBranch = wrongBranch

	return nil
}

// Clean switches back to the original branch and restores changes.
func (p *Pipe) Clean(ctx *context.Context) {
	if p.wrongBranch {
		log.Println("Checking out original branch...")
		cmd := exec.Command("git", "checkout", "-")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		if err := cmd.Run(); err != nil {
			log.Printf("failed changing branch: %s\n", stdout.String())
		}
	}

	if p.dirty {
		log.Println("Popping stashed changes...")
		cmd := exec.Command("git", "stash", "pop")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		if err := cmd.Run(); err != nil {
			log.Printf("failed popping stashed changes: %s\n", stdout.String())
		}
	}
}
