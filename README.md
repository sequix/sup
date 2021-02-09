# sup
Supervisor for single process. Supports CLI, auto-restart, log-rotate-clean based both on time and size.

# Getting Started

```bash
# Start Sup daemon
$ ./sup -c config.toml

# Using CLI
$ ./sup -c config.toml status
$ ./sup -c config.toml start
$ ./sup -c config.toml stop
$ ./sup -c config.toml restart
$ ./sup -c config.toml reload
$ ./sup -c config.toml kill
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
# Start the process as Sup goes up.
autoStart = false
# How to react when the supervised process went down. One of 'on-failure', 'always', 'none'. 'on-failure' by default.
restartStrategy = "on-failure"
# Environment variables to the supervised process.
[program.process.envs]
ENV_VAR1 = "val1"
ENV_VAR2 = "val2"

# Config related with log. Log will be acquired from stdout and stderr.
[program.log]
# Whether the rotated log files should be compressed with gzip, no compression by default.
compress = false
# Maximum number of days to retain old log files based on the UTC time encoded in their filename.
maxAge = 30
# Maximum number of old log files to retain. Retaining all old log files by default.
maxBackups = 32
# Maximum size in megabytes of the log file before it gets rotated. 128 MiB by default.
maxSize = 128
# Filename prefix of the rotated logs. Using basename of the supervised process by default.
path = "./tail.log"
```
