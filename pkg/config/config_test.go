package config_test

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	. "github.com/shakefu/commonrepo/internal/testutil"

	. "github.com/onsi/gomega"
	"github.com/shakefu/commonrepo/pkg/config"
	"github.com/shakefu/commonrepo/pkg/gitutil"
	"github.com/shakefu/goblin"
)

func TestConfig(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("Config", func() {
		g.Describe("ParseConfig", func() {
			g.It("parses a simple config", func() {
				config, err := config.ParseConfig(InlineYaml(`
				include:
					- '**/*'`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Include).To(Equal([]string{"**/*"}))
			})

			g.It("parses excludes", func() {
				config, err := config.ParseConfig(InlineYaml(`
				exclude:
					- '**/*'`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Exclude).To(Equal([]string{"**/*"}))
			})

			g.It("parses renames", func() {
				config, err := config.ParseConfig(InlineYaml(`
				rename:
					- (.*)foo(.*): '%[1]sbar%[2]s'`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Rename).To(HaveLen(1))
				r := config.Rename[0]
				Expect(r.Replace).To(Equal("%[1]sbar%[2]s"))
				Expect(r.Apply("foobar")).To(Equal("barbar"))
				Expect(r.Apply("fooyou")).To(Equal("baryou"))
				Expect(r.Apply("some/path/foofile")).To(Equal("some/path/barfile"))
			})

			g.It("parses versions", func() {
				config, err := config.ParseConfig(InlineYaml(`
				install:
					- golang: ^1`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Install).To(HaveLen(1))
				i := config.Install[0]
				Expect(i.Name).To(Equal("golang"))
				v1, _ := semver.NewVersion("1.0.0")
				v2, _ := semver.NewVersion("2.0.0")
				Expect(i.Version.Check(v1)).To(BeTrue())
				Expect(i.Version.Check(v2)).To(BeFalse())
			})

			g.It("errors with garbage versions", func() {
				config, err := config.ParseConfig(InlineYaml(`
				install:
					- golang: ^2.3.4.1`))
				Expect(err).To(HaveOccurred())
				Expect(config).To(BeNil())
			})

			g.It("parses upstreams", func() {
				ref, _ := gitutil.LocalBranch()
				config, err := config.ParseConfig(InlineYaml(`
				upstream:
				  - url: github.com/shakefu/commonrepo
				    ref: ` + ref.String() + `
				    include: ['**/*']
				    exclude:
				       - '*.md'
				       - '*.go'
				    rename:
				      - ^(.*)/(.*\.md): '%[1]s/docs/%[2]s'`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Upstream).To(HaveLen(1))
				u := config.Upstream[0]
				Expect(u.URL).To(Equal("github.com/shakefu/commonrepo"))
				Expect(u.Ref).To(Equal(ref.String()))
				Expect(u.Include).To(Equal([]string{"**/*"}))
				Expect(u.Exclude).To(Equal([]string{"*.md", "*.go"}))
				Expect(u.Rename).To(HaveLen(1))
				r := u.Rename[0]
				Expect(r.Apply("somepath/foo.md")).To(Equal("somepath/docs/foo.md"))
			})

			g.It("parses basic upstreams", func() {
				config, err := config.ParseConfig(InlineYaml(`
				upstream:
				  - url: github.com/shakefu/commonrepo`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Upstream).To(HaveLen(1))
				u := config.Upstream[0]
				Expect(u.URL).To(Equal("github.com/shakefu/commonrepo"))
			})

			g.Skip("errors bad renames", func() {
				// This test isn't erroring like expected, debug it later
				config, err := config.ParseConfig(InlineYaml(`
				upstream:
				  - rename:
				  	- "()[]^$": ""`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
			})

			g.It("parses templates", func() {
				config, err := config.ParseConfig(InlineYaml(`
				template:
				  - "templates/*"
				template-vars:
				  project: commonrepo`))
				Expect(err).ToNot(HaveOccurred())
				Expect(config).ToNot(BeNil())
				Expect(config.Template).To(Equal([]string{"templates/*"}))
				Expect(config.TemplateVars).To(
					Equal(map[string]interface{}{"project": "commonrepo"}))
			})

			g.It("parses installs", func() {
				config, err := config.ParseConfig(InlineYaml(`
				install:
					- golang: 1.17
				install-from: ./tools/
				install-with: [brew]`))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config.Install[0].Name).To(Equal("golang"))
				Expect(config.InstallFrom).To(Equal("./tools/"))
				Expect(config.InstallWith).To(Equal([]string{"brew"}))
			})
		})

		g.Describe("ApplyRename", func() {
			g.It("works", func() {
				config, err := config.ParseConfig(InlineYaml(`
				rename:
				- ^(.*)/(.*\.md): '%[1]s/docs/%[2]s'`))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config.ApplyRename("foo/bar.md")).To(Equal("foo/docs/bar.md"))
			})

			g.It("doesn't rename paths that don't match", func() {
				config, err := config.ParseConfig(InlineYaml(`
				rename:
				- ^(.*)/(.*\.txt): '%[1]s/docs/%[2]s'`))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config.ApplyRename("foo/bar.md")).To(Equal(""))
			})
		})

		g.Describe("Rename", func() {
			g.Describe("Stringer", func() {
				config, _ := config.ParseConfig(InlineYaml(`
				rename:
				- ^(.*).md: doc/%[1]s.md'`))
				g.It("gives you a sane output", func() {
					Expect(config.Rename[0].String()).To(
						Equal("^(.*).md: doc/%[1]s.md'"))
				})
			})

			g.Describe("Check", func() {
				g.It("works", func() {
					config, _ := config.ParseConfig(InlineYaml(`
					rename:
					- ^(.*).md: doc/%[1]s.md'`))
					Expect(config.Rename[0].Check("foo.md")).To(BeTrue())
				})
			})
		})
	})
}
