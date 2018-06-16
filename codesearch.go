package main

import (
	"codesearch/indexer"
	"codesearch/util"
	"html/template"
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
	indexer.QueryIndex(w, pat)
}

// RepoIndexHandler Handles repo upload requests
func RepoIndexHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	repoURL := strings.Join(r.Form["repoURL"], "")
	indexer.IndexRepo(repoURL)
	http.Redirect(w, r, "/search/", http.StatusFound)
}

// FileIndexHandler handles indexing requests
func FileIndexHandler(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("sourceFile")
	util.CheckError(err)
	// TODO: Make copy of uploaded file
	defer file.Close()
	indexer.IndexFile(handler.Filename, file)
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
