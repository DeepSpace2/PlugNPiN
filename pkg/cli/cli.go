package cli

import (
	flag "github.com/spf13/pflag"
)

type f struct {
	DryRun bool
}

var flags f = f{}

func ParseFlags() f {
	flag.BoolVarP(&flags.DryRun, "dry-run", "d", false, "Simulates the process of adding DNS records and proxy hosts without making any actual changes to Pi-hole or Nginx Proxy Manager.")
	flag.Parse()
	return flags
}
