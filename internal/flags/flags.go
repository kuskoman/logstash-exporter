package flags

import (
	"flag"
	"fmt"
	"os"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// FlagsConfig holds the parsed command-line flags
type FlagsConfig struct {
	ConfigLocation string
	HotReload      bool
	Version        bool
	Help           bool
}

// ParseFlags parses the provided command-line arguments for testability
func ParseFlags(args []string) (*FlagsConfig, error) {
	flags := flag.NewFlagSet("startup_manager", flag.ContinueOnError)

	versionFlag := flags.Bool("version", false, "prints the version and exits")
	helpFlag := flags.Bool("help", false, "prints the help message and exits")
	hotReloadFlag := flags.Bool("watch", false, "enable configuration hot reload")
	configLocationFlag := flags.String("config", config.ExporterConfigLocation, "location of the exporter config file")

	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	// Create a struct with the parsed flags
	return &FlagsConfig{
		ConfigLocation: *configLocationFlag,
		HotReload:      *hotReloadFlag,
		Version:        *versionFlag,
		Help:           *helpFlag,
	}, nil
}

// HandleFlags handles the parsed flags and performs actions based on them
func HandleFlags(flagsConfig *FlagsConfig) bool {
	if flagsConfig.Help {
		printHelp()
		return true
	}

	if flagsConfig.Version {
		printVersion()
		return true
	}

	return false
}

// printHelp prints usage instructions
func printHelp() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

// printVersion prints the application version
func printVersion() {
	fmt.Printf("%s\n", config.SemanticVersion)
}
