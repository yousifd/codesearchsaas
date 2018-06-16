package main

import (
	"html/template"
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
)

// CheckError General Error Checking
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

// UploadHandler Handles upload requests
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("upload.html")
	CheckError(err)
	t.Execute(w, nil)
}

// SearchHandler Handles search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("search.html")
	CheckError(err)
	t.Execute(w, nil)
}

// ResultHandler Handles result display requests
func ResultHandler(w http.ResponseWriter, r *http.Request) {
	g := regexp.Grep{
		Stdout: w,
		Stderr: w,
	}

	r.ParseForm()
	pat := strings.Join(r.Form["pattern"], "")
	pat = "(?m)" + pat
	re, err := regexp.Compile(pat)
	CheckError(err)
	g.Regexp = re
	query := index.RegexpQuery(re.Syntax)
	ix := index.Open(indexFile)

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}
}

// RepoIndexHandler Handles repo upload requests
func RepoIndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Support Private Repos
	r.ParseForm()
	repoURL := strings.Join(r.Form["repoURL"], "")
	// TODO: If repo exists just open it or just delete it for now
	repo, err := git.PlainClone("repo", false, &git.CloneOptions{
		URL: repoURL,
	})
	CheckError(err)

	ref, err := repo.Head()
	CheckError(err)

	commit, err := repo.CommitObject(ref.Hash())
	CheckError(err)

	tree, err := commit.Tree()
	CheckError(err)

	ix := index.Create(indexFile)
	var paths []string
	// Iterate over files in repo HEAD
	tree.Files().ForEach(func(f *object.File) error {
		// TODO: Figure out how to setup paths to repos
		file := "repo/" + f.Name
		paths = append(paths, file)
		log.Printf("index %s", f.Name)
		filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			log.Printf("path: %s", path)
			if _, elem := filepath.Split(path); elem != "" {
				// Skip various temporary or "hidden" files or directories.
				// TODO: Only ignores hidden and not directories in current setup
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
		return nil
	})
	ix.AddPaths(paths)
	log.Printf("flush index")
	ix.Flush()
	log.Printf("done")
	http.Redirect(w, r, "/search/", http.StatusFound)
}

// FileIndexHandler handles indexing requests
// TODO: Add options similar to cmd line flags
// TODO: Add filtering features
// TODO: Modify Index to store line numbers
// TODO: Add links to file and load at specified line number
// TODO: File reader with ability to highlight variables and functions
//	and be able to search for them specificially in directory:
//  Defenitions, Declerations, and References
func FileIndexHandler(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("sourceFile")
	CheckError(err)
	// TODO: Make copy of uploaded file
	defer file.Close()

	ix := index.Create(indexFile)
	log.Printf("index %s", handler.Filename)
	ix.Add(handler.Filename, file)
	log.Printf("flush index")
	ix.Flush()
	log.Printf("done")
	http.Redirect(w, r, "/search/", http.StatusFound)
}

func main() {
	http.HandleFunc("/upload/", UploadHandler)
	http.HandleFunc("/search/", SearchHandler)
	http.HandleFunc("/result/", ResultHandler)
	http.HandleFunc("/fileindex/", FileIndexHandler)
	http.HandleFunc("/repoindex/", RepoIndexHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
