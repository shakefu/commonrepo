package memrepo

import (
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/jinzhu/copier"
	. "github.com/onsi/gomega"
	"github.com/shakefu/commonrepo/gitutil"
	goblin "github.com/shakefu/goblin"
)

func TestMemGit(t *testing.T) {
	// Disable logging output for the memgit logger
	log.SetLevel("fatal")

	// Initialize the Goblin test suite
	g := goblin.Goblin(t)

	// Helper data reused throughout tests
	// const URL = "git@github.com:shakefu/commonrepo.git"
	var repo Repo
	var err error

	// For faster tests we reference the local repository so we don't have to do
	// any shenanigans over the network
	URL, err := gitutil.FindLocalRepoPath()
	if err != nil {
		g.Fatalf("Could not determine local repository root: %v", err)
	}

	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("Repo", func() {
		g.It("requires a URL", func() {
			repo, err := NewRepo()
			Expect(err).To(MatchError("URL field is required"))
			Expect(repo).To(BeNil())
		})

		g.It("clones the local repo", func() {
			repo, err = NewRepo(Dict{"url": URL})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
			Expect(repo.Stat("LICENSE")).ShouldNot(BeNil())
			stat, _ := repo.Stat("LICENSE")
			Expect(stat.Name()).To(Equal("LICENSE"))
			Expect(stat.Size()).To(BeEquivalentTo(11357))
			Expect(stat.Mode()).To(BeEquivalentTo(420))
		})

		g.It("allows consistent state", func() {
			Expect(repo).ShouldNot(BeNil())
		})

		g.It("globs nicely", func() {
			files, err := repo.Glob("go.*")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(files).To(Equal([]string{"go.mod", "go.sum"}))
		})

		g.It("reads nicely", func() {
			data, err := repo.ReadFile("test/sentinel")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(data)).To(Equal("echo sentinel\n"))
		})

		g.It("clones remote repositories", func() {
			url := "https://github.com/shakefu/commonrepo"
			repo, err := NewRepo(Dict{"url": url})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
			Expect(repo.Stat("LICENSE")).ShouldNot(BeNil())
		})
	})

	g.Describe("makeGitConfig", func() {
		defs := Dict{
			"insecureskiptls": true,
			"referencename":   "main",
		}

		// Boilerplate helper so we don't have to retype/update all the git
		// config defaults with every test
		withDefaults := func(d Dict) Dict {
			t := Dict{}
			copyDictLower(t, gitConfigDefaults)
			copyDictLower(t, d)
			return t
		}

		g.It("takes a string url", func() {
			opts, err := makeGitConfig(URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(withDefaults(Dict{"url": URL})))
		})

		g.It("ignores nils", func() {
			opts, err := makeGitConfig(nil, URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(withDefaults(Dict{"url": URL})))
		})

		g.It("merges a string url into defaults", func() {
			opts, err := makeGitConfig(defs, URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(withDefaults(Dict{
				"url":             URL,
				"insecureskiptls": true,
				"referencename":   "main",
			})))
		})

		g.It("merges a map value into defaults", func() {
			args := Dict{
				"url":             URL,
				"insecureskiptls": false,
			}
			opts, err := makeGitConfig(defs, args)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(withDefaults(Dict{
				"url":             URL,
				"insecureskiptls": false,
				"referencename":   "main",
			})))
			// Check that it didn't modify our defaults
			Expect(defs["insecureskiptls"]).To(Equal(true))
		})

		g.It("works with an empty map to provide defaults", func() {
			opts, err := makeGitConfig(make(map[string]interface{}), nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(gitConfigDefaults))
		})

		g.It("works with an empty Dict to provide defaults", func() {
			opts, err := makeGitConfig(Dict{}, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(gitConfigDefaults))
		})
	})

	g.Describe("makeGitCloneOptions", func() {
		defaultOpts := git.CloneOptions{
			URL:           "",
			ReferenceName: "refs/heads/main",
			SingleBranch:  true,
			Depth:         1,
		}

		g.It("gives us sensible defaults", func() {
			opts, err := makeGitCloneOptions([]Yaml{}...)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(&defaultOpts))
		})

		g.It("works with a single string argument", func() {
			var expected git.CloneOptions
			copier.Copy(&expected, defaultOpts)
			expected.URL = URL
			opts, err := makeGitCloneOptions(URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(&expected))
		})

		g.It("takes multiple configs", func() {
			var expected git.CloneOptions
			copier.Copy(&expected, defaultOpts)
			expected.URL = URL
			expected.InsecureSkipTLS = true
			expected.ReferenceName = "main"
			defs := Dict{
				"insecureskiptls": true,
				"referencename":   "main",
			}
			opts, err := makeGitCloneOptions(defs, URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(opts).To(Equal(&expected))
		})
	})
}
