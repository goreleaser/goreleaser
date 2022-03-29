// Package yaml wraps gopkg.in/yaml.v3 and helps transition from v2.
package yaml

import (
	"bytes"
	"errors"
	"io"

	yaml "gopkg.in/yaml.v3"
)

// UnmarshalStrict unmarshals a YAML document with strict behavior (only declared fields are tolerated).
func UnmarshalStrict(in []byte, out interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(in))
	decoder.KnownFields(true)

	return handleErr(decoder.Decode(out))
}

// Unmarshal some struct as a YAML document, without strict behavior.
func Unmarshal(in []byte, out interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(in))
	decoder.KnownFields(false)

	return handleErr(decoder.Decode(out))
}

// Marshal some struct as a YAML document.
func Marshal(in interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	encoder := yaml.NewEncoder(b)
	encoder.SetIndent(2) // default is 4

	if err := encoder.Encode(in); err != nil {
		encoder.Close()

		return nil, err
	}

	encoder.Close()

	return b.Bytes(), nil
}

// handleErr mutes io.EOF errors for backward-compatibility.
func handleErr(err error) error {
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}
