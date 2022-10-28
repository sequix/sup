package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sequix/sup/pkg/buildinfo"
	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/process"
	"github.com/sequix/sup/pkg/run"
)

func main() {
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
	stop := run.SetupSignalHandler()
	process.InitServer()
	serverRw := run.Run(process.Serve)
	log.Info("Sup daemon inited")

	<-stop
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
	case process.ActionStop:
		err = process.Stop()
	case process.ActionRestart:
		err = process.Restart()
	case process.ActionReload:
		err = process.Reload()
	case process.ActionKill:
		err = process.Kill()
	case process.ActionStatus:
		err = process.Status()
	case process.ActionExit:
		err = process.Exit()
	default:
		fmt.Printf("unknown action %q, want one of [start, stop, restart, kill, reload, status, exit]\n", action)
		os.Exit(1)
	}

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
