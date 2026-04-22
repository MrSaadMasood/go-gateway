package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	args := os.Args
	fmt.Print(args[1])
	if len(args) > 0 && args[1] == "read" {
		readInternal("./internal")
		return
	}

}

func readInternal(path string) {
	dir, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		fileName := file.Name()
		fullPath := filepath.Join(path, fileName)
		if !file.IsDir() {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				panic(err)
			}
			fmt.Println("---logging file---", fullPath)
			fmt.Print(string(data))
		} else {
			readInternal(fullPath)
		}

	}
}
