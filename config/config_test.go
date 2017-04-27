package config

import (
	"testing"

	yaml "gopkg.in/yaml.v1"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	config := Project{}

	if err := yaml.Unmarshal([]byte(cfg), &config); err != nil {
		t.Errorf("Not a valid config. %s", err.Error())
	}
}
