package indexer

import (
	"codesearch/util"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	indexFile = "searchindex"
	reposDir  = "repos/"
)

// TODO: Support multiple indexFiles, one for each repo/indexedfile
// TODO: Add links to file and load at specified line number
// 	- Modify Index to store line numbers
// TODO: Add options similar to cmd line flags
// TODO: Add filtering features
// TODO: File reader with ability to highlight variables and functions
// 	and be able to search for them specificially in directory:
// 	Defenitions, Declerations, and References

// IndexFile Indexes a file to indexFile
func IndexFile(filename string, file io.Reader) {
	ix := index.Create(indexFile)
	log.Printf("index %s", filename)
	ix.Add(filename, file)
	log.Printf("flush index")
	ix.Flush()
	log.Printf("done")
}

// CloneRepo Clones repo at url and returns tree of commit at HEAD
func CloneRepo(url string) (*object.Tree, string) {
	repoName := strings.Split(url, "/")
	repoPath := reposDir + repoName[len(repoName)-1]
	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL: url,
	})
	if err == git.ErrRepositoryAlreadyExists {
		repo, err = git.PlainOpen("repo")
		log.Printf("Repo already exists")
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

// IndexRepo Indexes a whole repo to indexFile
func IndexRepo(url string) {
	ix := index.Create(indexFile)
	var paths []string

	// Iterate over files in repo HEAD
	tree, repoPath := CloneRepo(url)
	tree.Files().ForEach(func(f *object.File) error {
		file := repoPath + f.Name
		paths = append(paths, file)
		log.Printf("index %s", f.Name)
		return nil
	})
	ix.AddPaths(paths)

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		log.Printf("path: %s", path)
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

	log.Printf("flush index")
	ix.Flush()
	log.Printf("done")
}

// QueryIndex Applies query to index and returns results
func QueryIndex(w http.ResponseWriter, pat string) {
	g := regexp.Grep{
		Stdout: w,
		Stderr: w,
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
