package repos_test

import (
	"os"
	"testing"

	"github.com/shakefu/commonrepo/pkg/common"
	"github.com/shakefu/commonrepo/pkg/config"
	. "github.com/shakefu/commonrepo/pkg/repos"

	. "github.com/onsi/gomega"
	. "github.com/shakefu/commonrepo/internal/testutil"
	"github.com/shakefu/goblin"
)

func TestRepo(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	// Helper data reused throughout tests
	// const URL = "git@github.com:shakefu/commonrepo.git"
	var repo *Repo
	var err error

	g.Describe("repos", func() {
		g.Describe("Repo", func() {
			g.It("requires a URL", func() {
				repo, err := New("")
				Expect(err).To(MatchError("repository not found"))
				Expect(repo).To(BeNil())
			})

			g.It("clones the local repo", func() {
				// repo, err = New(URL, "refs/heads/main")
				repo, err = GetLocalRepo()
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

			g.Describe("Glob", func() {
				g.It("globs nicely", func() {
					files, err := repo.Glob("go.*")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"go.mod", "go.sum"}))
				})

				g.It("globs specific filenames", func() {
					files, err := repo.Glob("README.md")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"README.md"}))
				})

				g.It("globs subpaths", func() {
					files, err := repo.Glob("*/*/schema.yml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"testdata/fixtures/schema.yml"}))
				})

				g.It("globs more subpaths", func() {
					files, err := repo.Glob("*/*/schema.yml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"testdata/fixtures/schema.yml"}))
				})

				g.It("globs all the subpaths to a filename", func() {
					files, err := repo.Glob("**/schema.yml")
					// files, err := repo.Glob("*/*/*/*")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"testdata/fixtures/schema.yml"}))
				})

				g.It("globs all the subpaths dir matching to file", func() {
					files, err := repo.Glob("*/*/*/deep.yml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(files).To(Equal([]string{"testdata/fixtures/local/deep.yml"}))
				})
			})

			g.Describe("ReadFile", func() {
				g.It("reads nicely", func() {
					data, err := repo.ReadFile("test/sentinel")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(data)).To(Equal("echo sentinel\n"))
				})

				g.It("throws some error if a file doesn't exist", func() {
					_, err := repo.ReadFile("test/doesntexist")
					Expect(err).Should(HaveOccurred())
				})
			})

			g.Describe("ApplyIncludes", func() {
				var cfg *config.Config

				g.Before(func() {
					if repo, err = GetLocalRepo(); err != nil {
						g.FailNow()
					}
					if cfg, err = config.ParseConfig(InlineYaml(`
						include:
						  - testdata/fixtures/local/*d*.yml`)); err != nil {
						g.FailNow()
					}
				})

				g.It("works", func() {
					Expect(err).ToNot(HaveOccurred())
					repo.ApplyIncludes(cfg.Include)
					Expect(SortTargetNames(repo.Targets())).To(Equal([]string{
						"testdata/fixtures/local/append.yml",
						"testdata/fixtures/local/deep.yml",
					}))
				})

				g.It("shortcuts to return an empty list if there's no includes", func() {
					found, err := repo.ApplyIncludes([]string{})
					Expect(err).ToNot(HaveOccurred())
					Expect(SortTargetNames(found)).To(Equal([]string{}))
				})
			})

			g.Describe("ApplyExcludes", func() {
				var cfg *config.Config

				g.Before(func() {
					if repo, err = GetLocalRepo(); err != nil {
						g.FailNow()
					}
					if cfg, err = config.ParseConfig(InlineYaml(`
						exclude:
						  - testdata/fixtures/local/deep.yml`)); err != nil {
						g.FailNow()
					}
					_, err = repo.ApplyIncludes([]string{"testdata/fixtures/local/*d*.yml"})
				})

				g.It("works", func() {
					found, err := repo.ApplyExcludes(cfg.Exclude)
					Expect(err).ToNot(HaveOccurred())
					Expect(SortTargetNames(found)).To(Equal([]string{
						"testdata/fixtures/local/append.yml",
					}))
				})

				g.It("fast fails if excludes are empty", func() {
					found, err := repo.ApplyExcludes([]string{})
					Expect(err).ToNot(HaveOccurred())
					Expect(SortTargetNames(found)).To(Equal([]string{
						"testdata/fixtures/local/append.yml",
					}))
				})
			})

			g.Describe("ApplyTemplates", func() {
				var cfg *config.Config
				var target = "testdata/fixtures/local/deep.yml"

				g.Before(func() {
					if repo, err = GetLocalRepo(); err != nil {
						g.FailNow()
					}
					if cfg, err = config.ParseConfig(InlineYaml(`
						template:
						  - testdata/fixtures/local/deep.yml
						template-vars:
						  templated: true`)); err != nil {
						g.FailNow()
					}
				})

				g.BeforeEach(func() {
					repo.ResetTargets()
					if _, err = repo.ApplyExcludes([]string{"**"}); err != nil {
						g.FailNow()
					}
				})

				g.It("works", func() {
					err := repo.ApplyTemplates(cfg.Template, cfg.TemplateVars)
					Expect(err).ToNot(HaveOccurred())
					targets := repo.Targets()
					Expect(SortTargetNames(targets)).To(Equal([]string{target}))
					Expect(targets[target].Vars).To(
						Equal(map[string]interface{}{"templated": true}))
				})

				g.It("fast fails if templates are empty", func() {
					err := repo.ApplyTemplates([]string{}, cfg.TemplateVars)
					Expect(err).ToNot(HaveOccurred())
					targets := repo.Targets()
					Expect(SortTargetNames(targets)).To(Equal([]string{}))
				})
			})

			g.Describe("Renames", func() {
				var cfg *config.Config

				g.Before(func() {
					if repo, err = GetLocalRepo(); err != nil {
						g.FailNow()
					}
					if cfg, err = config.ParseConfig(InlineYaml(`
						rename:
							- "^(README.md)": "rename-%[1]s"
							- "^(LICENSE)": "rename-%[1]s"
					`)); err != nil {
						g.FailNow()
					}
				})
				g.AfterEach(func() { repo.ResetTargets() })

				g.It("returns the existing file tree without any args", func() {
					renamed := repo.Targets()
					Expect(renamed["README.md"].Name).To(Equal("README.md"))
				})

				g.It("applies a rename from config", func() {
					renamed := repo.ApplyRenames(cfg.Rename)
					Expect(renamed["rename-README.md"].Name).To(Equal("README.md"))
				})

				g.It("ResetRenames resets things", func() {
					renamed := repo.Targets()
					Expect(renamed["README.md"].Name).To(Equal("README.md"))
				})

				g.It("removes the original entry", func() {
					renamed := repo.ApplyRenames(cfg.Rename)
					Expect(renamed["rename-README.md"].Name).To(Equal("README.md"))
					Expect(renamed["rename-LICENSE"].Name).To(Equal("LICENSE"))
					Expect(renamed["README.md"].Name).To(Equal(""))
					Expect(renamed["LICENSE"].Name).To(Equal(""))
				})

				g.It("lets you hard rename a commonrepo file", func() {
					data, err := os.ReadFile("../../testdata/fixtures/deep_source.yml")
					Expect(err).ShouldNot(HaveOccurred())
					conf, err := config.ParseConfig(data)
					Expect(err).ShouldNot(HaveOccurred())
					renamed := repo.ApplyRenames(conf.Upstream[0].Rename)
					// Expect(renamed).To(Equal(map[string]string{}))
					Expect(renamed[".commonrepo.yml"].Name).To(Equal("testdata/fixtures/single_source.yml"))
				})
			})

			g.Describe("GlobRenamed", func() {
				g.AfterEach(func() { repo.ResetTargets() })
				g.It("works", func() {
					renamed := repo.Targets()
					// Manually modifying the rename map to skip making a config
					renamed["renamed-README.md"] = Target{Name: "README.md"}
					matches, err := repo.GlobTargets("renamed-**")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(matches["renamed-README.md"].Name).To(Equal("README.md"))
				})

				g.It("finds the commonrepo.yml", func() {
					matches, err := repo.GlobTargets(common.ConfigFileGlob())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(matches[".commonrepo.yml"].Name).To(Equal(".commonrepo.yml"))
				})
			})
		})

		g.Describe("(external)", func() {
			g.SkipIf(os.Getenv("SKIP_EXTERNAL") != "")
			g.SkipIf(os.Getenv("SSH_AUTH_SOCK") == "")
			g.Describe("Repo", func() {
				g.It("clones ssh repositories", func() {
					url := "git@github.com:shakefu/commonrepo.git"
					repo, err := New(url)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(repo).ShouldNot(BeNil())
					Expect(repo.Stat("LICENSE")).ShouldNot(BeNil())
				})

				g.It("clones private repositories", func() {
					url := "git@github.com:shakefu/commonrepo-private.git"
					repo, err := New(url)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(repo).ShouldNot(BeNil())
					Expect(repo.Stat("LICENSE")).ShouldNot(BeNil())
				})
			})
		})

		g.Describe("GetLocalRepo", func() {
			g.It("should work", func() {
				repo, err := GetLocalRepo()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(repo).ShouldNot(BeNil())
			})
		})
	})
}
