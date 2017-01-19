// Package source provides pipes to take care of using the correct source files.
// For the releasing process we need the files of the tag we are releasing.
package source

import (
	"bytes"
	"errors"
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
	return "Using source from latest tag"
}

// Run uses the latest tag as source.
// Uncommitted changes are stashed.
func (p *Pipe) Run(ctx *context.Context) error {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	dirty := err != nil

	if dirty {
		log.Println("Stashing changes")
		if err = run("git", "stash", "--include-untracked", "--quiet"); err != nil {
			return fmt.Errorf("failed stashing changes: %v", err)
		}
	}

	p.dirty = dirty

	cmd = exec.Command("git", "describe", "--exact-match", "--match", ctx.Git.CurrentTag)
	err = cmd.Run()
	wrongBranch := err != nil

	if wrongBranch {
		log.Println("Checking out tag")
		if err = run("git", "checkout", ctx.Git.CurrentTag); err != nil {
			return fmt.Errorf("failed changing branch: %v", err)
		}
	}

	p.wrongBranch = wrongBranch

	return nil
}

// Clean switches back to the original branch and restores changes.
func (p *Pipe) Clean(ctx *context.Context) {
	if p.wrongBranch {
		log.Println("Checking out original branch")
		if err := run("git", "checkout", "-"); err != nil {
			log.Println("failed changing branch: ", err.Error())
		}
	}

	if p.dirty {
		log.Println("Popping stashed changes")
		if err := run("git", "stash", "pop"); err != nil {
			log.Println("failed popping stashed changes:", err.Error())
		}
	}
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return errors.New(out.String())
	}
	return nil
}
