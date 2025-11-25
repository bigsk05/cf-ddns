package checker

import (
	"cf-ddns/internal/config"
	"cf-ddns/internal/logger"

	"github.com/cloudflare/cloudflare-go"
	"go.uber.org/zap"
)

var API *cloudflare.API

// InitChecker inits checker API
func InitChecker() {
	// Init cloudflare API
	api, err := cloudflare.NewWithAPIToken(config.C.GetString("cf.token"))
	if err != nil {
		logger.L.Panic("Failed to initialize Cloudflare API", zap.Error(err))
	}
	API = api

	logger.L.Info("Initialized Cloudflare API")
}

// Checker is the check task runner
func Checker() {
	var currentV4, currentV6 string
	var err error

v4branch:
	for {
		switch config.C.GetString("api.v4") {
		case "ipip.net":
			currentV4, err = getPublicIPv4ByIPIPNET()
			if err != nil {
				logger.L.Warn("Failed to get public IPv4 by IPIP.NET", zap.Error(err))
				break v4branch
			}
		case "gh.ink":
			currentV4, err = getPublicIPv4ByGhink()
			if err != nil {
				logger.L.Warn("Failed to get public IPv4 by GH.INK", zap.Error(err))
				break v4branch
			}
		default:
			currentV4, err = getPublicIPByRaw(config.C.GetString("api.v4"))
			if err != nil {
				logger.L.Panic("Failed to get public IPv4 by raw", zap.Error(err))
				break v4branch
			}
		}

		logger.L.Debug("Current IPv4 address", zap.String("IP", currentV4))

		if err = updateDNSIPv4(
			API, config.C.GetString("cf.zone"), config.C.GetString("record.name"), currentV4,
		); err != nil {
			logger.L.Warn("Failed to update DNS IPv4", zap.Error(err), zap.String("IP", currentV4))
		} else {
			logger.L.Info("Updated DNS IPv4 successfully", zap.String("IP", currentV4))
		}
		break v4branch
	}

v6branch:
	for {
		switch config.C.GetString("api.v6") {
		case "gh.ink":
			currentV6, err = getPublicIPv6ByGhink()
			if err != nil {
				logger.L.Warn("Failed to get public IPv6 by GH.INK", zap.Error(err))
				break v6branch
			}
		default:
			currentV6, err = getPublicIPByRaw(config.C.GetString("api.v6"))
			if err != nil {
				logger.L.Warn("Failed to get public IPv6 by raw", zap.Error(err))
				break v6branch
			}
		}

		logger.L.Debug("Current IPv6 address", zap.String("IP", currentV6))

		if err = updateDNSIPv6(
			API, config.C.GetString("cf.zone"), config.C.GetString("record.name"), currentV6,
		); err != nil {
			logger.L.Warn("Failed to update DNS IPv6", zap.Error(err), zap.String("IP", currentV6))
		} else {
			logger.L.Info("Updated DNS IPv6 successfully", zap.String("IP", currentV6))
		}
		break v6branch
	}
}
