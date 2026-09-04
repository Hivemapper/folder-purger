package main

import (
	"fmt"
	"folder_purger/config"
	"folder_purger/purger"
	"os"
	"sort"
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

	folders := configFolders()
	if folders == nil {
		folders = argFolders(argsWithoutProg)
	}

	for _, f := range folders {
		fmt.Println("tracking folder:", f.Path, "max size:", humanize.Bytes(uint64(f.MaxSize)))
	}

	p := purger.NewPurger(folders, 5*time.Minute)
	p.Run()
}

// configFolders returns the folders the device config says to track, or nil to
// fall back to the CLI arguments.
func configFolders() []*purger.Folder {
	limits, err := config.FolderLimits()
	if err != nil {
		fmt.Printf("using CLI folders, no readable config: %s\n", err)
		return nil
	}
	return foldersFor(limits)
}

// foldersFor turns configurator limits into tracked folders, or returns nil if
// none are usable so the caller falls back to the CLI arguments.
func foldersFor(limits map[string]int64) []*purger.Folder {
	if len(limits) == 0 {
		fmt.Println("using CLI folders, config has no folder limits")
		return nil
	}

	paths := make([]string, 0, len(limits))
	for path := range limits {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	folders := make([]*purger.Folder, 0, len(paths))
	for _, path := range paths {
		if err := ensureFolder(path); err != nil {
			fmt.Printf("skipping config folder %s: %s\n", path, err)
			continue
		}
		folders = append(folders, &purger.Folder{Path: path, MaxSize: limits[path]})
	}

	if len(folders) == 0 {
		fmt.Println("using CLI folders, no usable folder from config")
		return nil
	}

	return folders
}

func argFolders(args []string) []*purger.Folder {
	var folders []*purger.Folder

	for i := 0; i < len(args); i += 2 {
		maxSize := uint64(0)
		folder := args[i]
		if strings.HasSuffix(folder, "/") {
			folder = folder[:len(folder)-1]
		}
		sizeParam := args[i+1]
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

		if err := ensureFolder(folder); err != nil {
			panic(fmt.Sprintf("Failed to create folder %s: %s", folder, err))
		}

		folders = append(folders, &purger.Folder{Path: folder, MaxSize: int64(maxSize)})
	}

	return folders
}

func ensureFolder(path string) error {
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		fmt.Printf("Creating folder: %s\n", path)
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
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
