package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/vodafon/urly/lib"
)

var (
	flagInput = flag.String("i", "", "input")
)

func main() {
	flag.Parse()
	if *flagInput == "" {
		processStdin()
	}
}

func processStdin() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		res, err := lib.ExtractURL(scanner.Bytes())
		if err != nil {
			fmt.Println(err)
		}
		fmt.Print(string(res))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		return
	}
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
