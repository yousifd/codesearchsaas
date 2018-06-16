package indexer

import (
	"codesearch/util"
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
	// IndexDir Path to searchindex files for each repo
	IndexDir = "searchindex/"
	reposDir = "repos/"
)

// TODO: If a repo is already indexed, then pull latests changes, and index new/modified files
// TODO: Have a list of all avialable repos and be able to specify them during search
//	- Always pull latest changes and reindex repo to make sure you are up to date on search
//  - Maybe have a timeout to avoid overloading the server
// TODO: Option to search all repos
// TODO: Return output as JSON so it can be used in any way the user wants not and just strings
// TODO: Add Search options similar to cmd line flags
// TODO: Add filtering features (Predefines queries) + Inline Query options
// TODO: File reader with ability to highlight variables and functions
// 	and be able to search for them specificially in directory:
// 	Defenitions, Declerations, and References
// 	- Modify Index to store line numbers
//  - Add links to file and load at specified line number
//	- When loading file from server identify all the keywords for vars and funcs
//  - Add links that basically do a query on that keyword and return all references
//  - Need to be able to identify keywords server side
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

// IndexRepo Indexes a whole repo to indexFile
func IndexRepo(url string) {
	repoName := GetRepoName(url)
	indexFile := IndexDir + repoName
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
func QueryIndex(w http.ResponseWriter, pat string, repoName string) {
	g := regexp.Grep{
		Stdout: w,
		Stderr: w,
	}

	pat = "(?m)" + pat
	re, err := regexp.Compile(pat)
	util.CheckError(err)
	g.Regexp = re
	query := index.RegexpQuery(re.Syntax)
	indexFile := IndexDir + repoName
	// TODO: Wait for index to finish
	ix := index.Open(indexFile)

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}
}

// GetRepoName Returns the name of the repo from its url
func GetRepoName(url string) string {
	repoName := strings.Split(url, "/")
	return repoName[len(repoName)-1]
}
