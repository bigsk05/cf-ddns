package cron

import (
	"cf-ddns/internal/checker"
	"cf-ddns/internal/config"
	"cf-ddns/internal/logger"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
)

var C *gocron.Scheduler

// checkerJob holds the currently registered checker job and its period so the
// scheduler can be reconfigured when the config is reloaded.
var (
	checkerJob    *gocron.Job
	checkerPeriod int
	checkerMutex  sync.Mutex
)

// Init inits global cron object
func Init() {
	C = gocron.NewScheduler(time.Local)

	registerDefault()

	C.StartAsync()

	// Re-apply the checker period whenever the config is reloaded.
	config.OnReload(reloadChecker)

	logger.L.Debug("cron initialized")
}

// registerDefault registers default cron tasks
func registerDefault() {
	checkerMutex.Lock()
	defer checkerMutex.Unlock()

	period := config.Get().Record.Period
	job, err := C.Every(period).Second().Do(checker.Checker)
	if err != nil {
		logger.L.Fatal("failed to register default cron 'checker.Checker'", zap.Error(err))
	}
	checkerJob = job
	checkerPeriod = period
}

// reloadChecker re-registers the checker job with the latest configured period.
// It is a no-op when the period is unchanged.
func reloadChecker() {
	checkerMutex.Lock()
	defer checkerMutex.Unlock()

	period := config.Get().Record.Period
	if period == checkerPeriod {
		return
	}

	if checkerJob != nil {
		C.RemoveByReference(checkerJob)
	}

	job, err := C.Every(period).Second().Do(checker.Checker)
	if err != nil {
		logger.L.Warn("failed to re-register cron 'checker.Checker' after reload",
			zap.Int("period", period), zap.Error(err))
		return
	}
	checkerJob = job
	checkerPeriod = period

	logger.L.Info("cron checker period updated", zap.Int("period", period))
}
