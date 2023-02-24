package helpers

import "go.uber.org/zap"

var Logger *zap.SugaredLogger

func InitializeLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	Logger = logger.Sugar()

	return nil
}
