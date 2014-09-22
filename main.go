// Contains horrible panics.
//
// Usage: prog [-p[ort]|-d[ir]]
//      -p or -port for port
//      -d or -dir for dir to save the uploaded files
//
// 2014, Lauri Peltomäki
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	flimit     = 320 * 1024
	uploadHTML = `<!doctype html>
<html>
<head>
    <title>File upload</title>
</head>
<body>
    <p>File size limit: 320Kb</p>
    <form action="/save" method="POST" enctype="multipart/form-data">
        <label for="file">Filename:</label>
        <input type="file" name="file" id="file">
        <input type="submit" name="submit" value="Submit">
    </form> 
</body>
</html>
`
)

var (
	port       string
	uploadPath string
)

func init() {
	flag.StringVar(&port, "port", "10080", "Port to use for serving the service.")
	flag.StringVar(&port, "p", "10080", "Port to use for serving the service.")
	flag.StringVar(&uploadPath, "dir", "/tmp/fuploads/", "Dir for uploaded files.")
	flag.StringVar(&uploadPath, "d", "/tmp/fuploads/", "Dir for uploaded files.")

	flag.Parse()

	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		if err := os.Mkdir(uploadPath, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create upload dir.")
			panic(err)
		}
	}

	if !strings.HasSuffix(uploadPath, "/") {
		uploadPath += "/"
	}
}

func genRandTitle() (title string) {
	co := "bcdfghjklmnpqrstvwxyz"
	vo := "aeiou"
	var l []string
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var st string = ""
	for i := 0; i < 12; i++ {
		if (i % 2) == 0 {
			if st != "" {
				l = append(l, st)
			}
			st = string(co[r.Intn(len(co))])
		} else {
			st = st + string(vo[r.Intn(len(vo))])
		}

	}
	for _, v := range l {
		title += v
	}
	return
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(uploadPath)
	if err != nil {
        panic(err)
	}
	// Don't keep too many files around, just in case.
	if len(files) > 30 {
		for _, f := range files {
			if err := os.Remove(uploadPath + f.Name()); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to remove redundant files.")
				panic(err)
			}
		}
	}

	// Naive, though works if there are no nasty complications nor,
	// more likely, a malicious party giving false ContentLength.
	// Should read to a buffer and check against the limit.
	if r.ContentLength > flimit {
		fmt.Fprintf(w, "Too large file...")
		return
	}
	file, _, err := r.FormFile("file") // the FormFile function takes in the POST input id file
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	defer file.Close()

	var title = genRandTitle()
	out, err := os.Create(uploadPath + title) // possibility of overwrite
	if err != nil {
		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
		return
	}
	defer out.Close()

	// write the content from POST to the file
	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	fmt.Println("File uploaded - " + title)

	http.Redirect(w, r, "/v/"+title, http.StatusFound)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, uploadHTML) })
	http.HandleFunc("/save", uploadHandler)
	http.HandleFunc("/v/",
		func(w http.ResponseWriter, r *http.Request) {
			up := r.URL.Path
			if up == "" {
				http.NotFound(w, r)
				return
			}
			ipath := uploadPath + (strings.Split(up, "/")[2])
			http.ServeFile(w, r, ipath)
		})
	http.ListenAndServe(":"+port, nil)
}
