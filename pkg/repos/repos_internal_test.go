package repos

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/shakefu/goblin"
)

func TestRepoInternal(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("Repo (internal)", func() {
		g.It("has a list of all our files", func() {
			repo := localRepo()
			Expect(repo.files).To(ContainElement("go.mod"))
			Expect(repo.files).To(ContainElement("go.sum"))
			Expect(repo.files).To(ContainElement(".commonrepo.yml"))
			Expect(repo.files).To(ContainElement("pkg/repos/repos.go"))
		})

		g.Describe("findConfig", func() {
			g.It("works", func() {
				repo := localRepo()
				path, err := repo.findConfig()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(path).To(Equal(".commonrepo.yml"))
			})

			g.It("lets you search a different path", func() {
				repo := localRepo()
				path, err := repo.findConfig("**/fixtures/.commonrepo.{yml,yaml}")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(path).To(Equal("testdata/fixtures/.commonrepo.yaml"))
			})

			g.It("sorts by shortest match", func() {
				repo := localRepo()
				path, err := repo.findConfig("**commonrepo.{yml,yaml}")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(path).To(Equal(".commonrepo.yml"))
			})
		})

		g.Describe("readConfig", func() {
			g.It("works", func() {
				repo := localRepo()
				data, err := repo.readConfig()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(data).ShouldNot(BeNil())
				Expect(data).To(HavePrefix("# CommonRepo"))
			})

			g.It("loads alternate paths", func() {
				repo := localRepo()
				path, err := repo.readConfig("**/fixtures/.commonrepo.{yml,yaml}")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(path).To(HavePrefix("include:"))
			})
		})

		g.Describe("LoadConfig", func() {
			g.It("works", func() {
				repo := localRepo()
				config, err := repo.LoadConfig()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config).ShouldNot(BeNil())
				// TODO: Make these tests stable at some point
				Expect(config.Include).To(HaveLen(0))
				Expect(config.Exclude).To(HaveLen(0))
			})

			g.It("loads alternate paths", func() {
				repo := localRepo()
				config, err := repo.LoadConfig("**/fixtures/.commonrepo.{yml,yaml}")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(config).ShouldNot(BeNil())
				Expect(config.Include).To(HaveLen(1))
				Expect(config.Exclude).To(HaveLen(1))
			})
		})
	})
}

// localRepo returns the local repo or raises an error.
//
// This is a reimplementation from testutil to prevent import cycle.
func localRepo() (repo *Repo) {
	repo, err := GetLocalRepo()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(repo).ShouldNot(BeNil())
	return
}

func BenchmarkRepoList(b *testing.B) {
	repo, err := GetLocalRepo()
	if err != nil {
		b.Fatal(err)
	}

	var files []string
	for n := 0; n < b.N; n++ {
		files, err = repo.list()
		if err != nil {
			b.Fatal(err)
		}
	}

	if len(files) < 1 {
		b.Fatal("no files")
	}
}
