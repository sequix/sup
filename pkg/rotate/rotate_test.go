package rotate

import (
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/sequix/sup/pkg/log"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	//b := make([]rune, n)
	//for i := range b {
	//	b[i] = letterRunes[rand.Intn(len(letterRunes))]
	//}
	//return string(b) + "\n"
	return "When parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basicaWhen parsing a time specified as a string, you have to specify the layout, the format of the input string. This basically tells what part of the input is the year, month, day, hour, min etc. The solution in this answer does print the milliseconds since the epoch. Please click on the link and run the code. "
}

type flog struct {
	interval time.Duration
	w io.WriteCloser
}

func (f *flog) run(stop chan struct{}) {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			line := randString(rand.Intn(80000))
			written, err := f.w.Write([]byte(line))
			if err != nil {
				log.Error("write log: written %d bytes, error %s", written, err)
			}
		}
	}
}

func Test(t *testing.T) {
	// no compress
	// x rotate when reach size limit
	// x clean when reach backups limit
	// restart program no trunc

	//fw, err := NewFileWriter(WithFilename("test.log"),
	//	WithCompress(false),
	//	WithMaxBackups(2),
	//	WithMaxBytes(2 * 1024 * 1024))
	//if err != nil {
	//	panic(err)
	//}
	//defer fw.Close()
	//
	//fg := flog{
	//	interval: time.Millisecond * 100,
	//	w:        fw,
	//}
	//
	//stop := make(chan struct{})
	//
	//go fg.run(stop)
	//
	//time.Sleep(5 * time.Minute)
	//close(stop)

	// compress
	// rotate when reach size limit and gzip it
	// after rotate merge gzip if multiple adjacent gzip not reach the size limit
	// after merge gzip rename the first of the merging bundle to the last of the bundle
	// clean when reach backups limit
	// restart program

	log.Init()

	fw, err := NewFileWriter(WithFilename("test.log"),
		WithCompress(true),
		WithMaxBackups(2),
		WithMaxBytes(2 * 1024 * 1024))
	if err != nil {
		panic(err)
	}
	defer fw.Close()

	fg := flog{
		interval: time.Millisecond * 20,
		w:        fw,
	}

	stop := make(chan struct{})

	go fg.run(stop)

	time.Sleep(5 * time.Minute)
	close(stop)

	// delete backups after the last log of a backup beyond max age
}

func Test2(t *testing.T) {
	f1, _ := os.OpenFile("1", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f2, _ := os.OpenFile("2", os.O_RDONLY, 0644)
	defer f1.Close()
	defer f2.Close()

	spew.Dump(io.Copy(f1, f2))
}