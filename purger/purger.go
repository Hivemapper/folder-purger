package purger

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fsnotify/fsnotify"
)

type FileInfo struct {
	name             string
	size             int64
	modificationTime time.Time
}

type Folder struct {
	lock        sync.Mutex
	path        string
	maxSize     int64
	currentSize int64
	files       []*FileInfo
	knowFiles   map[string]bool
}

//todo: purge base on remaining space on disk
//HDC-S Gyro is still returning -2000.061037018952

func NewFolder(path string, maxSize int64) *Folder {
	f := &Folder{
		path:      path,
		maxSize:   maxSize,
		knowFiles: map[string]bool{},
	}
	return f
}

func (d *Folder) AddFile(f *FileInfo) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if _, found := d.knowFiles[f.name]; !found {
		d.knowFiles[f.name] = true
		d.files = append(d.files, f)
		d.currentSize += f.size
	}
}

func (d *Folder) loadInitialState() error {
	fmt.Println("loading initial state of folder:", d.path, humanize.Bytes(uint64(d.currentSize)), humanize.Bytes(uint64(d.maxSize)))
	files, err := os.ReadDir(d.path)
	if err != nil {
		return fmt.Errorf("reading gps data path: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		i, err := f.Info()
		if err != nil {
			return fmt.Errorf("getting file info: %s , %w", f.Name(), err)
		}

		fi := &FileInfo{
			name:             f.Name(),
			size:             i.Size(),
			modificationTime: i.ModTime(),
		}
		d.AddFile(fi)
	}
	fmt.Println("loaded initial state of folder:", d.path, humanize.Bytes(uint64(d.currentSize)), humanize.Bytes(uint64(d.maxSize)))
	return nil
}

func (d *Folder) freeUpSpace(nextFileSize int64) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	//fmt.Println("free space: folder:", d.path, humanize.Bytes(uint64(d.currentSize)), "next file size:", humanize.Bytes(uint64(nextFileSize)), "max size:", humanize.Bytes(uint64(d.maxSize)))
	if d.currentSize+nextFileSize > d.maxSize {
		tenPercent := d.maxSize - (d.maxSize * 90 / 100)
		spaceToReclaim := d.currentSize - d.maxSize + tenPercent
		spaceReclaimed := int64(0)

		for spaceToReclaim > 0 {
			fi := d.files[0]
			d.files = d.files[1:]

			fp := path.Join(d.path, fi.name)
			if fileExists(fp) {
				err := os.Remove(fp)
				if err != nil {
					return fmt.Errorf("removing file: %s, %w", fp, err)
				}
			} else {
				log.Println("free space: skipping file that does not exist anymore: ", fp)
			}

			delete(d.knowFiles, fi.name)
			d.currentSize -= fi.size
			spaceToReclaim -= fi.size
			spaceReclaimed += fi.size
		}
		fmt.Println("reclaimed space for folder:", d.path, humanize.Bytes(uint64(spaceReclaimed)), "new size:", humanize.Bytes(uint64(d.currentSize)), "next file size:", humanize.Bytes(uint64(nextFileSize)), "max size:", humanize.Bytes(uint64(d.maxSize)))
	}

	return nil
}

type Purger struct {
	folders map[string]*Folder
	watcher *fsnotify.Watcher
}

func NewPurger(folders map[string]*Folder) *Purger {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("NewWatcher failed: ", err)
	}
	return &Purger{
		folders: folders,
		watcher: watcher,
	}
}

func (p *Purger) Purge() error {

	for _, folder := range p.folders {

		if !fileExists(folder.path) {
			fmt.Printf("Creating folder: %s\n", folder.path)
			err := os.MkdirAll(folder.path, os.ModePerm)
			if err != nil {
				return fmt.Errorf("creating sourceFolder %s : %w", folder.path, err)
			}
		}

		fmt.Printf("About to watch folder: %s\n", folder.path)
		err := p.watcher.Add(folder.path)
		if err != nil {
			log.Fatal(fmt.Sprintf("adding folder to watcher %s: %s", folder.path, err))
		}

		err = folder.loadInitialState()
		if err != nil {
			return fmt.Errorf("loading initial state of destination %s : %w", folder.path, err)
		}
	}

	err := p.purge() //blocking call
	if err != nil {
		return fmt.Errorf("moving files: %w", err)
	}

	return nil
}

func (p *Purger) purge() error {
	for {
		select {
		case event, ok := <-p.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			if event.Op == fsnotify.Create {
				//if strings.HasSuffix(event.Name, "jpg") {
				err := p.watchFile(event.Name)
				if err != nil {
					return fmt.Errorf("watching file %s: %w", event.Name, err)
				}
				//}
			}
		case err, ok := <-p.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			log.Println("error:", err)
		}
	}
}

func (p *Purger) watchFile(f string) error {
	dir := filepath.Dir(f)
	fileName := filepath.Base(f)

	if folder, ok := p.folders[dir]; ok {
		if stat, err := os.Stat(f); err == nil {
			fp := path.Join(folder.path, fileName)
			if fileExists(f) {
				folder.AddFile(&FileInfo{
					name:             fileName,
					size:             stat.Size(),
					modificationTime: stat.ModTime(),
				})
				err := folder.freeUpSpace(stat.Size())
				if err != nil {
					return fmt.Errorf("freeing space: %w", err)
				}
			} else {
				log.Println("watch: skipping file that does not exist anymore: ", fp)
			}

		}
	} else {
		log.Println("watch: skipping file that does not belong to a tracked folder: ", f, dir)
	}
	return nil
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}
