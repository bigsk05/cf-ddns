package checker

import (
	"cf-ddns/internal/config"
	"cf-ddns/internal/logger"

	"github.com/cloudflare/cloudflare-go"
	"go.uber.org/zap"
)

var API *cloudflare.API

// Init inits checker API
func Init() {
	// Init cloudflare API
	api, err := cloudflare.NewWithAPIToken(config.Get().CF.Token)
	if err != nil {
		logger.L.Panic("failed to initialize Cloudflare API", zap.Error(err))
	}
	API = api

	logger.L.Info("initialized Cloudflare API")
}

// Checker is the check task runner
func Checker() {
	var currentV4, currentV6 string
	var err error

v4branch:
	for {
		switch config.Get().API.V4 {
		case "":
			break v4branch
		case "ipip.net":
			currentV4, err = getPublicIPv4ByIPIPNET()
			if err != nil {
				logger.L.Warn("failed to get public IPv4 by IPIP.NET", zap.Error(err))
				break v4branch
			}
		case "gh.ink":
			currentV4, err = getPublicIPv4ByGhink()
			if err != nil {
				logger.L.Warn("failed to get public IPv4 by GH.INK", zap.Error(err))
				break v4branch
			}
		default:
			currentV4, err = getPublicIPByRaw(config.Get().API.V4, clientV4)
			if err != nil {
				logger.L.Warn("failed to get public IPv4 by raw", zap.Error(err))
				break v4branch
			}
		}

		logger.L.Debug("current IPv4 address", zap.String("IP", currentV4))

		if err = updateDNSIPv4(
			API, config.Get().CF.Zone, config.Get().Record.Name, currentV4,
		); err != nil {
			logger.L.Warn("failed to update DNS IPv4", zap.Error(err), zap.String("IP", currentV4))
		} else {
			logger.L.Info("updated DNS IPv4 successfully", zap.String("IP", currentV4))
		}
		break v4branch
	}

v6branch:
	for {
		switch config.Get().API.V6 {
		case "":
			break v6branch
		case "ipip.net":
			currentV6, err = getPublicIPv6ByIPIPNET()
			if err != nil {
				logger.L.Warn("failed to get public IPv6 by IPIP.NET", zap.Error(err))
				break v6branch
			}
		case "gh.ink":
			currentV6, err = getPublicIPv6ByGhink()
			if err != nil {
				logger.L.Warn("failed to get public IPv6 by GH.INK", zap.Error(err))
				break v6branch
			}
		default:
			currentV6, err = getPublicIPByRaw(config.Get().API.V6, clientV6)
			if err != nil {
				logger.L.Warn("failed to get public IPv6 by raw", zap.Error(err))
				break v6branch
			}
		}

		logger.L.Debug("current IPv6 address", zap.String("IP", currentV6))

		if err = updateDNSIPv6(
			API, config.Get().CF.Zone, config.Get().Record.Name, currentV6,
		); err != nil {
			logger.L.Warn("failed to update DNS IPv6", zap.Error(err), zap.String("IP", currentV6))
		} else {
			logger.L.Info("updated DNS IPv6 successfully", zap.String("IP", currentV6))
		}
		break v6branch
	}
}
