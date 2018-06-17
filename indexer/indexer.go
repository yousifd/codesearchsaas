package indexer

import (
	"bytes"
	"codesearch/util"
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

// TODO: Fix bug where who server crashes if invalid result param is specified "result/adfasdfsa"
// TODO: Paralellaize querying with regexp using go
// TODO: If a repo is already indexed, pull latests changes, and index new/modified files
// TODO: Always pull latest changes and reindex repo to make sure you are up to date on search
//  - Maybe have a timeout per repo to avoid overloading the server
// TODO: Option to search all repos
// TODO: A search option to specify a repo to upload and index + a search query
// TODO: Add Search options similar to cmd line flags
//  - main function options from cindex
//  - regexp.Grep() options
// TODO: Add filtering features (Predefined queries) + Inline Query filters
// TODO: Call IndexRepo concurrently and have a lock per repo when indexing or something better
// TODO: Project Files Explorer
// TODO: File reader with ability to highlight variables and functions:
//	- When loading file in server identify all the keywords for vars and funcs
//  - Add links that basically do a query on that keyword and return all relations
//   - Relation types: Defenitions, Declerations, and References
//  - Find common issues: secruity flaws, bugs, spelling mistakes, grammar?

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

// QueryIndex Applies query to index and returns results
func QueryIndex(pat string, repoName string) *Result {
	buf := new(bytes.Buffer)
	g := regexp.Grep{
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
	ix := index.Open(indexFile)

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
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
			File:    splitEntry[0],
			Line:    lineNumber,
			Content: splitEntry[2],
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
