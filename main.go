package main

import (
	"cf-ddns/internal/checker"
	"cf-ddns/internal/config"
	"cf-ddns/internal/cron"
	"cf-ddns/internal/logger"

	"go.gh.ink/timex"
)

func main() {
	// Load public config
	config.Init()
	defer config.Cleanup()

	// Init logger
	logger.Init()
	defer logger.Cleanup()

	// Init checker
	checker.Init()

	// Init cron
	cron.Init()

	timex.Sleep(timex.NewPosInfDuration())
}
