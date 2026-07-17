package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("internal/pipe/gentoo/gentoo.go")
	s := string(b)

	if !strings.Contains(s, "\"io\"") {
		s = strings.Replace(s, "\"encoding/xml\"", "\"encoding/xml\"\n\t\"io\"\n\t\"hash\"", 1)
	}

	searchHash := `
		b, err := os.ReadFile(art.Path)
		if err != nil {
			return err
		}

		line := fmt.Sprintf("DIST %s %d", art.Name, size)
		for _, algo := range manifestHashes {
			algo = strings.ToUpper(algo)
			switch algo {
			case "BLAKE2B":
				sum := blake2b.Sum512(b)
				line = fmt.Sprintf("%s BLAKE2B %x", line, sum)
			case "SHA512":
				sum := sha512.Sum512(b)
				line = fmt.Sprintf("%s SHA512 %x", line, sum)
			}
		}`

	replaceHash := `
		f, err := os.Open(art.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		var writers []io.Writer
		var b2b hash.Hash
		var s512 hash.Hash

		for _, algo := range manifestHashes {
			algo = strings.ToUpper(algo)
			if algo == "BLAKE2B" {
				b2b, _ = blake2b.New512(nil)
				writers = append(writers, b2b)
			} else if algo == "SHA512" {
				s512 = sha512.New()
				writers = append(writers, s512)
			}
		}

		if len(writers) > 0 {
			if _, err := io.Copy(io.MultiWriter(writers...), f); err != nil {
				return err
			}
		}

		line := fmt.Sprintf("DIST %s %d", art.Name, size)
		for _, algo := range manifestHashes {
			algo = strings.ToUpper(algo)
			if algo == "BLAKE2B" && b2b != nil {
				line += fmt.Sprintf(" BLAKE2B %x", b2b.Sum(nil))
			} else if algo == "SHA512" && s512 != nil {
				line += fmt.Sprintf(" SHA512 %x", s512.Sum(nil))
			}
		}`

	s = strings.Replace(s, searchHash, replaceHash, 1)
	os.WriteFile("internal/pipe/gentoo/gentoo.go", []byte(s), 0644)
	fmt.Println("Applied hash fix")
}
