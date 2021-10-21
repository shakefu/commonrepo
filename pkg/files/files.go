// Package files includes tools for finding and manipulating the repository files
package files

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/gammazero/deque"
	"github.com/go-git/go-billy/v5"
)

// UniqString creates an array of string with unique values.
//
// Borrowed from go-funk:
// https://github.com/thoas/go-funk/blob/v0.9.1/typesafe.go#L847
func UniqString(a []string) []string {
	var (
		length  = len(a)
		seen    = make(map[string]struct{}, length)
		results = make([]string, 0)
	)

	for i := 0; i < length; i++ {
		v := a[i]

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		results = append(results, v)
	}

	return results
}

// List files returns the list of all the file paths in a Filesystem
func List(filesystem billy.Filesystem) (files []string, err error) {
	// This function includes a commented out naive implementation of the stack
	// behavior... it's probably not worth it to use deque, but in theory it'll
	// be faster for very large repositories with many thousands of directories
	var more []fs.FileInfo
	var item fs.FileInfo
	// var dirs []string
	var dirs deque.Deque
	var found []string
	var dir string

	// Initailize with our root listing
	// dirs = []string{""}
	dirs.PushBack("")
	// for len(dirs) > 0 {
	for dirs.Len() > 0 {
		// Pop the top directory off the stack
		// dir, dirs = dirs[len(dirs)-1], dirs[:len(dirs)-1]
		dir = dirs.PopBack().(string)
		// List everything in there
		more, err = filesystem.ReadDir(dir)
		if err != nil {
			return
		}
		// And process it
		for _, item = range more {
			if item.IsDir() {
				dirs.PushBack(filepath.Join(dir, item.Name()))
			} else {
				found = append(found, filepath.Join(dir, item.Name()))
			}
		}
	}
	// Ensure our files our sorted for sanity's sake
	sort.Strings(found)
	// Make sure we don't return a partial list if there's an error
	files = found
	return
}
