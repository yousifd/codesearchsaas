package indexer

import (
	"bytes"
	"codesearch/util"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	// IndexDir Path to searchindex files for each repo
	IndexDir = "searchindex/"
	reposDir = "repos/"
)

// Entry Represents an entry from a result query
type Entry struct {
	File    string
	Line    int
	Content string
}

// Result Represents all resulting entries of a query
type Result struct {
	Entries []Entry
}

// PullRepoChanges Pulls latests changes from repo
func PullRepoChanges(repoPath string) *git.Repository {
	repo, err := git.PlainOpen(repoPath)
	util.CheckError(err)

	w, err := repo.Worktree()
	util.CheckError(err)

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
	})
	if err != git.NoErrAlreadyUpToDate {
		util.CheckError(err)
	}

	return repo
}

// TreeFromRepo Returns a tree representing commit at HEAD
func TreeFromRepo(repo *git.Repository) *object.Tree {
	ref, err := repo.Head()
	util.CheckError(err)

	commit, err := repo.CommitObject(ref.Hash())
	util.CheckError(err)

	tree, err := commit.Tree()
	util.CheckError(err)

	return tree
}

// CloneRepo Clones repo at url and returns tree of commit at HEAD
func CloneRepo(url string) (*object.Tree, string) {
	repoName := GetRepoName(url)
	repoPath := reposDir + repoName
	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL: url,
	})
	if err == git.ErrRepositoryAlreadyExists {
		log.Printf("Repo already exists")
		repo = PullRepoChanges(repoPath)
		err = nil
	}
	util.CheckError(err)

	return TreeFromRepo(repo), repoPath
}

// IndexFiles Indexes files in under repoPath
func IndexFiles(indexFile string, repoPath string, paths []string) {
	ix := index.Create(indexFile)
	ix.AddPaths(paths)

	// Index all files in repo
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if _, elem := filepath.Split(path); elem != "" {
			// Skip various temporary or "hidden" files or directories.
			if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if err != nil {
			log.Printf("%s: %s", path, err)
			return nil
		}
		if info != nil && info.Mode()&os.ModeType == 0 {
			ix.AddFile(path)
		}
		return nil
	})

	ix.Flush()
}

// IndexTree Indexes tree within indexFile
func IndexTree(tree *object.Tree, repoPath string, indexFile string) {
	var paths []string
	tree.Files().ForEach(func(f *object.File) error {
		file := repoPath + f.Name
		paths = append(paths, file)
		return nil
	})
	IndexFiles(indexFile, repoPath, paths)
}

// IndexRepo Indexes a whole repo to indexFile
func IndexRepo(url string) {
	repoName := GetRepoName(url)
	indexFile := IndexDir + repoName

	tree, repoPath := CloneRepo(url)
	IndexTree(tree, repoPath, indexFile)
}

// ApplyQuery Applies query to indexfile and outputs to g
func ApplyQuery(indexFile string, pat string, buf *bytes.Buffer) {
	g := &regexp.Grep{
		Stdout: buf,
		Stderr: buf,
		N:      true,
	}

	pat = "(?m)" + pat
	re, err := regexp.Compile(pat)
	util.CheckError(err)
	g.Regexp = re
	query := index.RegexpQuery(re.Syntax)

	ix := index.Open(indexFile)

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}
}

// PullIndexAndQuery Pulls latests changes for repo and indexes them then applies query
func PullIndexAndQuery(indexFile string, repoPath string, pat string, buf *bytes.Buffer) {
	repo := PullRepoChanges(repoPath)
	IndexTree(TreeFromRepo(repo), repoPath, indexFile)
	ApplyQuery(indexFile, pat, buf)
}

// QueryIndex Applies query to index and returns results
func QueryIndex(pat string, repoName string) *Result {
	buf := new(bytes.Buffer)
	indexFile := IndexDir + repoName
	if indexFile == IndexDir {
		files, err := ioutil.ReadDir(indexFile)
		util.CheckError(err)

		for _, f := range files {
			fileName := indexFile + f.Name()
			repoPath := reposDir + f.Name()
			PullIndexAndQuery(fileName, repoPath, pat, buf)
		}
	} else {
		repoPath := reposDir + repoName
		PullIndexAndQuery(indexFile, repoPath, pat, buf)
	}

	// Generate Results object from reader
	result := new(Result)
	for _, e := range strings.Split(buf.String(), "\n") {
		if e == "" {
			continue
		}
		splitEntry := strings.Split(e, ":")
		lineNumber, err := strconv.Atoi(splitEntry[1])
		util.CheckError(err)
		entry := Entry{
			File: splitEntry[0],
			Line: lineNumber,
			// Rejoin content that has colons in them using colons
			Content: strings.Join(splitEntry[2:], ":"),
		}
		result.Entries = append(result.Entries, entry)
	}

	return result
}

// GetRepoName Returns the name of the repo from its url
func GetRepoName(url string) string {
	repoName := strings.Split(url, "/")
	return repoName[len(repoName)-1]
}
