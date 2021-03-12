package rotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sequix/sup/pkg/log"
)

// for unit test
var timeNow = time.Now

// FileWriter is an Writer that writes to the specified filename.
//
// Backups use the log file name given to FileWriter, in the form
// `name.timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05` and the extension is the
// original extension.  For example, if your FileWriter.filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server.2016-11-04T18-30-00.log`
//
// Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted.  The most
// recent files according to filesystem modified time will be retained, up to a
// number equal to maxBackups (or all of them if maxBackups is 0).  Any files
// with an encoded timestamp older than maxAge days are deleted, regardless of
// maxBackups.  Note that the time encoded in the timestamp is the rotation
// time, which may differ from the last time that file was written to.
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

	// compress is the switch to let you control if the log is compressed with gzipCopyClose.
	// The default is not to compress.
	compress bool

	// maxAge is the maximum duration of old log files to retain. The default
	// is to retain all old log files.
	maxAge time.Duration

	// make aligncheck happy
	mu     sync.Mutex
	backMu sync.Mutex
	size   int64
	file   *os.File
	gw     gzip.Writer
}

type Option func(*FileWriter)

func WithFilename(filename string) Option {
	return func(w *FileWriter) {
		w.filename = filename
	}
}

func WithMaxBytes(maxBytes int64) Option {
	return func(w *FileWriter) {
		if maxBytes <= 0 {
		}
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

func WithMaxAge(maxAge time.Duration) Option {
	return func(w *FileWriter) {
		w.maxAge = maxAge
	}
}

func NewFileWriter(opts ...Option) (*FileWriter, error) {
	fw := &FileWriter{
		filename: "rotate.log",
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
	// TODO maxAge
	return fw, nil
}

var _ io.WriteCloser = (*FileWriter)(nil)

// Write implements io.Writer.  If a write would cause the log file to be larger
// than maxBytes, the file is closed, rotate to include a timestamp of the
// current time, and update symlink with log name file to the new file.
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
		w.size = 0
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
	if w.file != nil {
		err = w.file.Close()
		w.file = nil
		w.size = 0
	}
	w.mu.Unlock()
	return
}

// Rotate causes Logger to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside of the normal rotation rules, such as in response to
// SIGHUP.  After rotating, this initiates compression and removal of old log
// files according to the configuration.
func (w *FileWriter) Rotate() (err error) {
	w.mu.Lock()
	err = w.rotate()
	w.mu.Unlock()
	return
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
		w.gzipMerge()
	}
	w.cleanExtraBackups()
	w.backMu.Unlock()
}

func (w *FileWriter) gzip(srcFilename string) {
	src, err := os.OpenFile(srcFilename, os.O_RDONLY|os.O_EXCL, 0644)
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
}

func (w *FileWriter) gzipCopyClose(src io.ReadCloser, dst io.WriteCloser) (written int64, err error) {
	logErr := func(err error) {
		if err != nil {
			log.Error("error on gzip: %s", err)
		}
	}
	w.gw.Reset(dst)
	written, err = io.Copy(&w.gw, src)
	logErr(w.gw.Close())
	logErr(dst.Close())
	logErr(src.Close())
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
		if curBytes+fi.Size() >= w.maxBytes && len(toMerge) > 0 {
			if err := w.mergeToFirstRenameToLast(dir, append(toMerge, fi)); err != nil {
				log.Error(err.Error())
				return
			}
			curBytes = 0
			toMerge = nil
			continue
		}
		toMerge = append(toMerge, fi)
		curBytes += fi.Size()
	}
}

func (w *FileWriter) mergeToFirstRenameToLast(dir string, toMerge []os.FileInfo) error {
	logErr := func(err error) {
		if err != nil {
			log.Error("error on merging gzips: %s", err)
		}
	}

	dstFi := toMerge[0]
	dstFilename := filepath.Join(dir, dstFi.Name())
	dst, err := os.OpenFile(dstFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file %s: %s", dstFilename, err)
	}
	defer logErr(dst.Close())

	for _, srcFi := range toMerge[1:] {
		srcFilename := filepath.Join(dir, srcFi.Name())
		err := func() error {
			src, err := os.OpenFile(srcFilename, os.O_EXCL|os.O_RDONLY, 0644)
			if err != nil {
				return fmt.Errorf("open file %s: %s", dstFilename, err)
			}
			defer logErr(src.Close())
			if written, err := io.Copy(dst, src); err != nil {
				return fmt.Errorf("append gzip: written %d, err %s", written, err)
			}
			if err := os.Remove(srcFilename); err != nil {
				return fmt.Errorf("remove %s: %s", srcFilename)
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
	for i := 0; i < len(fis)-w.maxBackups-1; i++ {
		name := fis[i].Name()
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
	return prefix + now.Format("2006-01-02T15-04-05-999Z") + ext
}
