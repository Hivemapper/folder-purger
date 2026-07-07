package purger

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
)

type Folder struct {
	Path    string
	MaxSize int64
}

type Purger struct {
	folders  []*Folder
	interval time.Duration
}

func NewPurger(folders []*Folder, interval time.Duration) *Purger {
	return &Purger{
		folders:  folders,
		interval: interval,
	}
}

func (p *Purger) Run() {
	for {
		for _, f := range p.folders {
			if err := f.CheckAndPurge(); err != nil {
				log.Printf("error purging %s: %v", f.Path, err)
			}
		}
		time.Sleep(p.interval)
	}
}

func (f *Folder) CheckAndPurge() error {
	size, err := dirSize(f.Path)
	if err != nil {
		return fmt.Errorf("computing size of %s: %w", f.Path, err)
	}

	fmt.Printf("folder %s: %s / %s\n", f.Path, humanize.Bytes(uint64(size)), humanize.Bytes(uint64(f.MaxSize)))

	if size <= f.MaxSize {
		return nil
	}

	entries, err := os.ReadDir(f.Path)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", f.Path, err)
	}

	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}

	if len(dirs) == 0 {
		return nil
	}

	removeCount := len(dirs) * 80 / 100
	if removeCount == 0 {
		removeCount = 1
	}

	var reclaimed int64
	for i := 0; i < removeCount; i++ {
		p := filepath.Join(f.Path, dirs[i].Name())
		s, _ := dirSize(p)
		if err := os.RemoveAll(p); err != nil {
			log.Printf("failed to remove %s: %v", p, err)
			continue
		}
		reclaimed += s
		fmt.Printf("removed %s (%s)\n", p, humanize.Bytes(uint64(s)))
	}

	fmt.Printf("reclaimed %s from %s (removed %d/%d subdirs)\n",
		humanize.Bytes(uint64(reclaimed)), f.Path, removeCount, len(dirs))

	return nil
}

func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}
