package main

// example index.html file:

// <meta http-equiv="refresh" content="0; URL='/devtools-html/4.0.0/en/extensions-devtools'" />

import (
	"flag"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

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
var oldFileMetadata map[string]fileMetadata
var newFileMetadata map[string]fileMetadata
var existingRedirs []fileRedirs

func main() {

	// base-url old-dir-tree new-dir-tree to-be-created-tree
	//
	flag.Parse()

	baseUrl = flag.Arg(0) // eg: https://docs.deque.com/devtools-html/4.0.0/en

	oldFilesDir := flag.Arg(1)
	newFilesDir := flag.Arg(2)
	createDir := flag.Arg(3)

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

	// TODO: maybe this should just create the directory if it doesn't exist
	err = checkDir(createDir)
	if err != nil {
		log.Fatal(err)
	}

	oldFileMetadata = make(map[string]fileMetadata, 0)
	newFileMetadata = make(map[string]fileMetadata, 0)

	existingRedirs = make([]fileRedirs, 0)

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

	log.Println("Checking old (source) against new (destination)")
	match(oldFileMetadata, newFileMetadata)
	log.Println("Checking new (source) against old (destination)")
	match(newFileMetadata, oldFileMetadata)

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

	return err
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

	filemeta[permalink] = fileMetadata{
		filepath: contentPath,
		id:       id,
	}

	// *filemeta = append(*filemeta, fileMetadata{
	// 	permalink: permalink,
	// 	filepath:  contentPath,
	// 	id:        id,
	// })

}

func match(source map[string]fileMetadata, dest map[string]fileMetadata) {

	for k, _ := range source {
		_, ok := dest[k]
		if !ok {
			log.Printf("Couldn't find %s in destination pages", k)
		}
	}
}
