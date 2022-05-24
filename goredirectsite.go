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
	filepath string
	id       string
}

type fileRedirs struct {
	sourceFile string
	redirects  []string
}

var baseUrl string
var oldFilesDir string
var newFilesDir string

var oldFileMetadata map[string]fileMetadata
var newFileMetadata map[string]fileMetadata
var existingRedirs []fileRedirs

var fileTemplate = `<meta http-equiv="refresh" content="0; URL='%s'" />`

func main() {

	// base-url old-dir-tree new-dir-tree to-be-created-tree
	//
	flag.Parse()

	baseUrl = flag.Arg(0) // eg: https://docs.deque.com/devtools-html/4.0.0/en

	oldFilesDir = flag.Arg(1)
	newFilesDir = flag.Arg(2)
	createDir := flag.Arg(3) // The output directory for the redirect site

	if baseUrl == "" || oldFilesDir == "" || newFilesDir == "" || createDir == "" {
		log.Fatal("must specify four parameters: one url and three directories: <base-url> <oldFilesDir> <newFilesDir> <createDir>")
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

	existingRedirs = make([]fileRedirs, 0)

	// Walk old files
	log.Println("Walking old site's files")
	err = walkFiles(oldFilesDir, oldFileMetadata, true)

	if err != nil {
		log.Printf("error walking the path: %v\n", err)
		return
	}

	log.Println("Walking new site's files")
	err = walkFiles(newFilesDir, newFileMetadata, false) // Don't save the new site's metadata redirects

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

	// Now create the main page redirects

	for k, src := range oldFileMetadata {
		dest, ok := newFileMetadata[k]
		if ok {

			newSrc := fixSrc(src.filepath)
			newDest := dest.id
			log.Printf("Found src: %s with dest: %s using key: %s\n",
				strings.TrimPrefix(src.filepath, oldFilesDir), strings.TrimPrefix(dest.filepath, newFilesDir),
				k)

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
	}

	// Now create manual redirects based on the redirect_from metadata

	for _, redir := range existingRedirs {
		for _, redirFile := range redir.redirects {
			log.Printf("Src file: %s, redir: %s\n", redir.sourceFile, redirFile)
		}
	}

	// Create an index page redirect
}

func checkDir(directory string) (err error) {
	fileInfo, err := os.Stat(directory)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return err
	}

	return nil
}

func walkFiles(filesDir string, metadata map[string]fileMetadata, saveRedirects bool) (err error) {

	err = filepath.Walk(filesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("File walk error %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			parseFile(path, filesDir, metadata, saveRedirects)
		}

		return nil
	})

	return err
}

func parseFile(contentPath string, baseDir string, filemeta map[string]fileMetadata, saveRedirects bool) {

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

	id, ok := metaData["id"].(string)
	if !ok {
		id = ""
	}

	permalink, ok := metaData["permalink"].(string)
	if !ok {
		log.Println("Couldn't find permalink metadata on: " + contentPath)
		return
	}

	_, ok = filemeta[permalink]

	if ok {
		log.Printf("Permalink value already defined: %s for file: %s, other file: %s\n",
			permalink, contentPath, filemeta[permalink].filepath)
		return
	}

	if saveRedirects {
		redirect, ok := metaData["redirect_from"]

		if ok {
			arr, ok := redirect.([]interface{})

			if ok {
				newRedir := fileRedirs{
					sourceFile: contentPath,
					redirects:  make([]string, 0),
				}
				for _, value := range arr {
					newRedir.redirects = append(newRedir.redirects, value.(string))
				}

				existingRedirs = append(existingRedirs, newRedir)

			}
		}
	}

	filemeta[permalink] = fileMetadata{
		filepath: contentPath,
		id:       id,
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
	newSrc, err := filepath.Rel(oldFilesDir, src)
	if err != nil {
		log.Fatalf("In fixSrc: %q", err)
	}

	newSrc = strings.Replace(newSrc, ".md", ".html", 1)

	//	log.Printf("fixSrc: Changed %s to %s", src, newSrc)
	return newSrc
}
