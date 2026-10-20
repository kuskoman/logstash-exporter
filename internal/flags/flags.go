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

// newFlagSet defines the command-line flags and binds them to flagsConfig
func newFlagSet(flagsConfig *FlagsConfig) *flag.FlagSet {
	flags := flag.NewFlagSet("startup_manager", flag.ContinueOnError)

	flags.BoolVar(&flagsConfig.Version, "version", false, "prints the version and exits")
	flags.BoolVar(&flagsConfig.Help, "help", false, "prints the help message and exits")
	flags.BoolVar(&flagsConfig.HotReload, "watch", false, "enable configuration hot reload")
	flags.StringVar(&flagsConfig.ConfigLocation, "config", config.ExporterConfigLocation, "location of the exporter config file")

	return flags
}

// ParseFlags parses the provided command-line arguments for testability
func ParseFlags(args []string) (*FlagsConfig, error) {
	flagsConfig := &FlagsConfig{}
	if err := newFlagSet(flagsConfig).Parse(args); err != nil {
		return nil, err
	}

	return flagsConfig, nil
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
	flags := newFlagSet(&FlagsConfig{})
	flags.SetOutput(os.Stdout)
	flags.PrintDefaults()
}

// printVersion prints the application version
func printVersion() {
	fmt.Printf("%s\n", config.SemanticVersion)
}
