// The util package contains miscellaneous helper functions.

package util

import (
	"bytes"
	"os/exec"
	"path"
	"runtime"

	"golang.org/x/tools/go/vcs"
)

// RepoInfo attempts to find the repo information for the caller, and returns
// the path to the top of the repo, the commit ID, and the tag.
func RepoInfo() (repoRoot string, repoRevision string, repoTag string) {
	var vcsCmd *vcs.Cmd

	// Walk up the call stack until we find main.main
	for i := 1; ; i++ {
		caller, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}

		f := runtime.FuncForPC(caller)
		if f == nil {
			continue
		}
		if path.Base(f.Name()) == "main.main" {
			repoRoot = path.Dir(file)
			break
		}

		// see if we're in testing
		if path.Base(f.Name()) == "testing.tRunner" {
			// go back up one level
			_, file, _, _ := runtime.Caller(i - 1)
			repoRoot = path.Dir(file)
			break
		}
	}

	// Try and find the root of the VCS repo using repoRoot as a starting point.
	if repoRoot != "" {
		// walk up the dir until we get to the first directory in root.
		// This is an unfortunately necessity to use vcs.FromDir()
		topDir := repoRoot
		for dir := path.Dir(topDir); dir != "." && dir[len(dir)-1] != '/' && dir != topDir; dir = path.Dir(topDir) {
			topDir = dir
		}

		var vcsRoot string
		vcsCmd, vcsRoot, _ = vcs.FromDir(path.Dir(repoRoot), topDir)
		if len(vcsRoot) > 0 {
			if vcsRoot[0] != '/' {
				vcsRoot = topDir + "/" + vcsRoot
			}
			repoRoot = vcsRoot
		}
	}

	if vcsCmd != nil && vcsCmd.Name == "Git" {
		execCmd := exec.Command("git", "describe", "--dirty", "--match", "", "--always")
		execCmd.Dir = repoRoot
		output, err := execCmd.Output()
		if err == nil {
			repoRevision = string(bytes.TrimRight(output, "\n"))
		}

		execCmd = exec.Command("git", "describe", "--dirty", "--tags", "--always")
		execCmd.Dir = repoRoot
		output, err = execCmd.Output()
		if err == nil {
			repoTag = string(bytes.TrimRight(output, "\n"))
		}
	}

	return
}
