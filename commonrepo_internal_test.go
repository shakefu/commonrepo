package commonrepo

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"

	// . "github.com/shakefu/commonrepo"
	"github.com/shakefu/commonrepo/pkg/config"
	"github.com/shakefu/commonrepo/pkg/files"
	"github.com/shakefu/commonrepo/pkg/gitutil"
	"github.com/shakefu/commonrepo/pkg/repos"

	. "github.com/onsi/gomega"
	"github.com/shakefu/goblin"
	// . "github.com/shakefu/commonrepo/internal/testutil"
)

func TestCommonRepoInternal(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("commonrepo", func() {
		g.It("clones a single source", func() {
			cr, err := NewFrom("testdata/fixtures/single_source.yml", ".")
			Expect(err).ToNot(HaveOccurred())
			Expect(cr).ToNot(BeNil())
		})

		g.It("works as a stringer", func() {
			cr, _ := NewFrom("testdata/fixtures/single_source.yml", ".")
			Expect(cr.String()).To(HavePrefix("./testdata/fixtures/single_source.yml@"))
		})

		g.Describe("New", func() {
			g.It("works against the current repository", func() {
				path, err := gitutil.FindLocalRepoPath()
				Expect(err).ToNot(HaveOccurred())

				cr, err := New(path)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				// TODO: Make these tests stable at some point
				// Expect(cr.config.Include).To(ContainElement("testdata/fixtures/.commonrepo.yml"))
			})

			g.It("works with a dot path", func() {
				cr, err := New(".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				// TODO: Make these tests stable at some point
				// Expect(cr.config.Include).To(ContainElement("testdata/fixtures/.commonrepo.yml"))
			})
		})

		g.Describe("NewFrom", func() {
			g.It("errors with a bad pattern", func() {
				_, err := NewFrom("badconfig.yaml", ".")
				Expect(err).To(MatchError("no config file found"))
			})

			g.It("finds configs in the subpath", func() {
				cr, err := NewFrom("**/fixtures/.commonrepo.{yml,yaml}", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(cr.config.Include).To(ContainElement("**"))
			})

			g.It("works with different configs", func() {
				cr, err := NewFrom("testdata/fixtures/multi_source.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(cr.config.Upstream).To(HaveLen(2))
			})

			g.It("works with different configs 2", func() {
				cr, err := NewFrom("testdata/fixtures/single_source.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(cr.config.Upstream).To(HaveLen(1))
			})
		})

		g.Describe("NewFromRename", func() {
			g.It("works on the local repo", func() {
				repo, err := repos.GetLocalRepo()
				Expect(err).ToNot(HaveOccurred())

				cr, err := NewFromRename(repo, []config.Rename{})
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				// TODO: Make these tests stable at some point
				// Expect(cr.config.Include).To(Equal([]string{"testdata/fixtures/.commonrepo.yml"}))
			})
		})

		g.Describe("LoadUpstreams", func() {
			g.It("errors if the recursion goes too deep", func() {
				cr, err := NewFrom("testdata/fixtures/single_source.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(0)
				Expect(err).To(MatchError("maximum recursion depth reached"))
			})

			g.It("works with deep local nonsense", func() {
				cr, err := NewFrom("**/fixtures/local/deep.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(4)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		g.Describe("FlattenUpstreams", func() {
			g.It("works with a single repo", func() {
				cr, err := NewFrom("testdata/fixtures/local/single.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(4)
				Expect(err).ToNot(HaveOccurred())
				upstreams := cr.FlattenUpstreams()
				Expect(len(upstreams)).To(Equal(2))
				Expect(upstreams[1].config.Include).To(
					Equal([]string{"testdata/fixtures/local/single.yml"}))
				Expect(upstreams[0].config.Include).To(
					Equal([]string{"**.commonrepo.y*ml"}))
			})

			g.It("works with multi repo", func() {
				cr, err := NewFrom("testdata/fixtures/local/multi.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(4)
				Expect(err).ToNot(HaveOccurred())
				upstreams := cr.FlattenUpstreams()
				Expect(len(upstreams)).To(Equal(4))
				// Multi config
				Expect(upstreams[3].config.Include).To(
					Equal([]string{"testdata/fixtures/local/multi.yml"}))
				// - Inherits Single
				Expect(upstreams[2].config.Include).To(
					Equal([]string{"testdata/fixtures/local/single.yml"}))
				//     - Single inherits Empty
				Expect(upstreams[1].config.Include).To(
					Equal([]string{"**.commonrepo.y*ml"}))
				// - Inherits Fixtures
				// TODO: Make these tests stable at some point
				// Expect(upstreams[0].config.Include).To(
				// Equal([]string{"testdata/fixtures/.commonrepo.yml"}))
			})

			g.It("works with deep repo", func() {
				cr, err := NewFrom("testdata/fixtures/local/deep.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(4)
				Expect(err).ToNot(HaveOccurred())
				upstreams := cr.FlattenUpstreams()
				Expect(len(upstreams)).To(Equal(5))
				// Deep config
				Expect(upstreams[4].config.Include).To(
					Equal([]string{"testdata/fixtures/local/deep.yml"}))
				// - Inherits Multi
				Expect(upstreams[3].config.Include).To(
					Equal([]string{"testdata/fixtures/local/multi.yml"}))
				//     - Multi inherits Single
				Expect(upstreams[2].config.Include).To(
					Equal([]string{"testdata/fixtures/local/single.yml"}))
				//         - Single inherits Empty
				Expect(upstreams[1].config.Include).To(
					Equal([]string{"**.commonrepo.y*ml"}))
				//     - Multi inherits Fixtures
				// TODO: Make this test stable at some point
				// Expect(upstreams[0].config.Include).To(
				// Equal([]string{"testdata/fixtures/.commonrepo.yml"}))
			})

			g.It("appends includes, excludes and renames", func() {
				cr, err := NewFrom("testdata/fixtures/local/append.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.LoadUpstreams(4)
				Expect(err).ToNot(HaveOccurred())
				upstreams := cr.FlattenUpstreams()
				Expect(len(upstreams)).To(Equal(2))
				// Append config
				append := upstreams[1]
				Expect(append.config.Include).To(Equal([]string{}))
				Expect(append.config.Exclude).To(Equal([]string{}))
				Expect(append.config.Rename).To(Equal([]config.Rename{}))
				// - Inherits local repo, renames .commonrepo.yml out of the way
				// to make it empty, but it gets the appended config
				empty := upstreams[0]
				Expect(empty.config.Include).To(Equal([]string{"*.md"}))
				Expect(empty.config.Exclude).To(Equal([]string{"action.*"}))
				Expect(empty.config.Rename).To(HaveLen(1))
			})
		})

		g.Describe("Init", func() {
			g.It("works", func() {
				cr, err := NewFrom("testdata/fixtures/local/single.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		g.Describe("Composite", func() {
			g.It("works", func() {
				cr, err := NewFrom("testdata/fixtures/local/single.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
				composite := cr.Composite()
				Expect(composite).ToNot(BeNil())
				keys := Keys(composite)
				Expect(keys).To(Equal([]string{
					"testdata/out/.not_commonrepo.yml",
					"testdata/out/empty.yml",
					"testdata/out/single.yml"}))
			})

			g.It("combines all template vars", func() {
				cr, err := NewFrom("testdata/fixtures/templating.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
				composite := cr.Composite()
				Expect(composite).ToNot(BeNil())
				keys := Keys(composite)
				Expect(keys).To(Equal([]string{"templated.yml"}))
				target := composite["templated.yml"]
				var buf = new(bytes.Buffer)
				err = target.Write(buf)
				Expect(err).ToNot(HaveOccurred())
				Expect(buf.String()).To(
					Equal("project: commonrepo\nversion: 1.0.0\ntemplated: true\n"))
			})
		})

		g.Describe("WriteFS", func() {
			g.It("works", func() {
				cr, err := NewFrom("testdata/fixtures/local/single.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
				composite := cr.Composite()
				Expect(composite).ToNot(BeNil())
				fs := memfs.New()
				err = composite.WriteFS(fs, "/")
				Expect(err).ToNot(HaveOccurred())
				found, err := files.List(fs)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(Equal([]string{
					"testdata/out/.not_commonrepo.yml",
					"testdata/out/empty.yml",
					"testdata/out/single.yml",
				}))
			})

			g.It("works with the actual filesystem", func() {
				cr, err := NewFrom("testdata/fixtures/local/single.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
				composite := cr.Composite()
				Expect(composite).ToNot(BeNil())
				fs := osfs.New("/tmp/test/commonrepo")
				defer os.RemoveAll("/tmp/test/commonrepo")
				err = composite.WriteFS(fs, "/")
				Expect(err).ToNot(HaveOccurred())
				found, err := files.List(fs)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(Equal([]string{
					"testdata/out/.not_commonrepo.yml",
					"testdata/out/empty.yml",
					"testdata/out/single.yml",
				}))
			})
		})

		g.Describe("Write", func() {
			var repoPath string
			var outPath string
			var err error

			g.Before(func() {
				repoPath, err = gitutil.FindLocalRepoPath()
				if err != nil {
					g.FailNow()
				}
				outPath = filepath.Join(repoPath, "testdata/out")
			})

			g.It("works", func() {
				cr, err := NewFrom("testdata/fixtures/local/write.yml", ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				err = cr.Init()
				Expect(err).ToNot(HaveOccurred())
				composite := cr.Composite()
				Expect(composite).ToNot(BeNil())
				defer os.RemoveAll(outPath)
				err = composite.Write()
				Expect(err).ToNot(HaveOccurred())
				fs := osfs.New(outPath)
				Expect(files.List(fs)).To(Equal([]string{"write.yml"}))
			})
		})

		g.Describe("(external)", func() {
			g.SkipIf(os.Getenv("SKIP_EXTERNAL") != "")
			g.Describe("LoadUpstreams", func() {
				upstreamBranchRef := "main"
				g.It("works with a single source", func() {
					cr, err := NewFrom("testdata/fixtures/single_source.yml", ".")
					Expect(err).ToNot(HaveOccurred())
					Expect(cr).ToNot(BeNil())
					err = cr.LoadUpstreams(3)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.upstreams).To(HaveLen(1))
					Expect(cr.upstreams[0].repo.URL).To(Equal("https://github.com/shakefu/humbledb"))
				})

				g.It("works with multiple sources", func() {
					cr, err := NewFrom("testdata/fixtures/multi_source.yml", ".")
					Expect(err).ToNot(HaveOccurred())
					Expect(cr).ToNot(BeNil())
					err = cr.LoadUpstreams(3)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.upstreams).To(HaveLen(2))
					Expect(cr.upstreams[0].repo.URL).To(Equal("https://github.com/shakefu/humbledb"))
					Expect(cr.upstreams[1].repo.URL).To(Equal("https://github.com/shakefu/commonrepo"))
					Expect(cr.upstreams[1].repo.Ref).To(Equal(upstreamBranchRef))
					// TODO: Make this test stable/use a config that's not empty
					// Expect(cr.upstreams[1].config.Include).To(
					//  Equal([]string{"testdata/fixtures/.commonrepo.yml"}))
				})

				g.It("works with deep sources and renames", func() {
					cr, err := NewFrom("testdata/fixtures/deep_source.yml", ".")
					Expect(err).ToNot(HaveOccurred())
					Expect(cr).ToNot(BeNil())
					err = cr.LoadUpstreams(3)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.upstreams).To(HaveLen(1))
					u := cr.upstreams[0]
					Expect(u.repo.URL).To(Equal("https://github.com/shakefu/commonrepo"))
					Expect(u.repo.Ref).To(Equal(upstreamBranchRef))
					// We're overriding the config with the rename, so this
					// should be empty matching single_source
					Expect(u.config.Include).To(HaveLen(0))
					Expect(u.config.Upstream).To(HaveLen(1))
					Expect(u.upstreams).To(HaveLen(1))
					u = u.upstreams[0]
					Expect(u.repo.URL).To(Equal("https://github.com/shakefu/humbledb"))
				})
			})
		})
	})
}

func Keys(m map[string]repos.Target) []string {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}
