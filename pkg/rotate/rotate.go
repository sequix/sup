package rotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/run"
)

// for unit test
var timeNow = time.Now

var _ io.WriteCloser = (*FileWriter)(nil)

type FileWriter struct {
	// filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	filename string

	// maxBytes is the maximum size in bytes of the log file before it gets rotated.
	// The default is not to rotate at all.
	maxBytes int64

	// maxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files.
	maxBackups int

	// compress decides whether the log is compressed with gzip.
	// The default is not to compress.
	compress bool

	// mergeCompressedBackups decides whether the gzip backups is merged if they below maxBytes.
	// The default is not to merge.
	mergeCompressedBackups bool

	// maxAge is the maximum duration of old log files to retain. The default
	// is to retain all old log files.
	maxAge time.Duration

	// make align check happy
	mu     sync.Mutex
	backMu sync.Mutex
	size   int64
	file   *os.File
	stop   *run.Runner
}

type Option func(*FileWriter)

func WithFilename(filename string) Option {
	return func(w *FileWriter) {
		w.filename = filename
	}
}

func WithMaxBytes(maxBytes int64) Option {
	return func(w *FileWriter) {
		w.maxBytes = maxBytes
	}
}

func WithMaxBackups(maxBackups int) Option {
	return func(w *FileWriter) {
		w.maxBackups = maxBackups
	}
}

func WithCompress(compress bool) Option {
	return func(w *FileWriter) {
		w.compress = compress
	}
}

func WithMergeCompressedBackups(merge bool) Option {
	return func(w *FileWriter) {
		w.mergeCompressedBackups = merge
	}
}

func WithMaxAge(maxAge time.Duration) Option {
	return func(w *FileWriter) {
		w.maxAge = maxAge
	}
}

func NewFileWriter(opts ...Option) (*FileWriter, error) {
	fw := &FileWriter{
		maxBytes: 128 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(fw)
	}
	if len(fw.filename) == 0 {
		return nil, fmt.Errorf("expected non-empty filename")
	}
	if fw.maxBytes <= 0 {
		return nil, fmt.Errorf("expected maxBytes >= 0, got %d", fw.maxBytes)
	}
	if fw.maxAge > 0 {
		fw.stop = run.Run(fw.ager)
	}
	return fw, nil
}

// Write implements io.Writer.  If a write would cause the log file to be larger
// than maxBytes, the file is closed, rotate to include a timestamp of the
// current time.
func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	n, err = w.write(p)
	w.mu.Unlock()
	return
}

func (w *FileWriter) write(p []byte) (n int, err error) {
	if w.file == nil {
		if err = os.MkdirAll(filepath.Dir(w.filename), 0755); err != nil {
			return
		}
		w.file, err = os.OpenFile(w.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		var stat os.FileInfo
		stat, err = w.file.Stat()
		if err != nil {
			return
		}
		w.size = stat.Size()
	}

	n, err = w.file.Write(p)
	if err != nil {
		return
	}

	w.size += int64(n)
	if w.maxBytes > 0 && w.size > w.maxBytes {
		err = w.rotate()
	}

	return
}

// Close implements io.Closer, and closes the current logfile.
func (w *FileWriter) Close() (err error) {
	w.mu.Lock()
	w.backMu.Lock()
	if w.file != nil {
		err = w.file.Close()
		w.file = nil
		w.size = 0
	}
	if w.stop != nil {
		w.stop.StopAndWait()
	}
	w.backMu.Unlock()
	w.mu.Unlock()
	return
}

func (w *FileWriter) ager(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-stop:
			ticker.Stop()
			return
		case now := <-ticker.C:
			w.backMu.Lock()
			w.cleanAgedBackups(now)
			w.backMu.Unlock()
		}
	}
}

func (w *FileWriter) cleanAgedBackups(now time.Time) {
	dir := filepath.Dir(w.filename)
	fis, err := w.listBackups()
	if err != nil {
		log.Error(err.Error())
		return
	}
	for _, fi := range fis {
		backupNow := w.parseTimeFromBackup(fi.Name())
		if now.Sub(backupNow) > w.maxAge {
			filename := filepath.Join(dir, fi.Name())
			if err := os.Remove(filename); err != nil {
				log.Error("remove %s: %s", filename, err)
			} else {
				log.Info("deleted aged log %s", fi.Name())
			}
		}
	}
}

