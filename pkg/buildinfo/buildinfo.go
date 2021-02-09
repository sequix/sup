package buildinfo

import (
	"flag"
	"fmt"
	"os"
)

var version = flag.Bool("v", false, "show version")

// Must be set via -ldflags '-X'
var (
	Commit string
)

// Init must be called after flag.Parse call.
func Init() {
	if *version {
		printVersionAndHelp()
		os.Exit(0)
	}
}

func init() {
	flag.Usage = printVersionAndHelp
}

func printVersionAndHelp() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -h                     # show this message\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -v                     # show this message\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml         # start sup daemon\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml start   # start program\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml stop    # stop program\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml restart # restart program\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml reload  # reload program\n")
	fmt.Fprintf(flag.CommandLine.Output(), "sup -c config.toml kill    # kill program\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Sup Commit ID: %s\n", Commit)
}