package purger

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
)

const purgeBatchPercent = 80

type Folder struct {
	Path    string
	MaxSize int64
}

type Purger struct {
	folders  []*Folder
	interval time.Duration
}

type purgeItem struct {
	name    string
	modTime time.Time
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

	items, err := purgeItems(f.Path)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", f.Path, err)
	}

	if len(items) == 0 {
		return nil
	}

	removeCount := len(items) * purgeBatchPercent / 100
	if removeCount == 0 {
		removeCount = 1
	}

	var reclaimed int64
	for i := 0; i < removeCount; i++ {
		p := filepath.Join(f.Path, items[i].name)
		s, _ := dirSize(p)
		if err := os.RemoveAll(p); err != nil {
			log.Printf("failed to remove %s: %v", p, err)
			continue
		}
		reclaimed += s
		fmt.Printf("removed %s (%s)\n", p, humanize.Bytes(uint64(s)))
	}

	fmt.Printf("reclaimed %s from %s (removed %d/%d items)\n",
		humanize.Bytes(uint64(reclaimed)), f.Path, removeCount, len(items))

	return nil
}

func purgeItems(path string) ([]purgeItem, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	items := make([]purgeItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("failed to stat %s: %v", filepath.Join(path, entry.Name()), err)
			continue
		}
		items = append(items, purgeItem{
			name:    entry.Name(),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.Before(items[j].modTime)
	})

	return items, nil
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
