package startup_manager

import (
	"flag"
	"fmt"
	"os"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

type flagsConfig struct {
	configLocation *string
}

func handleFlags() (*flagsConfig, bool) {
	versionFlag := flag.Bool("version", false, "prints the version and exits")
	helpFlag := flag.Bool("help", false, "prints the help message and exits")
	configLocationFlag := flag.String("config", config.ExporterConfigLocation, "location of the exporter config file")

	flag.Parse()

	if *helpFlag {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()

		return nil, true
	}

	if *versionFlag {
		fmt.Printf("%s\n", config.SemanticVersion)

		return nil, true
	}

	flagsConfig := &flagsConfig{
		configLocation: configLocationFlag,
	}

	return flagsConfig, false
}
