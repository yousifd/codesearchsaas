package main

import (
	"codesearch/indexer"
	"codesearch/util"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// UploadHandler Handles upload requests
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("upload.html")
	util.CheckError(err)
	t.Execute(w, nil)
}

// SearchHandler Handles search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("search.html")
	util.CheckError(err)
	t.Execute(w, nil)
}

// ResultHandler Handles result display requests
func ResultHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	pat := strings.Join(r.Form["pattern"], "")
	repoName := strings.Join(r.Form["repo"], "")
	indexer.QueryIndex(w, pat, repoName)
}

// RepoIndexHandler Handles repo upload requests
func RepoIndexHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	repoURL := strings.Join(r.Form["repoURL"], "")
	//go indexer.IndexRepo(repoURL) // TODO: Make this concurrent (would be awesome)
	indexer.IndexRepo(repoURL)

	files, err := ioutil.ReadDir(indexer.IndexDir)
	util.CheckError(err)

	t, err := template.ParseFiles("search.html")
	util.CheckError(err)
	t.Execute(w, files)
}

func main() {
	http.HandleFunc("/", UploadHandler)
	http.HandleFunc("/search/", SearchHandler)
	http.HandleFunc("/result/", ResultHandler)
	http.HandleFunc("/repoindex/", RepoIndexHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
