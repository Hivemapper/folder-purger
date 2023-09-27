package main

import (
	"fmt"
	"folder_purger/purger"
	"os"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"golang.org/x/sys/unix"
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
		maxSize := uint64(0)
		folder := argsWithoutProg[i]
		if strings.HasSuffix(folder, "/") {
			folder = folder[:len(folder)-1]
		}
		sizeParam := argsWithoutProg[i+1]
		if strings.HasSuffix(sizeParam, "%") {
			maxSizePercent, err := strconv.Atoi(sizeParam[:len(sizeParam)-1])
			if err != nil {
				panic(fmt.Sprintf("Failed to parse destination max size: %s", argsWithoutProg[i+2]))
			}
			maxSize, err = getDriveFreeSpace(folder, uint64(maxSizePercent))

		} else {
			size, err := strconv.Atoi(sizeParam)
			if err != nil {
				panic(fmt.Sprintf("Failed to parse destination max size: %s", argsWithoutProg[i+2]))
			}
			maxSize = uint64(size)
		}
		f := purger.NewFolder(folder, int64(maxSize))
		folders[folder] = f
		fmt.Println("tracking folder:", folder, "max size:", humanize.Bytes(maxSize))
	}

	p := purger.NewPurger(folders)
	err := p.Purge() //blocking call

	panic(err)

}

func getDriveFreeSpace(path string, percent uint64) (uint64, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	availableBytes := stat.Bavail * uint64(stat.Bsize)
	totalBytes := stat.Blocks * uint64(stat.Bsize)

	fmt.Println("availableBytes:", humanize.Bytes(availableBytes), "totalBytes:", humanize.Bytes(totalBytes))

	return (totalBytes / uint64(100)) * percent, nil
}
