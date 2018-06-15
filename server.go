package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("upload.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	t.Execute(w, nil)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("search.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	t.Execute(w, nil)
}

func ResultHandler(w http.ResponseWriter, r *http.Request) {
	g := regexp.Grep{
		Stdout: w,
		Stderr: w,
	}
	r.ParseForm()
	pat := strings.Join(r.Form["query"], "")
	pat = "(?m)" + pat
	re, err := regexp.Compile(pat)
	if err != nil {
		fmt.Println(err)
		return
	}
	g.Regexp = re
	query := index.RegexpQuery(re.Syntax)
	ix := index.Open("searchindex")

	post := ix.PostingQuery(query)
	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}

	// t, err := template.ParseFiles("result.html")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// t.Execute(w, nil)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("sourceFile")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	ix := index.Create("searchindex")
	ix.Add(handler.Filename, file)
	ix.Flush()
	http.Redirect(w, r, "/search/", http.StatusFound)
}

func main() {
	http.HandleFunc("/upload/", UploadHandler)
	http.HandleFunc("/search/", SearchHandler)
	http.HandleFunc("/result/", ResultHandler)
	http.HandleFunc("/index/", IndexHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
