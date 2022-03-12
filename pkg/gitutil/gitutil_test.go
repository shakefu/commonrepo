package gitutil

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	goblin "github.com/shakefu/goblin"
)

func TestGitUtil(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("gitutil", func() {
		g.Describe("FindLocalRepoPath", func() {
			g.It("should work", func() {
				repo, err := FindLocalRepoPath()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(repo).ShouldNot(Equal(""))
				Expect(filepath.Base(repo)).To(Equal("commonrepo"))
			})
		})

		g.Describe("DetectGitPath", func() {
			g.It("should work", func() {
				path, err := DetectGitPath(".")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(path).ShouldNot(Equal(""))
				Expect(filepath.Base(path)).To(Equal("commonrepo"))
			})
		})

		g.Describe("DefaultRef", func() {
			g.It("should work", func() {
				repo, err := FindLocalRepoPath()
				Expect(err).ShouldNot(HaveOccurred())

				ref, err := DefaultRef(repo)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ref).To(HavePrefix("refs/heads/"))
			})
		})

		g.Describe("LocalBranch", func() {
			g.It("works", func() {
				ref, err := LocalBranch()
				Expect(err).ToNot(HaveOccurred())
				// We don't know the branch name, but we know it's a branch and
				// so it shouldn't be an empty name... mostly an exercise test
				Expect(len(ref)).To(BeNumerically(">=", 3))
			})
		})

		g.Describe("RemoteDefaultBranch", func() {
			g.It("works", func() {
				ref, err := RemoteDefaultBranch()
				Expect(err).ToNot(HaveOccurred())
				Expect(ref.String()).To(Equal("main"))
			})
		})

		g.Describe("FindRef", func() {
			g.It("should work", func() {
				repo, err := FindLocalRepoPath()
				Expect(err).ShouldNot(HaveOccurred())
				ref, err := FindRef(repo, "main")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ref).To(HavePrefix("refs/heads/"))
			})

			g.It("does something with an empty ref", func() {
				repo, err := FindLocalRepoPath()
				Expect(err).ShouldNot(HaveOccurred())
				ref, err := FindRef(repo, "")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ref).To(HavePrefix("refs/heads/"))
			})
		})

		g.Describe("ApplyInsteadOf", func() {
			g.It("works", func() {
				// Exercise test, it doesn't really do anything testable
				url, err := ApplyInsteadOf("git@github.com/shakefu/commonrepo")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(url).To(HaveSuffix("shakefu/commonrepo"))
			})
		})

		g.Describe("(external)", func() {
			g.SkipIf(os.Getenv("SKIP_EXTERNAL") != "")
			g.SkipIf(os.Getenv("SSH_AUTH_SOCK") == "")
			g.Describe("DefaultRef", func() {
				g.It("returns the default head when it's not master", func() {
					url := "https://github.com/shakefu/commonrepo"
					ref, err := DefaultRef(url)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/heads/main"))
				})

				g.It("returns the default head when it is master", func() {
					url := "https://github.com/shakefu/bananafish"
					ref, err := DefaultRef(url)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/heads/master"))
				})

				g.It("works with ssh urls", func() {
					url := "git@github.com:shakefu/commonrepo.git"
					ref, err := DefaultRef(url)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/heads/main"))
				})
			})

			g.Describe("FindRef", func() {
				g.It("works with other refs", func() {
					url := "git@github.com:shakefu/humbledb.git"
					ref, err := FindRef(url, "6.0.0")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/tags/6.0.0"))
				})

				g.It("returns the default branch if it can't find the ref", func() {
					url := "git@github.com:shakefu/humbledb.git"
					ref, err := FindRef(url, "badtag")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/heads/master"))
				})

				g.It("returns the default branch for an empty ref", func() {
					url := "git@github.com:shakefu/humbledb.git"
					ref, err := FindRef(url, "")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ref).To(BeEquivalentTo("refs/heads/master"))
				})
			})
		})
	})
}
