package checker

import (
	"errors"
	"io"
	"net/http"

	"github.com/ghinknet/regexp"
)

func getPublicIPv4ByIPIPNET() (string, error) {
	var ipRegex = regexp.MustCompile(`当前 IP：(\d+\.\d+\.\d+\.\d+)`)

	ipAPI := "https://myip.ipip.net"

	resp, err := http.Get(ipAPI)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ipMatch := ipRegex.FindStringSubmatch(string(body))

	if len(ipMatch) < 2 {
		return "", errors.New("failed to parse public IP")
	}
	return ipMatch[1], nil
}

func getPublicIPByRaw(api string) (string, error) {
	resp, err := http.Get(api)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func getPublicIPv4ByGhink() (string, error) {
	return getPublicIPByRaw("https://v4-myip.gh.ink")
}

func getPublicIPv6ByGhink() (string, error) {
	return getPublicIPByRaw("https://v6-myip.gh.ink")
}
