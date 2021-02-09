package config

import (
	"flag"
	"os"
	"os/exec"

	"github.com/pelletier/go-toml"

	"github.com/sequix/sup/pkg/log"
)

var (
	flagConfigPath = flag.String("c", "", "config path")
)

var (
	G = &Config{}
)

func Init() {
	if len(*flagConfigPath) == 0 {
		log.Fatal("need specify config path with flag -c")
	}

	tf, err := toml.LoadFile(*flagConfigPath)
	if err != nil {
		log.Fatal("read config %q: %s", *flagConfigPath, err)
	}

	if err := tf.Unmarshal(G); err != nil {
		log.Fatal("unmarshal config: %s", err)
	}

	if len(G.ProgramConfig.Process.RestartStrategy) == 0 {
		G.ProgramConfig.Process.RestartStrategy = RestartStrategyOnFailure
	}

	if len(G.ProgramConfig.Process.WorkDir) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("get working dir: %s", err)
		}
		G.ProgramConfig.Process.WorkDir = wd
	}

	path, err := exec.LookPath(G.ProgramConfig.Process.Path)
	if err != nil {
		log.Fatal("lookup binary path: %s", err)
	}
	G.ProgramConfig.Process.Path = path
}
