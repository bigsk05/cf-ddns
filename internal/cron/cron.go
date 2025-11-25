package cron

import (
	"cf-ddns/internal/checker"
	"cf-ddns/internal/config"
	"cf-ddns/internal/logger"
	"fmt"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var C *cron.Cron

// InitCron inits global cron object
func InitCron() {
	C = cron.New()

	registerDefault()

	C.Start()

	logger.L.Debug("Cron initialized")
}

// registerDefault registers default cron tasks
func registerDefault() {
	// Register checker
	checker.Checker()

	_, err := C.AddFunc(fmt.Sprintf("@every %ds", config.C.GetInt("record.period")), checker.Checker)
	if err != nil {
		logger.L.Panic("Failed to register update cron", zap.Error(err))
	}
}
