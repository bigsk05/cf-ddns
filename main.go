package main

import (
	"cf-ddns/internal/checker"
	"cf-ddns/internal/config"
	"cf-ddns/internal/cron"
	"cf-ddns/internal/logger"
)

func main() {
	// Load public config
	config.LoadStatic()

	// Init logger
	logger.InitLogger()

	// Init checker
	checker.InitChecker()

	// Init cron
	cron.InitCron()

	select {}
}
