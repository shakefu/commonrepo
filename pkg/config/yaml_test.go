// Package config_test tests the config
package config_test

import (
	"testing"

	. "github.com/shakefu/commonrepo/internal/testutil"

	. "github.com/onsi/gomega"
	"github.com/shakefu/commonrepo/pkg/config"
	"github.com/shakefu/commonrepo/pkg/gitutil"
	"github.com/shakefu/goblin"
)

func TestYamlConfig(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("yaml", func() {
		g.Describe("YamlConfig", func() {
			g.It("holds the raw yaml", func() {
				data := InlineYaml(`
				include:
				- "testdata/fixtures/.commonrepo.yml"`)
				config := &config.YamlConfig{}
				err := config.Unmarshal(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(config.Raw()).To(Equal(string(data)))
			})

			g.Describe("Unmarshal", func() {
				// These tests are lower level and more verbose than the
				// YamlParse tests below which exercise the same code paths
				g.It("works", func() {
					data := InlineYaml(`
						include:
						- '**/*'`)
					config := &config.YamlConfig{}
					err := config.Unmarshal(data)
					Expect(err).NotTo(HaveOccurred())
					Expect(config.Include).Should(Equal([]string{"**/*"}))
				})
			})
		})

		g.Describe("YamlParse", func() {
			g.It("parses excludes", func() {
				conf, err := config.YamlParse(InlineYaml(`
					exclude:
					- '**/*'`))
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Exclude).Should(Equal([]string{"**/*"}))
			})

			g.It("parses upstream", func() {
				ref, _ := gitutil.LocalBranch()
				conf, err := config.YamlParse(InlineYaml(`
					upstream:
					- url: https://github.com/shakefu/commonrepo
					  ref: ` + ref.String()))
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Upstream).To(HaveLen(1))
				Expect(conf.Upstream[0].Ref).Should(Equal(ref.String()))
			})

			g.It("parses installs", func() {
				config, err := config.YamlParse(InlineYaml(`
					install:
					  - golang: 1.17
					install-from: ./tools/
					install-with: [brew]
				`))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config.Install).To(
					Equal([]map[string]string{{"golang": "1.17"}}))
				Expect(config.InstallFrom).To(Equal("./tools/"))
				Expect(config.InstallWith).To(Equal([]string{"brew"}))
			})

			g.It("parses templates", func() {
				config, err := config.YamlParse(InlineYaml(`
					template:
					  - "templates/*"
					template-vars:
					  project: commonrepo`))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config.Template).To(Equal([]string{"templates/*"}))
				Expect(config.TemplateVars).To(
					Equal(map[string]interface{}{"project": "commonrepo"}))
			})
		})
	})
}
