package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MaxFilenameLen is the maximum length of a file name in bytes
const MaxFilenameLen = 255

func genName(path string) string {
	if len(path) >= MaxFilenameLen {
		ex := len(path) - MaxFilenameLen
		path = path[0 : len(path)-ex-1]
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return (path)
	}
	name := strings.TrimSuffix(path, filepath.Ext(path))
	for i := 0; ; i++ {
		if i > 10 {
			panic("Something wrong with new filename")
		}
		s := fmt.Sprint(name + "-" + strconv.Itoa(i) + filepath.Ext(path))
		fmt.Println(s)
		if _, err := os.Stat(s); os.IsNotExist(err) {
			fmt.Println(filepath.Abs(s))
			return (s)
		}
	}
}

func jsonExport(export interface{}, filename string) {
	b, err := json.MarshalIndent(export, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	// open output file
	fo, err := os.Create(genName(filename))
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	// make a buffer to keep chunks that are read
	if _, err := fo.Write(b); err != nil {
		panic(err)
	}
}
