package cli

import (
	flag "github.com/spf13/pflag"
)

type Flags struct {
	DryRun bool
}

var flags = Flags{}

func ParseFlags() Flags {
	flag.BoolVarP(&flags.DryRun, "dry-run", "d", false, "Simulates the process of adding DNS records and proxy hosts without applying changes to Pi-Hole, AdGuard Home or Nginx Proxy Manager.")
	flag.Parse()
	return flags
}
