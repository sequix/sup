package process

import (
	"net"
	"net/rpc"
	"os/exec"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/util"
)

var (
	processConfig *config.Process
	server        *rpc.Server
	controller    *Controller
	unixListener  *net.UnixListener
)

func InitServer() {
	processConfig = &config.G.ProgramConfig.Process
	logConfig := &config.G.ProgramConfig.Log

	cmd := exec.Command(processConfig.Path, processConfig.Args...)

	envs := make([]string, 0, len(processConfig.Envs))
	for k, v := range processConfig.Envs {
		envs = append(envs, k+"="+v)
	}
	cmd.Env = envs
	cmd.Dir = processConfig.WorkDir

	logger := &lumberjack.Logger{
		Filename:   logConfig.Path,
		MaxSize:    logConfig.MaxSize,
		MaxAge:     logConfig.MaxAge,
		MaxBackups: logConfig.MaxBackups,
		Compress:   logConfig.Compress,
		LocalTime:  false,
	}

	controller = &Controller{
		cmd:      cmd,
		logger:   logger,
		waiterCh: make(chan struct{}),
		wantStop: 0,
	}

	server = rpc.NewServer()
	if err := server.Register(controller); err != nil {
		log.Fatal("registry controller to rpc: %s", err)
	}

	socketPath := config.G.SupConfig.Socket
	ua, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		log.Fatal("resolve unix socket path %q: %s", socketPath, err)
	}

	unixListener, err = net.ListenUnix("unix", ua)
	if err != nil {
		log.Fatal("listen to socket %q: %s", socketPath, err)
	}

	if processConfig.AutoStart {
		go controller.handleStart()
	}
}

func Server(stop util.BroadcastCh) {
	waiterRw := util.Run(controller.waiter)
	defer waiterRw.StopAndWait()

	go func() {
		<-stop
		if err := unixListener.Close(); err != nil {
			log.Error("close socket listener: %s", err)
		}
	}()

	for {
		uc, err := unixListener.AcceptUnix()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Error("accept conn: %s", err)
			continue
		}
		go server.ServeConn(uc)
	}
}
