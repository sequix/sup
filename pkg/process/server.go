package process

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/rotate"
	"github.com/sequix/sup/pkg/run"
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
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if processConfig.User != "" {
		uid, err := getUid(processConfig.User)
		if err != nil {
			log.Fatal(err.Error())
		}
		if cmd.SysProcAttr.Credential == nil {
			cmd.SysProcAttr.Credential = &syscall.Credential{}
		}
		cmd.SysProcAttr.Credential.Uid = uid
	}

	if processConfig.Group != "" {
		gid, err := getGid(processConfig.Group)
		if err != nil {
			log.Fatal(err.Error())
		}
		if cmd.SysProcAttr.Credential == nil {
			cmd.SysProcAttr.Credential = &syscall.Credential{}
		}
		cmd.SysProcAttr.Credential.Gid = gid
	}

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
		rotate.WithMaxAge(time.Hour*24*time.Duration(logConfig.MaxDays)),
	)
	if err != nil {
		log.Fatal("init rotate logger: %s", err)
	}

	controller = &Controller{
		cmd:       cmd,
		logger:    logger,
		startedCh: make(chan struct{}),
		exitedCh:  make(chan struct{}),
		wantStop:  0,
		wantExit:  0,
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
	if err := removeNotUsingSocket(socketPath); err != nil {
		log.Fatal(err.Error())
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
		go func() { _ = controller.startHandler() }()
	}
}

func removeNotUsingSocket(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat socket %s: %s", path, err)
	}
	if (stat.Mode() & os.ModeSocket) == 0 {
		return fmt.Errorf("not a socket file %s", path)
	}
	sockBytes, err := os.ReadFile("/proc/net/unix")
	if err != nil {
		return fmt.Errorf("failed to read /proc/net/unix: %s", err)
	}
	var sockInode string
	for _, line := range strings.Split(string(sockBytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, path) {
			fields := strings.Fields(line)
			if len(fields) != 8 {
				return fmt.Errorf("invalid line in /proc/net/unix: %q", line)
			}
			sockInode = fields[6]
			break
		}
	}
	if len(sockInode) == 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove socket %s: %s", path, err)
		}
		log.Info("deleted socket file not used by any process: %s", path)
		return nil
	}
	return fmt.Errorf("socket is being used by other process: %s", path)
}

func Serve(stop <-chan struct{}) {
	controllerRw := run.Run(controller.run)

	go func() {
		<-stop
		controllerRw.StopAndWait()
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

func getUid(username string) (uint32, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, fmt.Errorf("failed to get uid of user %q: %s", username, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("invalid uid %q: %s", u.Uid, err)
	}
	return uint32(uid), nil
}

func getGid(group string) (uint32, error) {
	g, err := user.LookupGroup(group)
	if err != nil {
		return 0, fmt.Errorf("failed to get gid of group %q: %s", group, err)
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, fmt.Errorf("invalid gid %q: %s", g.Gid, err)
	}
	return uint32(gid), nil
}
