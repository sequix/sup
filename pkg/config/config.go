package config

import (
	"flag"
	"os"
	"path/filepath"

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

	if !filepath.IsAbs(G.ProgramConfig.Process.WorkDir) {
		log.Fatal("expected an absolute path for process workdir")
	}

	if len(G.SupConfig.Socket) == 0 {
		log.Fatal("expected non-empty socket path")
	}
	if !filepath.IsAbs(G.SupConfig.Socket) {
		G.SupConfig.Socket = filepath.Clean(filepath.Join(G.ProgramConfig.Process.WorkDir, G.SupConfig.Socket))
	}

	if !filepath.IsAbs(G.ProgramConfig.Process.Path) {
		G.ProgramConfig.Process.Path = filepath.Clean(filepath.Join(G.ProgramConfig.Process.WorkDir, G.ProgramConfig.Process.Path))
	}
	stat, err := os.Stat(G.ProgramConfig.Process.Path)
	if err != nil {
		log.Fatal("failed to stat program file %s: %s", G.ProgramConfig.Process.Path, err)
	}
	if (stat.Mode() & 0111) == 0 {
		log.Fatal("program file is not executable: %s", G.ProgramConfig.Process.Path)
	}
}
