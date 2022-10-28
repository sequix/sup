[![Build Status](https://github.com/sequix/sup/workflows/main/badge.svg)](https://github.com/sequix/sup/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/sequix/sup)](https://goreportcard.com/report/github.com/sequix/sup)

# sup
Supervisor for single process. Supports CLI, auto-restart, log-rotate-clean based both on time and size.

A statically linked binary only and reduced size 1.59MiB with 'strip -s' and 'upx -9'.

You can download the binary [here](https://github.com/sequix/sup/releases).

# Getting Started

```bash
# Start Sup daemon
$ ./sup -c config.toml

# Using CLI
$ ./sup -c config.toml start        # Start the process.
$ ./sup -c config.toml stop         # Stop the process by sending SIGTERM(15).
$ ./sup -c config.toml restart      # Equilevant to Stop & Start.
$ ./sup -c config.toml reload       # Send SIGHUP(1) to the process.
$ ./sup -c config.toml kill         # Send SIGKILL(9) to the process and all its child processes.
$ ./sup -c config.toml status       # Show the process status.
$ ./sup -c config.toml exit         # Exit the Sup daemon and the process.

# General directory format
.
├── bin
│   └── flog # binary file, get it here: https://github.com/mingrammer/flog/releases
├── conf
│   ├── flog.sh   # command to start nginx
│   └── flog.toml # config of Sup 
├── log
│   ├── flog-20210302085409.log
│   ├── flog-20210302092412.log # rotated logs, filename encoded with UTC datetime
│   ├── flog-20210302095926.log # the datetime is the instant when the last write taken place
│   └── flog.log                # output of the combination of stdout and stderr of flog
├── sup           # Sup binary
└── sup.d
    ├── flog.sock # socket the Sup CLI will connect to
    └── flog.log  # log of Sup

# Content of flog.sh
exec ./bin/flog -l

# Content of flog.toml
[sup]
socket = "./sup.d/flog.sock"  # Recommand using absolute path in production.

[program]
[program.process]
path = "/bin/sh"
args = ["./conf/flog.sh"]
workDir = "./"
startSeconds = 5
autoStart = true
restartStrategy = "on-failure"

[program.log]
path = "./log/flog.log"
compress = false
maxDays = 30
maxBackups = 2
maxSize = 16
```

# Config 

```toml
# Config related with Sup.
[sup]
# Path to an unix socket, to which Sup daemon will be listening.
socket = "./sup.sock"

# Config related with the supervised process.
[program]
# Config related with process.
[program.process]
# Path to an executable, which would spawn the supervised process.
path = "/bin/sleep"
# Arguments to the supervised process.
args = ["5"]
# Working directory of the supervised process. Current directory by default.
workDir = "./"
# Start the process as Sup goes up. False by default.
autoStart = false
# Sup waits 'startSeconds' after each start to avoid the process restarts too rapidly.
startSeconds = 5
# How to react when the supervised process went down. One of 'on-failure', 'always', 'none'. 'on-failure' by default.
restartStrategy = "on-failure"
# Environment variables to the supervised process.
[program.process.envs]
ENV_VAR1 = "val1"
ENV_VAR2 = "val2"

# Config related with log. Log will be acquired from stdout and stderr only.
[program.log]
# Path where to save the current un-rotated log. Using basename of the supervised process by default.
path = "./tail.log"
# Whether the rotated log files should be compressed with gzip, no compression by default.
compress = false
# Whether the gzipped backups would be merged or not, no merging by default.
mergeCompressed = false
# Maximum days to retain old log files based on the UTC time encoded in their filename.
maxDays = 30
# Maximum number of old log files to retain. Retaining all old log files by default.
maxBackups = 32
# Maximum size in MiB of the log file before it gets rotated. 128 MiB by default.
maxSize = 128
```

# FAQs

1.Can I reload the config of sup itself?

No. I recommand to use a config file for the program to leave the config of sup immutable.

For those using command line flags (like [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics)), write a bash shell with `exec` command to execute the actual program,

so that `restart` could sense the changes of flags, just like the example above.
