package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/vodafon/urly/lib"
)

var (
	flagInput = flag.String("i", "", "input")
)

func main() {
	flag.Parse()
	if *flagInput == "" {
		processReader(os.Stdin)
		return
	}

	path := *flagInput
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Fatalf("The path %s does not exist.\n", path)
		return
	}
	if err != nil {
		log.Fatalf("Error checking the path: %v\n", err)
		return
	}

	if info.IsDir() {
		processDir(path)
	} else {
		processFile(path)
	}
}

func processDir(path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
		return
	}
	for _, file := range files {
		if file.IsDir() {
			processDir(filepath.Join(path, file.Name()))
		} else {
			processFile(filepath.Join(path, file.Name()))
		}
	}
}

func processFile(path string) {
	reader, err := os.Open(path)
	if err != nil {
		log.Fatalf("read file %s error: %v\n", path, err)
	}
	defer reader.Close()

	processReader(reader)
}

func processReader(reader io.Reader) {
	res, err := lib.ExtractURL(reader)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(res))
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		println(s)
	}
	return nil
}
