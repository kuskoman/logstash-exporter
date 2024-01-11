package config

import (
	"os"
	"time"
)

const (
	defaultHttpTimeout = time.Second * 2
	httpTimeoutEnvVar  = "HTTP_TIMEOUT"
)

func GetHttpTimeout() (time.Duration, error) {
	userDefinedTimeout := os.Getenv(httpTimeoutEnvVar)
	if userDefinedTimeout == "" {
		return defaultHttpTimeout, nil
	}

	timeout, err := time.ParseDuration(userDefinedTimeout)
	if err != nil {
		return 0, err
	}

	return timeout, nil
}
