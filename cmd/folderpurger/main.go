package main

import (
	"fmt"
	"folder_purger/purger"
	"os"
	"strconv"
)

func main() {
	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) == 0 {
		panic("Expected at least one source and destination folders and destination max size")
	}

	if len(argsWithoutProg)%2 != 0 {
		panic("Wrong number of arguments")
	}

	folders := map[string]*purger.Folder{}

	for i := 0; i < len(argsWithoutProg); i += 2 {
		maxSize, err := strconv.Atoi(argsWithoutProg[i+1])
		if err != nil {
			panic(fmt.Sprintf("Failed to parse destination max size: %s", argsWithoutProg[i+2]))
		}
		folders[argsWithoutProg[i]] = purger.NewFolder(argsWithoutProg[i], int64(maxSize))
	}

	p := purger.NewPurger(folders)
	err := p.Purge() //blocking call

	panic(err)

}
