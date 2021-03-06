package process

import (
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/rotate"
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

	supEnvs := os.Environ()
	envsMap := make(map[string]string, len(supEnvs)+len(processConfig.Envs))
	for _, supEnv := range supEnvs {
		eqi := strings.Index(supEnv, "=")
		if eqi == -1 {
			log.Fatal("invalid env %s", supEnv)
		}
		kv := strings.SplitN(supEnv, "=", 2)
		envsMap[kv[0]] = kv[1]
	}
	for k, v := range processConfig.Envs {
		envsMap[k] = v
	}
	envs := make([]string, 0, len(envsMap))
	for k, v := range envsMap {
		envs = append(envs, k+"="+v)
	}
	cmd.Env = envs
	cmd.Dir = processConfig.WorkDir

	logger, err := rotate.NewFileWriter(
		rotate.WithFilename(logConfig.Path),
		rotate.WithMaxBytes(int64(logConfig.MaxSize)*1024*1024),
		rotate.WithMaxBackups(logConfig.MaxBackups),
		rotate.WithCompress(logConfig.Compress),
		rotate.WithMergeCompressedBackups(logConfig.MergeCompressed),
		rotate.WithMaxAge(time.Hour * 24 * time.Duration(logConfig.MaxDays)),
	)
	if err != nil {
		log.Fatal("init rotate logger: %s", err)
	}

	controller = &Controller{
		cmd:      cmd,
		logger:   logger,
		waiterCh: make(chan struct{}),
	}

	server = rpc.NewServer()
	if err := server.Register(controller); err != nil {
		log.Fatal("registry controller to rpc: %s", err)
	}

	socketPath := config.G.SupConfig.Socket
	socketPathDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketPathDir, 0755); err != nil {
		log.Fatal("mkdir %s: %s", socketPathDir, err)
	}

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

func Serve(stop util.BroadcastCh) {
	waiterRw := util.Run(controller.waiter)
	defer waiterRw.StopAndWait()

	go func() {
		<-stop
		controller.close()
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
