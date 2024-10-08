package main

import (
	"log/slog"
	"os"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)

func main() {
	flagsConfig, err := flags.ParseFlags(os.Args[1:])
	if err != nil {
		slog.Error("failed to parse flags", "err", err)
		return
	}

	if shouldExit := flags.HandleFlags(flagsConfig); shouldExit {
		os.Exit(0)
	}
}