func (w *FileWriter) rotate() error {
	if w.file == nil {
		return nil
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	w.file = nil
	w.size = 0

	rotatedFilename := w.rotatedFilename(timeNow())

	if err := os.Rename(w.filename, rotatedFilename); err != nil {
		return err
	}
	go w.rotateBackground(rotatedFilename)
	return nil
}

func (w *FileWriter) rotateBackground(rotatedFilename string) {
	w.backMu.Lock()
	if w.compress {
		w.gzip(rotatedFilename)
		if w.mergeCompressedBackups {
			w.gzipMerge()
		}
	}
	w.cleanExtraBackups()
	w.backMu.Unlock()
}

func (w *FileWriter) gzip(srcFilename string) {
	src, err := os.OpenFile(srcFilename, os.O_RDONLY, 0644)
	if err != nil {
		log.Error("gzip open src file %s: %s", srcFilename, err)
		return
	}
	dstFilename := srcFilename + ".gz"
	dst, err := os.OpenFile(dstFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("gzip open dst file %s: %s", dstFilename, err)
		return
	}
	if _, err = w.gzipCopyClose(src, dst); err != nil {
		log.Error("gzip file %s: %s", dstFilename, err)
		return
	}
	if err := os.Remove(srcFilename); err != nil {
		log.Error("remove file %s: %s", srcFilename, err)
		return
	}
}

func (w *FileWriter) gzipCopyClose(src io.ReadCloser, dst io.WriteCloser) (written int64, err error) {
	gw := gzip.NewWriter(dst)
	written, err = io.Copy(gw, src)
	log.ErrorFunc(gw.Close, "gzip")
	log.ErrorFunc(dst.Close, "gzip")
	log.ErrorFunc(src.Close, "gzip")
	return
}

func (w *FileWriter) gzipMerge() {
	dir := filepath.Dir(w.filename)
	fis, err := w.listBackups()
	if err != nil {
		log.Error(err.Error())
		return
	}
	var (
		curBytes int64
		toMerge  []os.FileInfo
	)
	for _, fi := range fis {
		if curBytes+fi.Size() >= w.maxBytes {
			if len(toMerge) > 1 {
				if err := w.mergeToFirstRenameToLast(dir, toMerge); err != nil {
					log.Error(err.Error())
					return
				}
			}
			curBytes = 0
			toMerge = nil
		}
		if fi.Size() < w.maxBytes {
			toMerge = append(toMerge, fi)
			curBytes += fi.Size()
		}
	}
	if curBytes <= w.maxBytes && len(toMerge) > 1 {
		if err := w.mergeToFirstRenameToLast(dir, toMerge); err != nil {
			log.Error(err.Error())
			return
		}
	}
}

func (w *FileWriter) mergeToFirstRenameToLast(dir string, toMerge []os.FileInfo) error {
	dstFi := toMerge[0]
	dstFilename := filepath.Join(dir, dstFi.Name())
	dst, err := os.OpenFile(dstFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open file %s: %s", dstFilename, err)
	}
	defer log.ErrorFunc(dst.Close, "close merge dst %s", dstFilename)

	for _, srcFi := range toMerge[1:] {
		err := func() error {
			srcFilename := filepath.Join(dir, srcFi.Name())
			src, err := os.Open(srcFilename)
			if err != nil {
				return fmt.Errorf("open file %s: %s", srcFilename, err)
			}
			defer log.ErrorFunc(src.Close, "close merge src %s", srcFilename)
			if written, err := io.Copy(dst, src); err != nil {
				return fmt.Errorf("append gzip: written %d, err %s", written, err)
			}
			if err := os.Remove(srcFilename); err != nil {
				return fmt.Errorf("remove %s: %s", srcFilename, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	newDstFilename := filepath.Join(dir, toMerge[len(toMerge)-1].Name())
	return os.Rename(dstFilename, newDstFilename)
}

func (w *FileWriter) cleanExtraBackups() {
	dir := filepath.Dir(w.filename)
	fis, err := w.listBackups()
	if err != nil {
		log.Error(err.Error())
		return
	}
	if len(fis) <= w.maxBackups {
		return
	}
	for _, fi := range fis[:len(fis)-w.maxBackups] {
		name := fi.Name()
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			log.Error("remove backup file %s: %s", name, err)
		}
	}
}

func (w *FileWriter) listBackups() ([]os.FileInfo, error) {
	dir := filepath.Dir(w.filename)
	dirfile, err := os.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("open dir %s: %s", dir, err)
	}

	infos, err := dirfile.Readdir(-1)
	if err := dirfile.Close(); err != nil {
		log.Warn("close dir %s", dir)
	}
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %s", dir, err)
	}
	base := filepath.Base(w.filename)
	ext := filepath.Ext(w.filename)
	prefix := base[:len(base)-len(ext)]

	matches := make([]os.FileInfo, 0)
	for _, info := range infos {
		name := info.Name()
		if name != base && strings.HasPrefix(name, prefix) {
			matches = append(matches, info)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name() < matches[j].Name()
	})
	return matches, nil
}

// rotatedFilename returns a new filename based on the original name and the given time.
func (w *FileWriter) rotatedFilename(now time.Time) string {
	now = now.UTC()
	ext := filepath.Ext(w.filename)
	prefix := w.filename[:len(w.filename)-len(ext)]
	filename := prefix + "-" + now.Format("20060102150405")
	if len(ext) > 0 {
		filename = filename + ext
	}
	return filename
}

var reTimeFromBackup = regexp.MustCompile(`^.*-([0-9]{14})(\..*)?$`)

func (w *FileWriter) parseTimeFromBackup(filename string) time.Time {
	m := reTimeFromBackup.FindStringSubmatch(filepath.Base(filename))
	if len(m) != 3 {
		log.Error("invalid backup filename format: %s", filename)
		return time.Unix(math.MaxInt64, 0)
	}
	t, err := time.Parse("20060102150405", m[1])
	if err != nil {
		log.Error("invalid backup filename format: %s", filename)
		return time.Unix(math.MaxInt64, 0)
	}
	return t
}
