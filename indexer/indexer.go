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

// CloneRepo Clones repo at url and returns tree of commit at HEAD
func CloneRepo(url string) (*object.Tree, string) {
	repoName := GetRepoName(url)
	repoPath := reposDir + repoName
	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL: url,
	})
	if err == git.ErrRepositoryAlreadyExists {
		log.Printf("Repo already exists")
		repo, err = git.PlainOpen(repoPath)
		util.CheckError(err)

		w, err := repo.Worktree()
		util.CheckError(err)
		err = w.Pull(&git.PullOptions{
			RemoteName: "origin",
		})
	}
	util.CheckError(err)

	ref, err := repo.Head()
	util.CheckError(err)

	commit, err := repo.CommitObject(ref.Hash())
	util.CheckError(err)

	tree, err := commit.Tree()
	util.CheckError(err)

	return tree, repoPath
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

// IndexRepo Indexes a whole repo to indexFile
func IndexRepo(url string) {
	repoName := GetRepoName(url)
	indexFile := IndexDir + repoName
	var paths []string

	// Iterate over files in repo HEAD
	tree, repoPath := CloneRepo(url)
	tree.Files().ForEach(func(f *object.File) error {
		file := repoPath + f.Name
		paths = append(paths, file)
		return nil
	})
	IndexFiles(indexFile, repoPath, paths)
}

// ApplyQuery Applies query to indexfile and outputs to g
func ApplyQuery(g *regexp.Grep, indexFile string, query *index.Query) {
	ix := index.Open(indexFile)

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}
}

// QueryIndex Applies query to index and returns results
func QueryIndex(pat string, repoName string) *Result {
	buf := new(bytes.Buffer)
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
	indexFile := IndexDir + repoName
	if indexFile == IndexDir {
		files, err := ioutil.ReadDir(indexFile)
		util.CheckError(err)

		for _, f := range files {
			fileName := indexFile + f.Name()
			log.Printf("file Name: %s", fileName)
			ApplyQuery(g, fileName, query)
		}
	} else {
		ApplyQuery(g, indexFile, query)
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
