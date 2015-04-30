package util

import (
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoRoot(t *testing.T) {
	// assume this test file is `sawmill/util/repo_root_test.go`
	_, gitRoot, _, _ := runtime.Caller(0)
	gitRoot = path.Dir(path.Dir(gitRoot))

	repoRoot, repoTag, repoVersion := RepoInfo()
	assert.Equal(t, gitRoot, repoRoot)
	assert.NotEmpty(t, repoTag)
	assert.NotEmpty(t, repoVersion)
}
