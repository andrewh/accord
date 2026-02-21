// File path resolution: expands globs, recursive patterns, and directories.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	yamlExt = ".yaml"
	ymlExt  = ".yml"
)

func isGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == yamlExt || ext == ymlExt
}

// resolveFilePaths expands a list of arguments into concrete file paths.
// Arguments may be literal file paths, glob patterns (including ** for
// recursive matching), or directories (which are walked recursively for
// .yaml and .yml files). An error is returned for each glob pattern that
// matches no files.
func resolveFilePaths(args []string) ([]string, error) {
	var paths []string
	for _, arg := range args {
		if isGlobPattern(arg) {
			matches, err := doublestar.FilepathGlob(arg)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern %q: %w", arg, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no files matched pattern %q", arg)
			}
			paths = append(paths, matches...)
			continue
		}

		info, err := os.Stat(arg)
		if err != nil {
			paths = append(paths, arg)
			continue
		}

		if !info.IsDir() {
			paths = append(paths, arg)
			continue
		}

		var found []string
		err = filepath.WalkDir(arg, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && isYAMLFile(path) {
				found = append(found, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking directory %q: %w", arg, err)
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("no .yaml or .yml files found in directory %q", arg)
		}
		paths = append(paths, found...)
	}
	return paths, nil
}
