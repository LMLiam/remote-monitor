package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/lmliam/remote-monitor/internal/monitor"
)

func main() {
	if err := monitor.RunCLI(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "remote-monitor: %v\n", err)
		os.Exit(1)
	}
}
