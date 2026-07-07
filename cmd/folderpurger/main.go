package main

import (
	"fmt"
	"folder_purger/purger"
	"os"
	"strconv"
	"strings"
	"time"

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

	var folders []*purger.Folder

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
				panic(fmt.Sprintf("Failed to parse destination max size: %s", sizeParam))
			}
			maxSize, err = getDriveFreeSpace(folder, uint64(maxSizePercent))
			if err != nil {
				panic(fmt.Sprintf("Failed to get drive free space: %s", err))
			}
		} else {
			size, err := strconv.Atoi(sizeParam)
			if err != nil {
				panic(fmt.Sprintf("Failed to parse destination max size: %s", sizeParam))
			}
			maxSize = uint64(size)
		}

		if _, err := os.Stat(folder); os.IsNotExist(err) {
			fmt.Printf("Creating folder: %s\n", folder)
			if err := os.MkdirAll(folder, os.ModePerm); err != nil {
				panic(fmt.Sprintf("Failed to create folder %s: %s", folder, err))
			}
		}

		f := &purger.Folder{Path: folder, MaxSize: int64(maxSize)}
		folders = append(folders, f)
		fmt.Println("tracking folder:", folder, "max size:", humanize.Bytes(maxSize))
	}

	p := purger.NewPurger(folders, 5*time.Minute)
	p.Run()
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
