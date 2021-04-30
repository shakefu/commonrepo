// Package commonrepo provides top level common functions that can be used by
// subpackages when there is no import cycle.
package commonrepo

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/goccy/go-yaml"
	"github.com/shakefu/commonrepo/gitutil"
	"github.com/shakefu/commonrepo/memrepo"
)

// GetLocalRepo returns a local Repo reference assuming we're executing this
// somewhere in the context of a git repository structure.
//
// This isn't really useful for anything except tests.
func GetLocalRepo() (repo memrepo.Repo, err error) {
	path, err := gitutil.FindLocalRepoPath()
	if err != nil {
		return
	}
	// repo, err = memrepo.NewRepo(fmt.Sprintf("file://%s", path))
	repo, err = memrepo.NewRepo(path)
	return
}

/****************************
* Configuration parsing stuff
 */

type Config struct {
	Source   `yaml:",inline"`
	Upstream []Upstream
	raw      []byte
}

type Source struct {
	Include []string
	Exclude []string
	Rename  []interface{}
	parsed  bool
}

type Rename struct {
	Match   *regexp.Regexp
	Replace string
}

type Upstream struct {
	URL    string `yaml:"url"`
	Source `yaml:",inline"`
}

var (
	ErrRenameInvalid = errors.New("rename entry is not valid")
)

// Unmarshal data into this Config struct and parse the rename entries.
func (config *Config) Unmarshal(data []byte) (err error) {
	config.raw = data
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return
	}
	err = config.ParseRenames()
	return
}

// ParseRenames parses the rename entries in the config from maps to Rename
// structs.
func (config *Config) ParseRenames() (err error) {
	// Prevent trying to parse multiple times since the casting will fail
	if config.parsed {
		return errors.New("already parsed")
	}

	var renames = []interface{}{}

	// Iterate over our unparsed items
	for i, val := range config.Rename {
		raw, ok := val.(map[string]interface{})
		if !ok {
			err = config.yamlError(ErrRenameInvalid, "$.rename[%d]", i)
			return
		}
		// May be more than one entry per in the map
		for pattern, rawReplace := range raw {
			rename := &Rename{}
			replace, ok := rawReplace.(string)
			if !ok {
				err = config.yamlError(ErrRenameInvalid, "$.rename[%d]", i)
				return
			}
			err = rename.Parse(pattern, string(replace))
			if err != nil {
				// Return a nice annotated error
				err = config.yamlError(err, "$.rename[%d]", i)
				return
			}
			renames = append(renames, rename)
		}
	}

	config.Rename = renames
	return
}

// Parse ensures that the given pattern is a valid regex.
func (rename *Rename) Parse(pattern string, replace string) (err error) {
	rename.Match, err = regexp.Compile(pattern)
	rename.Replace = replace
	return
}

// String returns a string representation of the Rename mapping.
func (rename *Rename) String() string {
	return fmt.Sprintf("%s=>%s", rename.Match.String(), rename.Replace)
}

func (config *Config) yamlError(orig error, pathFmt string,
	pathArgs ...interface{}) error {
	return yamlError(config.raw, orig, pathFmt, pathArgs...)
}

func yamlError(raw []byte, orig error, pathFmt string,
	pathArgs ...interface{}) (err error) {
	pathStr := fmt.Sprintf(pathFmt, pathArgs...)
	path, err := yaml.PathString(pathStr)
	if err != nil {
		return
	}
	source, err := path.AnnotateSource(raw, true)
	if err != nil {
		return
	}
	msg := "\n" + string(source)
	err = errors.Wrap(orig, msg)
	return
}
