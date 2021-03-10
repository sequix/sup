package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sequix/sup/pkg/buildinfo"
	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/process"
	"github.com/sequix/sup/pkg/util"
)

var (
	stop util.BroadcastCh
)

func main() {
	stop = util.SetupSignalHandler()
	log.Init()
	flag.Parse()
	buildinfo.Init()
	config.Init()

	if len(flag.Args()) == 0 {
		server()
	} else {
		client()
	}
}

func server() {
	process.InitServer()
	serverRw := util.Run(process.Serve)
	log.Info("Sup daemon inited")

	stop.Wait()
	log.Info("recv term signal")

	serverRw.StopAndWait()
	log.Info("Sup daemon finished")
}

func client() {
	var err error
	process.InitClient()
	defer process.ClientClose()

	switch action := flag.Arg(0); action {
	case process.ActionStart:
		err = process.Start()
	case process.ActionStartWait:
		err = process.StartWait()
	case process.ActionStop:
		err = process.Stop()
	case process.ActionStopWait:
		err = process.StopWait()
	case process.ActionRestart:
		err = process.Restart()
	case process.ActionRestartWait:
		err = process.RestartWait()
	case process.ActionReload:
		err = process.Reload()
	case process.ActionKill:
		err = process.Kill()
	case process.ActionStatus:
		err = process.Status()
	case process.ActionExit:
		err = process.Exit()
	case process.ActionExitWait:
		err = process.ExitWait()
	default:
		fmt.Printf("unknown action %q, want one of [start, start-wait, stop, stop-wait, restart, restart-wait, kill, reload, status, exit, exit-wait]\n", action)
		os.Exit(1)
	}

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
