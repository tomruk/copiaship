package ifile

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	pathspec "github.com/karagenc/go-pathspec"
	"github.com/karagenc/kopyat/internal/utils"
)

const (
	gitignore    = ".gitignore"
	kopyatignore = ".kopyatignore"

	// We need to be fast while walking, and this is faster than
	// comparing strings every time we need to check whether we're
	// running on Windows.
	runningOnWindows = runtime.GOOS == "windows"
)

func (i *Ifile) Walk(root string) error {
	ignorefiles := make([]*ignorefile, 0, 100)
	entries := make(entries, 0, 10000)
	err := addIgnoreIfExists(&ignorefiles, root)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if e, ok := err.(*fs.PathError); ok {
				if e, ok := e.Err.(syscall.Errno); ok && e.Is(fs.ErrPermission) {
					return nil
				}
			}
			return err
		} else if path == root {
			return nil
		}

		t := d.Type()
		if t.IsDir() {
			err = addIgnoreIfExists(&ignorefiles, path)
			if err != nil {
				return err
			}
		}

		anyMatches := false
		for j := len(ignorefiles) - 1; j >= 0; j-- {
			igFile := ignorefiles[j]

			if strings.HasPrefix(path, igFile.dir) {
				trimmed := path[len(igFile.dir):]
				if t.IsDir() && !strings.HasSuffix(trimmed, "/") {
					trimmed += "/"
				}
				match := igFile.p.Match(trimmed)

				if i.mode == ModeRestic && match {
					if t.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if i.mode == ModeSyncthing && match {
					anyMatches = true
					path = path[len(root):]
					break
				}
			}
		}

		if i.mode == ModeSyncthing && !anyMatches {
			return nil
		} else if i.mode == ModeSyncthing {
			if runningOnWindows {
				path = utils.StripDriveLetter(path)
			}
		}

		entries = append(entries, &entry{
			path:  path,
			isDir: t.IsDir(),
		})
		return nil
	})
	if err != nil {
		return err
	}

	growLen := 0
	for _, entry := range entries {
		growLen += entry.Len()
	}

	i.bufMu.Lock()
	defer i.bufMu.Unlock()
	i.buf.Grow(growLen)

outer:
	for _, entry := range entries {
		// If the entry is a directory, check if it contains a children.
		// If it's empty (doesn't have a children), add it to the list.
		if entry.isDir {
			for _, _entry := range entries {
				if strings.HasPrefix(_entry.path, entry.path) && strings.ContainsRune(strings.TrimPrefix(_entry.path, entry.path), os.PathSeparator) {
					continue outer
				}
			}
		}

		_, err = i.buf.WriteString(entry.String())
		if err != nil {
			return err
		}
	}
	return err
}

func addIgnoreIfExists(ignorefiles *[]*ignorefile, dir string) error {
	path := filepath.Join(dir, gitignore)
	if f, err := os.Stat(path); err == nil && f.Mode().Type().IsRegular() {
		p, err := pathspec.FromFile(path)
		if err != nil {
			return err
		}
		*ignorefiles = append(*ignorefiles, &ignorefile{
			p:   p,
			dir: dir,
		})
	}

	path = filepath.Join(dir, kopyatignore)
	if f, err := os.Stat(path); err == nil && f.Mode().Type().IsRegular() {
		p, err := pathspec.FromFile(path)
		if err != nil {
			return err
		}
		*ignorefiles = append(*ignorefiles, &ignorefile{
			p:   p,
			dir: dir,
		})
	}
	return nil
}
