package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func Globs(pats []string) ([]string, error) {
	var result1 []string

	// first expand extensions

	for _, pat := range pats {
		result1 = append(result1, ExpandExtension(pat)...)
	}

	// then expand globs

	var result2 []string

	for _, pat := range result1 {
		matches, err := ExpandGlob(pat)
		if err != nil {
			return nil, err
		}

		result2 = append(result2, matches...)
	}

	return result2, nil
}

func ExpandGlob(pat string) ([]string, error) {
	pieces := strings.Split(pat, "**")

	if len(pieces) == 1 {
		return filepath.Glob(pat)
	}

	matches := []string{""}

	// say the glob is a/b/**/c/**/d/e.f

	// pieces = [a/b/, c/, d/e.f]. Say current piece is c/, and everything that matches a/b/** is in matches
	for _, piece := range pieces {

		matchSet := make(map[string]bool)

		// everything matching a/b/** is in matches
		// we glob now
		for _, match := range matches {
			validChildren, err := filepath.Glob(filepath.Clean(match + piece))
			if err != nil {
				return nil, err
			}

			// for each child, we need to find all subchildren, to prep for the next **

			for _, child := range validChildren {
				err := filepath.WalkDir(child, func(path string, info fs.DirEntry, err error) error {
					matchSet[path] = true
					return nil
				})

				if err != nil {
					return nil, err
				}
			}
		}

		newMatches := make([]string, 0, len(matchSet))
		for match := range matchSet {
			newMatches = append(newMatches, match)
		}

		matches = newMatches
	}

	return matches, nil
}

func ExpandExtension(pat string) []string {
	if pat == "" || pat[len(pat)-1] != '}' {
		return []string{pat}
	}

	// Find the last '{' in the pattern
	i := strings.LastIndexByte(pat, '{')
	if i == -1 {
		return []string{pat}
	}

	exts := strings.Split(pat[i+1:len(pat)-1], ",")

	result := make([]string, 0, len(exts))

	for _, ext := range exts {
		result = append(result, pat[:i]+strings.TrimSpace(ext))
	}

	return result
}
