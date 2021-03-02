package test

import (
	"fmt"
	"os/exec"
	"testing"
	"time"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func TestC(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "2")

	cmd.Start()

	/*
		[root@vm-h7xn7vng-2-u-huhehaote-rdtq4 test]# go test -v
		=== RUN   TestC
		2: time 2021-02-09T14:27:24+08:00, stat exit status 0, err %!s(<nil>)
		1: time 2021-02-09T14:27:24+08:00, stat <nil>, err wait: no child processes
		--- PASS: TestC (3.00s)
		PASS
		ok  	test	3.003s
	*/
	go func() {
		procStat, err := cmd.Process.Wait()
		fmt.Printf("1: time %s, stat %s, err %s\n", time.Now().Format(time.RFC3339), procStat, err)
	}()

	go func() {
		procStat, err := cmd.Process.Wait()
		fmt.Printf("2: time %s, stat %s, err %s\n", time.Now().Format(time.RFC3339), procStat, err)
	}()

	/*
		=== RUN   TestC
		1 err exec: Wait was already called
		2 err %!s(<nil>)
		--- PASS: TestC (3.00s)
		PASS
		ok  	test	3.003s
	*/
	//go func() {
	//	 err := cmd.Wait()
	//	fmt.Printf("1 err %s\n", err)
	//}()
	//
	//go func() {
	//	 err := cmd.Wait()
	//	fmt.Printf("2 err %s\n", err)
	//}()

	time.Sleep(3 * time.Second)
}
