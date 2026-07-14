package main

import (
	"log"

	"github.com/BALLUUNN/authStartGrowingUp/internal/config"
	"github.com/BALLUUNN/authStartGrowingUp/pkg/logger"
)

const startupMessage = "auth service bootstrap is running"

func serviceMessage() string {
	return startupMessage
}

func main() {
	loggerConfig, err := config.LoggerConfig()
	if err != nil {
		log.Fatalf("load logger config: %v", err)
	}

	appLogger, err := logger.New(loggerConfig)
	if err != nil {
		log.Fatalf("build logger: %v", err)
	}
	defer func() {
		if err := appLogger.Sync(); err != nil {
			log.Printf("sync logger: %v", err)
		}
	}()

	appLogger.Info(
		serviceMessage(),
		logger.Action("service_start"),
		logger.Result("success"),
	)
}
