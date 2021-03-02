package test

import (
	"fmt"
	"os/exec"
	"syscall"
	"testing"

	"github.com/prometheus/procfs"
	"github.com/sanity-io/litter"
)

func TestB(t *testing.T) {
	cmd := exec.Command("/usr/bin/tail", "-f", "/dev/null")
	//cmd.SysProcAttr = &syscall.SysProcAttr{}

	// 进程未开始：cmd.Process == nil
	//fmt.Println( litter.Sdump(cmd))
	//proc, err := procfs.NewProc(cmd.Process.Pid)
	//panicIfErr(err)

	// 进程未开始：cmd.Process != nil
	fmt.Println(cmd.Start())
	//fmt.Println(litter.Sdump(cmd))

	proc, err := procfs.NewProc(cmd.Process.Pid)
	panicIfErr(err)

	fmt.Println("started")
	procStat, err := proc.Stat()
	panicIfErr(err)
	litter.Dump(procStat)

	fmt.Println(cmd.Process.Signal(syscall.SIGTERM))
	litter.Dump(cmd)

	fmt.Println("sigtermed")
	procStat, err = proc.Stat()
	panicIfErr(err)
	litter.Dump(procStat)

	fmt.Println("waited")
	state, err := cmd.Process.Wait()
	panicIfErr(err)
	litter.Dump(state)

	//procStat, err = proc.Stat()
	//panicIfErr(err)
	//litter.Dump(procStat)
	litter.Dump(cmd)
}
