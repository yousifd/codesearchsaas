package main

import (
	"codesearch/indexer"
	"codesearch/util"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	templateDir = "templates/"
	searchHTML  = templateDir + "search.html"
	resultHTML  = templateDir + "result.html"
)

// SearchHandler Handles search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(indexer.IndexDir)
	util.CheckError(err)

	t, err := template.ParseFiles(searchHTML)
	util.CheckError(err)
	t.Execute(w, files)
}

// ResultHandler Handles result display requests
func ResultHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	pat := strings.Join(r.Form["pattern"], "")
	repoName := strings.Join(r.Form["repo"], "")
	var result *indexer.Result
	if pat != "" || repoName != "" {
		result = indexer.QueryIndex(pat, repoName)
	}

	t, err := template.ParseFiles(resultHTML)
	util.CheckError(err)
	t.Execute(w, result)
}

// RepoIndexHandler Handles repo upload requests
func RepoIndexHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	repoURL := strings.Join(r.Form["repoURL"], "")
	indexer.IndexRepo(repoURL)

	http.Redirect(w, r, "/", http.StatusFound)
}

// FileHandler Handles Opening files for user
func FileHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("url %s", r.URL)

	fileName := r.URL.Query()["f"][0]

	log.Printf("filename %s", fileName)
	file, err := os.Open(fileName)
	util.CheckError(err)
	defer file.Close()

	io.Copy(w, file)
}

func main() {
	indexer.Init()
	http.HandleFunc("/", SearchHandler)
	http.HandleFunc("/result/", ResultHandler)
	http.HandleFunc("/repoindex/", RepoIndexHandler)
	http.HandleFunc("/file/", FileHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
