package configuration

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDemoConfig(t *testing.T) {
	config := Default()

	err := yaml.NewEncoder(os.Stdout).Encode(config)
	if err != nil {
		t.Fatal(err)
	}
}
