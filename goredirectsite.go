package main

// example index.html file:

// <meta http-equiv="refresh" content="0; URL='/devtools-html/4.0.0/en/extensions-devtools'" />

import (
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// type fileMetadata struct {
// 	filepath  string
// 	id        string
// 	permalink string
// }

type fileMetadata struct {
	filepath  string
	id        string
	redirects []string
}

var baseUrl string
var defaultPageID string
var oldFilesDir string
var newFilesDir string
var createDir string

var oldFileMetadata map[string]fileMetadata
var newFileMetadata map[string]fileMetadata

var fileTemplate = `<meta http-equiv="refresh" content="0; URL='%s'" />`

func main() {

	// base-url default-page-id old-dir-tree new-dir-tree to-be-created-tree
	//
	flag.Parse()

	baseUrl = flag.Arg(0) // eg: https://docs.deque.com/devtools-html/4.0.0/en

	defaultPageID = flag.Arg(1)

	oldFilesDir = flag.Arg(2)
	newFilesDir = flag.Arg(3)
	createDir = flag.Arg(4) // The output directory for the redirect site

	if baseUrl == "" || oldFilesDir == "" || newFilesDir == "" || createDir == "" {
		log.Fatal("must specify five parameters: <base-url> <default-page-id> <oldFilesDir> <newFilesDir> <createDir>")
	}

	err := checkDir(oldFilesDir)
	if err != nil {
		log.Fatal(err)
	}

	err = checkDir(newFilesDir)
	if err != nil {
		log.Fatal(err)
	}

	// Not really necessary because it's created later, so remove this code.
	err = checkDir(createDir)
	if err != nil {
		os.MkdirAll(createDir, 0755)
	}

	oldFileMetadata = make(map[string]fileMetadata, 0)
	newFileMetadata = make(map[string]fileMetadata, 0)

	// Walk old files
	log.Println("Walking old site's files")
	err = walkFiles(oldFilesDir, oldFileMetadata)

	if err != nil {
		log.Printf("error walking the path: %v\n", err)
		return
	}

	log.Println("Walking new site's files")
	err = walkFiles(newFilesDir, newFileMetadata)

	if err != nil {
		log.Printf("error walking the path: %v\n", err)
		return
	}

	// Check forward matches on permalink
	log.Println("Checking old (source) against new (destination)")
	match(oldFileMetadata, newFileMetadata)
	// Just for fun (see if all the new site's files match to old site's):
	// log.Println("Checking new (source) against old (destination)")
	// match(newFileMetadata, oldFileMetadata)

	for k, src := range oldFileMetadata {
		dest, ok := newFileMetadata[k]
		if ok {

			// Now create the main page redirects

			makeRedirect(k, dest.id) // Use k, the permalink, as the src path

			// Now create manual redirects based on the redirect_from metadata

			for _, redir := range src.redirects {
				log.Printf("Src file: %s, redir: %s\n", fixSrc(k), redir)
				makeRedirect(redir, dest.id)

			}

		}
	}

	// Create an index page redirect

	makeRedirect("index.html", defaultPageID)
}

func checkDir(directory string) (err error) {
	fileInfo, err := os.Stat(directory)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", directory)
	}

	return nil
}

func walkFiles(filesDir string, metadata map[string]fileMetadata) (err error) {

	err = filepath.Walk(filesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("File walk error %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			parseFile(path, filesDir, metadata)
		}

		return nil
	})
	return
}

func parseFile(contentPath string, baseDir string, filemeta map[string]fileMetadata) {

	if path.Ext(contentPath) != ".md" {
		return
	}

	md := goldmark.New(goldmark.WithExtensions(meta.Meta))

	buffer, err := ioutil.ReadFile(contentPath)

	if err != nil {
		log.Fatal(err)
	}

	reader := text.NewReader(buffer)

	context := parser.NewContext()

	md.Parser().Parse(reader, parser.WithContext(context))

	metaData := meta.Get(context)
	if metaData == nil {
		//		log.Fatalf("No metadata on %s\n", contentPath)
		return
	}
	id, ok := metaData["id"].(string)
	if !ok {
		id = ""
	}

	permalink, ok := metaData["permalink"].(string)

	// If there is no permalink metadata, give up.
	// Giving up has big consequences because then there
	// are no redirects created, but if there is no permalink
	// they can't be matched to a URL on the new site so we're
	// stuck anyway without a permalink.

	if !ok {
		log.Println("Couldn't find permalink metadata on: " + contentPath)
		return
	}

	_, ok = filemeta[permalink]

	if ok {
		log.Printf("Permalink value already defined: %s for file: %s, other file: %s\n",
			permalink, contentPath, filemeta[permalink].filepath)
	}

	redirects := make([]string, 0)
	redirect, ok := metaData["redirect_from"]

	if ok {
		arr, ok := redirect.([]interface{})

		if ok {

			for _, value := range arr {
				redirects = append(redirects, value.(string))
			}
		}
	}

	if id != "" {
		filemeta[permalink] = fileMetadata{
			filepath:  contentPath,
			id:        id,
			redirects: redirects,
		}
	}
}

func match(source map[string]fileMetadata, dest map[string]fileMetadata) {

	for k := range source {
		_, ok := dest[k]
		if !ok {
			log.Printf("Couldn't find %s in destination pages", k)
		}
	}
}

// Modifications to the src path go here

func fixSrc(src string) string {
	newSrc := filepath.Join(oldFilesDir, src)

	newSrc, err := filepath.Rel(oldFilesDir, newSrc)
	if err != nil {
		log.Fatalf("In fixSrc: %q", err)
	}

	newSrc = strings.Replace(newSrc, ".md", ".html", 1)

	//	log.Printf("fixSrc: Changed %s to %s", src, newSrc)
	return newSrc
}

func makeRedirect(srcFilepath string, destId string) {

	var newSrc string

	if filepath.IsAbs(srcFilepath) {
		newSrc = fixSrc(srcFilepath)
	} else {
		newSrc = srcFilepath
		if !strings.HasSuffix(newSrc, ".html") {
			newSrc = filepath.Join(newSrc, "index.html")
		}
		log.Printf("**** Got relative path for makeRedirect: %s changed to: %s", srcFilepath, newSrc)
	}

	newDest := destId
	log.Printf("Found src: %s with destId: %s\n",
		strings.TrimPrefix(srcFilepath, oldFilesDir), destId)

	// Create a directory at the output directory plus existing dir to hold the redirect file.
	os.MkdirAll(filepath.Join(createDir, filepath.Dir(newSrc)), 0755)

	destinationUrl, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal("Could not parse baseUrl")
	}
	destinationUrl.Path = path.Join(destinationUrl.Path, newDest)

	contents := fmt.Sprintf(fileTemplate, destinationUrl.String())

	file, err := os.Create(path.Join(createDir, newSrc))
	if err != nil {
		log.Fatalf("Could not create %s", newSrc)
	}
	file.WriteString(contents)
	file.Close()

}
