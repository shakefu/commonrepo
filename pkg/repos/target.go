package repos

import (
	"fmt"
	"io"
	"os"
	"text/template"
)

// Target represents a single target file or template
type Target struct {
	Name string                 // Original file name
	Vars map[string]interface{} // Template variables, if it is a template
	repo *Repo                  // Source repo, for reading the file content
}

// String returns a Target as a string
func (targ *Target) String() string {
	if targ.Vars == nil || len(targ.Vars) == 0 {
		return fmt.Sprintf("<Repo.File:%s>", targ.Name)
	}
	return fmt.Sprintf("<Repo.Template:%s>", targ.Name)
}

// Stat returns the os.FileInfo for the target
func (targ *Target) Stat() (os.FileInfo, error) {
	return targ.repo.Stat(targ.Name)
}

// Write writes the target file to the given writer
func (targ *Target) Write(dest io.Writer) (err error) {
	// If there's no Vars it's not a template so it's a simple copy operation
	if targ.Vars == nil || len(targ.Vars) == 0 {
		return targ.CopyTo(dest)
	}

	// Otherwise we render out the template
	return targ.RenderTo(dest)
}

// CopyTo copies the given file to the given writer
func (targ *Target) CopyTo(dest io.Writer) (err error) {
	// Open the file
	var file io.Reader
	if file, err = targ.repo.Open(targ.Name); err != nil {
		return
	}
	// Get its info and size
	var info os.FileInfo
	if info, err = targ.repo.Stat(targ.Name); err != nil {
		return
	}
	size := info.Size()
	// Copy to the destination
	var written int64
	if written, err = io.Copy(dest, file); err != nil {
		return
	}
	// And make sure we got the whole thing
	if written != size {
		err = fmt.Errorf("wrote %d bytes, expected %d", written, size)
	}
	return
}

// RenderTo renders the template with the current Vars
func (targ *Target) RenderTo(dest io.Writer) (err error) {
	// TBD: Test passing repo.fs and globbing, it might be faster
	var data []byte
	if data, err = targ.repo.ReadFile(targ.Name); err != nil {
		return
	}
	// Try to parse the template
	// TBD: Should there be a set of standard functions?
	// e.g. https://github.com/Masterminds/sprig
	var tmpl *template.Template
	tmpl = template.New(targ.Name).Option("missingkey=error")
	if tmpl, err = tmpl.Parse(string(data)); err != nil {
		return
	}
	// Render it out to our destination file
	err = tmpl.Execute(dest, targ.Vars)
	return
}
