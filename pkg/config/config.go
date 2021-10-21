// Package config provides the functionality to parse a commonrepo configuration file
package config

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
)

// ParseConfig takes yaml data and returns a Config instance
func ParseConfig(data []byte) (config *Config, err error) {
	cfg, err := YamlParse(data)
	if err != nil {
		return nil, err
	}
	var include []string
	if cfg.Include != nil {
		include = cfg.Include
	} else {
		include = []string{}
	}

	var exclude []string
	if cfg.Exclude != nil {
		exclude = cfg.Exclude
	} else {
		exclude = []string{}
	}

	var template []string
	if cfg.Template != nil {
		template = cfg.Template
	} else {
		template = []string{}
	}

	var templateVars map[string]interface{}
	if cfg.TemplateVars != nil {
		templateVars = cfg.TemplateVars
	} else {
		templateVars = map[string]interface{}{}
	}

	config = &Config{}
	config.Include = include
	config.Exclude = exclude
	config.Template = template
	config.TemplateVars = templateVars
	config.InstallFrom = cfg.InstallFrom
	config.InstallWith = cfg.InstallWith

	if err = config.copyRename(cfg.Rename); err != nil {
		return nil, err
	}
	if err = config.copyUpstream(cfg.Upstream); err != nil {
		return nil, err
	}
	if err = config.copyInstall(cfg.Install); err != nil {
		return nil, err
	}
	return
}

// Config provides the desired configuration for the commonrepo
type Config struct {
	Include      []string               // File globs to include
	Exclude      []string               // File globs to exclude
	Template     []string               // File globs to treat as templates
	TemplateVars map[string]interface{} // Map of template variables
	Install      []Install              // List of tool versions to install
	InstallFrom  string                 // Path to install from
	InstallWith  []string               // Priority list of install managers to use
	Rename       []Rename               // Rename regex rules to apply to files
	Upstream     []Upstream             // List of upstream CommonRepos
}

type Upstream struct {
	URL     string
	Ref     string
	Include []string
	Exclude []string
	Rename  []Rename
}

type Install struct {
	Name    string
	Version *semver.Constraints
}

// copyRename parses and copies the renames into our config
func (config *Config) copyRename(renames []map[string]string) (err error) {
	config.Rename, err = parseRenames(renames)
	return
}

// copyUpstream parses and copies the upstreams into our config
func (config *Config) copyUpstream(upstreams []yamlUpstream) (err error) {
	var renames []Rename
	for _, item := range upstreams {
		if renames, err = parseRenames(item.Rename); err != nil {
			return
		}

		var includes []string
		if item.Include != nil {
			includes = item.Include
		} else {
			includes = []string{}
		}

		var excludes []string
		if item.Exclude != nil {
			excludes = item.Exclude
		} else {
			excludes = []string{}
		}

		config.Upstream = append(config.Upstream, Upstream{
			URL:     item.URL,
			Ref:     item.Ref,
			Include: includes,
			Exclude: excludes,
			Rename:  renames,
		})
	}
	return
}

// copyInstall parses and copies the installs into our config
func (config *Config) copyInstall(installs []map[string]string) (err error) {
	var constraints *semver.Constraints
	// Iterate over the ordered list of installable versions
	for _, item := range installs {
		for name, version := range item {
			// Try to parse our install constraint
			if constraints, err = semver.NewConstraint(version); err != nil {
				return
			}
			config.Install = append(config.Install, Install{name, constraints})
		}
	}
	return
}

// ApplyRename modifies the given path according to the rename rules
func (config *Config) ApplyRename(path string) string {
	// Apply all the rename transforms to this path, in order
	// ... This is sort of a design decision since it'll let a single file be
	// renamed multiple times, which could be useful in some very narrow edge
	// cases.
	for _, rename := range config.Rename {
		path = rename.Apply(path)
	}
	return path
}

// Rename is a parsed rename structure which transforms paths within the repository
type Rename struct {
	Match   *regexp.Regexp
	Replace string
}

// String gives us a string representation of the rename
func (rename *Rename) String() string {
	return rename.Match.String() + ": " + rename.Replace
}

// Check tests whether a rename applies to a given path
func (rename *Rename) Check(path string) bool {
	return rename.Match.MatchString(path)
}

// Apply transforms the given path using the rename rule
func (rename *Rename) Apply(path string) string {
	rvals := rename.Match.FindStringSubmatch(path)
	if rvals == nil {
		return ""
	}
	vals := rvals[1:]
	// We have to copy our strings over to a slice of interfaces so we can jam
	// it into Sprintf
	ivals := make([]interface{}, len(vals))
	for i, v := range vals {
		ivals[i] = v
	}
	path = fmt.Sprintf(rename.Replace, ivals...)
	return path
}

// parseRenames parses and returns a list of renames
func parseRenames(renames []map[string]string) (parsed []Rename, err error) {
	var re *regexp.Regexp
	parsed = []Rename{}
	// Iterate over our list of rename maps
	for _, item := range renames {
		for match, replace := range item {
			// Try to compile the regex given
			re, err = regexp.Compile(match)
			if err != nil {
				return
			}
			// This may be slow since it's not pre-allocated
			parsed = append(parsed, Rename{re, replace})
		}
	}
	return
}
