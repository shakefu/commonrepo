// Package config provides config parsing and other helpers
package config

import (
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

type YamlConfig struct {
	// Source options
	YamlSource  `yaml:",inline"`
	Template    []string            `yaml:"template"`
	Install     []map[string]string `yaml:"install"`
	InstallFrom string              `yaml:"install-from"`
	InstallWith []string            `yaml:"install-with"`
	// Consumer options
	Upstream     []yamlUpstream         `yaml:"upstream"`
	TemplateVars map[string]interface{} `yaml:"template-vars"`
	// Internal
	raw []byte
}

type YamlSource struct {
	Include []string
	Exclude []string
	Rename  []map[string]string
}

type yamlUpstream struct {
	URL        string `yaml:"url"`
	Ref        string `yaml:"ref"`
	YamlSource `yaml:",inline"`
}

var (
	ErrRenameInvalid = errors.New("rename entry is not valid")
)

// Unmarshal data into this YamlConfig
func (config *YamlConfig) Unmarshal(data []byte) (err error) {
	config.raw = data
	err = yaml.Unmarshal(data, config)
	// TODO: Handle making pretty error messages for when config fails parsing.
	// This would be super useful for the CLI output to be really nice.
	return
}

// Raw returns the raw yaml data
func (config *YamlConfig) Raw() string {
	return string(config.raw)
}

// YamlParse returns a YamlConfig instance from the given data
func YamlParse(data []byte) (config *YamlConfig, err error) {
	config = &YamlConfig{}
	err = config.Unmarshal(data)
	return
}
