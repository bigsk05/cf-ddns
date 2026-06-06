package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.gh.ink/regexp"

	"cf-ddns/internal/meta"
)

// newPinnedClient builds an HTTP client that only dials over the given network
// family ("tcp4" or "tcp6"). Pinning the address family makes the public IP
// reported by dual-stack services (e.g. myip.ipip.net) deterministic: a v4
// query always connects over IPv4 and a v6 query always over IPv6.
func newPinnedClient(network string) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			// Force the requested address family regardless of what the
			// caller passed (http always passes "tcp").
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: transport}
}

var (
	// clientV4 dials only over IPv4.
	clientV4 = newPinnedClient("tcp4")
	// clientV6 dials only over IPv6.
	clientV6 = newPinnedClient("tcp6")
)

func httpGet(api string, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", meta.UserAgent)
	return client.Do(req)
}

// ipipRegex matches the IP (IPv4 or IPv6) reported by myip.ipip.net.
var ipipRegex = regexp.MustCompile(`当前 IP：([0-9a-fA-F:.]+)`)

func getPublicIPByIPIPNET(client *http.Client) (string, error) {
	resp, err := httpGet("https://myip.ipip.net", client)
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
	ipMatch := ipipRegex.FindStringSubmatch(string(body))

	if len(ipMatch) < 2 {
		return "", errors.New("failed to parse public IP")
	}
	return ipMatch[1], nil
}

func getPublicIPv4ByIPIPNET() (string, error) {
	return getPublicIPByIPIPNET(clientV4)
}

func getPublicIPv6ByIPIPNET() (string, error) {
	return getPublicIPByIPIPNET(clientV6)
}

func getPublicIPByRaw(api string, client *http.Client) (string, error) {
	resp, err := httpGet(api, client)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, api)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", errors.New("empty response body")
	}

	return ip, nil
}

func getPublicIPv4ByGhink() (string, error) {
	return getPublicIPByRaw("https://v4-myip.gh.ink", clientV4)
}

func getPublicIPv6ByGhink() (string, error) {
	return getPublicIPByRaw("https://v6-myip.gh.ink", clientV6)
}
