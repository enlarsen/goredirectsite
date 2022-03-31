package main

import (
	"flag"
	"fmt"
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

type fileMetadata struct {
	filepath  string
	id        string
	permalink string
}

var baseUrl string
var oldFileMetadata []fileMetadata
var newFileMetadata []fileMetadata

func main() {

	// base-url old-dir-tree new-dir-tree to-be-created-tree
	//
	flag.Parse()

	baseUrl = flag.Arg(0)

	// TODO: try to parse baseUrl (want to modify this URL to point to each new page)

	oldFilesDir := flag.Arg(1)
	newFilesDir := flag.Arg(2)
	createDir := flag.Arg(3)

	if oldFilesDir == "" || newFilesDir == "" || createDir == "" {
		log.Fatal("must specify three directories: <oldFilesDir> <newFilesDir> <createDir>")
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

	oldFileMetadata = make([]fileMetadata, 0)
	newFileMetadata = make([]fileMetadata, 0)

	// Walk old files

	err = walkFiles(oldFilesDir, oldFileMetadata)

	if err != nil {
		fmt.Printf("error walking the path: %v\n", err)
		return
	}

	err = walkFiles(newFilesDir, newFileMetadata)

	if err != nil {
		fmt.Printf("error walking the path: %v\n", err)
		return
	}

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

func walkFiles(filesDir string, metadata []fileMetadata) (err error) {

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

func parseFile(contentPath string, baseDir string, filemeta []fileMetadata) {

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
		log.Fatalf("Permalink not found on file %s\n", contentPath)
	}

	filemeta = append(filemeta, fileMetadata{
		permalink: permalink,
		filepath:  contentPath,
		id:        id,
	})

}
