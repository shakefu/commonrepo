package commonrepo

import (
	"fmt"
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	. "github.com/onsi/gomega"
	goblin "github.com/shakefu/goblin"
)

func TestCommonRepo(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("GetLocalRepo", func() {
		g.It("should work", func() {
			repo, err := GetLocalRepo()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
		})
	})

	g.Describe("Config", func() {
		g.Describe("Unmarshal", func() {
			g.It("should work", func() {
				data := []byte(heredoc.Doc(`
				include:
				- '**/*'`))
				config := &Config{}
				err := config.Unmarshal(data)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Include).Should(Equal([]string{"**/*"}))
			})

			g.It("parses the schema document", func() {
				data, err := os.ReadFile("./testdata/fixtures/schema.yml")
				Expect(err).NotTo(HaveOccurred())
				config := &Config{}
				err = config.Unmarshal(data)
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("%+v\n", config)
				Expect(config.Include[0]).Should(Equal("**/*"))
			})
		})
	})
}
