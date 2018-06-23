package main

import (
	"codesearch/indexer"
	"codesearch/util"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	templateDir = "templates/"
	searchHTML  = templateDir + "search.html"
	resultHTML  = templateDir + "result.html"
	fileHTML    = templateDir + "file.html"
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

// FileTreeString returns a string representation of a tree of files following a directory structure
func FileTreeString(file string, depth int) string {
	out := ""
	fi, err := os.Stat(file)
	util.CheckError(err)
	if fi.IsDir() {
		files, err := ioutil.ReadDir(file)
		util.CheckError(err)

		out += strings.Repeat("&nbsp;", depth) + fi.Name() + "</br>"
		for _, f := range files {
			fileName := f.Name()
			if fileName[0] == '.' || fileName[0] == '#' || fileName[0] == '~' || fileName[len(fileName)-1] == '~' {
				continue
			}
			filePath := file + "/" + fileName
			log.Printf("filePath: %s", filePath)
			out += FileTreeString(filePath, depth+1)
		}
	} else {
		log.Printf("reg filename: %s", file)
		out += strings.Repeat("&nbsp;", depth) +
			"<a href=\"/file/?f=" + file + "\">" + file + "</a></br>"
	}

	return out
}

// FileHandler Handles Opening files for user
func FileHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var fileName, lineStr string
	line := -1
	if len(q["f"]) != 0 {
		fileName = q["f"][0]
	}
	if len(q["l"]) != 0 {
		lineStr = q["l"][0]
		l, err := strconv.Atoi(lineStr)
		util.CheckError(err)
		line = l
	}

	log.Printf("fileName: %s", fileName)
	file, err := os.Open(fileName)
	util.CheckError(err)
	defer file.Close()

	fileContentByte, err := ioutil.ReadFile(fileName)
	util.CheckError(err)
	fileContentStr := strings.Split(string(fileContentByte), "\n")
	fileContentStr = append(fileContentStr, "</pre>")
	if line != -1 {
		fileContentStr[line-1] = "<span style=\"background-color: yellow;\">" + fileContentStr[line-1] + "</span>"
	}
	for i, cont := range fileContentStr {
		fileContentStr[i] = fmt.Sprintf("<div id=\"%d\">%d. %s</div>", i+1, i+1, cont)
	}
	fileContentStr = append([]string{"<pre>"}, fileContentStr...)
	out := strings.Join(fileContentStr, "\n")

	splitFile := strings.Split(fileName, "/")
	rootFile := splitFile[0] + "/" + splitFile[1]

	t, err := template.ParseFiles(fileHTML)
	util.CheckError(err)
	outStruct := struct {
		FileName    string
		FileContent template.HTML
		FileTree    template.HTML
	}{
		FileName:    fileName,
		FileContent: template.HTML(out),
		FileTree:    template.HTML("<par>" + FileTreeString(rootFile, 0) + "</par>"),
	}
	t.Execute(w, outStruct)
}

func main() {
	indexer.Init()
	http.HandleFunc("/", SearchHandler)
	http.HandleFunc("/result/", ResultHandler)
	http.HandleFunc("/repoindex/", RepoIndexHandler)
	http.HandleFunc("/file/", FileHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
