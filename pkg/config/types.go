package config

type Config struct {
	SupConfig     Sup     `toml:"sup" comment:"Config related with Sup."`
	ProgramConfig Program `toml:"program" comment:"Config related with the supervised process."`
}

type Sup struct {
	Socket string `toml:"socket" comment:"Path to an unix socket, to which Sup daemon will be listening." default:"./sup.sock"`
}

type Program struct {
	Process Process `toml:"process" comment:"Config related with process."`
	Log     Log     `toml:"log" comment:"Config related with log."`
}

type Process struct {
	Path            string                 `toml:"path" comment:"Path to an executable, which would spawn the supervised process."`
	Args            []string               `toml:"args" comment:"Arguments to the supervised process."`
	Envs            map[string]string      `toml:"envs" comment:"Environment variables to the supervised process."`
	WorkDir         string                 `toml:"workDir" comment:"Working directory of the supervised process. Current directory by default." default:"./"`
	AutoStart       bool                   `toml:"autoStart" comment:"Start the process as Sup goes up. False by default." default:"false"`
	RestartStrategy ProcessRestartStrategy `toml:"restartStrategy" comment:"How to react when the supervised process went down. One of 'on-failure', 'always', 'none'. 'on-failure' by default." default:"on-failure"`
}

// How to react when the supervised process went down.
type ProcessRestartStrategy string

const (
	RestartStrategyAlways    ProcessRestartStrategy = "always"
	RestartStrategyOnFailure ProcessRestartStrategy = "on-failure"
	RestartStrategyNone      ProcessRestartStrategy = "none"
)

type Log struct {
	Path       string `toml:"path" comment:"Path where to save the current un-rotated log. Using basename of the supervised process by default."`
	MaxSize    int    `toml:"maxSize" comment:"Maximum size in megabytes of the log file before it gets rotated. 128 MiB by default." default:"128"`
	MaxAge     int    `toml:"maxAge" comment:"Maximum number of days to retain old log files based on the UTC time encoded in their filename. 30 by default." default:"30"`
	MaxBackups int    `toml:"maxBackups" comment:"Maximum number of old log files to retain. Retaining all old log files by default. 32 by default." default:"32"`
	Compress   bool   `toml:"compress" comment:"Whether the rotated log files should be compressed with gzip, no compression by default." default:"false"`
}