package common_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/shakefu/commonrepo/pkg/common"

	. "github.com/onsi/gomega"
	"github.com/shakefu/goblin"
	// . "github.com/shakefu/commonrepo/internal/testutil"
)

func TestCommon(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("common", func() {
		g.Describe("ShortestKey", func() {
			g.It("returns the shortest key", func() {
				data := map[string]string{
					"fnord":  "blah",
					"foobar": "boop",
					"foo":    "bar",
				}
				key := ShortestKey(data)
				Expect(key).To(Equal("foo"))
			})
		})

		g.Describe("SortedKeys", func() {
			g.It("returns sorted keys", func() {
				data := map[string]string{
					"fnord":  "blah",
					"foobar": "boop",
					"foo":    "bar",
				}
				keys := SortedKeys(data)
				Expect(keys).To(Equal([]string{"fnord", "foo", "foobar"}))
			})
		})

		g.Describe("ConfigFileGlob", func() {
			g.It("returns the glob for config", func() {
				glob := ConfigFileGlob()
				Expect(glob).To(Equal(".commonrepo.{yaml,yml}"))
			})

			g.It("allows for env override", func() {
				os.Setenv("COMMON_CONFIG_GLOB", ".commonreporc")
				glob := ConfigFileGlob()
				Expect(glob).To(Equal(".commonreporc"))
			})

			g.After(func() {
				os.Unsetenv("COMMON_CONFIG_GLOB")
			})
		})
	})
}

func ExampleShortestKey() {
	// This is useful if you might have a series of glob matches and want to
	// find the most matchiest match.
	data := map[string]string{
		".gitattributes": ".gitattributes",
		".gitignore":     ".gitignore",
		".git":           ".git",
		".github":        ".github",
	}
	key := ShortestKey(data)
	fmt.Println(key)
	// Output: .git
}

func ExampleConfigFileGlob() {
	// This lets you override the default config file name for the entire tool.
	// This is likely going to be only useful if you're in a closed Enterprise
	// ecosystem.
	os.Setenv("COMMON_CONFIG_GLOB", ".commonreporc")
	glob := ConfigFileGlob()
	fmt.Println(glob)
	// Output: .commonreporc
}
