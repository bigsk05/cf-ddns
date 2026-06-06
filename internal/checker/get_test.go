package checker

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cf-ddns/internal/meta"
)

// TestHTTPGetSetsUserAgent verifies that requests carry the meta.UserAgent header.
func TestHTTPGetSetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := httpGet(srv.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("httpGet returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotUA != meta.UserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, meta.UserAgent)
	}
}

func TestGetPublicIPByRaw(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantIP  string
		wantErr bool
	}{
		{name: "plain ipv4", status: http.StatusOK, body: "1.1.1.1", wantIP: "1.1.1.1"},
		{name: "trailing newline trimmed", status: http.StatusOK, body: "1.2.3.4\n", wantIP: "1.2.3.4"},
		{name: "surrounding whitespace trimmed", status: http.StatusOK, body: "  2.2.2.2  ", wantIP: "2.2.2.2"},
		{name: "ipv6", status: http.StatusOK, body: "2606:4700:4700::1111\n", wantIP: "2606:4700:4700::1111"},
		{name: "empty body", status: http.StatusOK, body: "   \n", wantErr: true},
		{name: "non-200 status", status: http.StatusInternalServerError, body: "1.1.1.1", wantErr: true},
		{name: "not found", status: http.StatusNotFound, body: "<html>404</html>", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			ip, err := getPublicIPByRaw(srv.URL, http.DefaultClient)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got ip %q", ip)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ip != tc.wantIP {
				t.Errorf("ip = %q, want %q", ip, tc.wantIP)
			}
		})
	}
}

// TestGetPublicIPByRawConnectionError ensures transport errors surface as errors.
func TestGetPublicIPByRawConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately so the connection fails

	if _, err := getPublicIPByRaw(url, http.DefaultClient); err == nil {
		t.Error("expected error for closed server, got nil")
	}
}

// TestIPIPNetRegexParse verifies the ipip.net response parser handles both
// IPv4 and IPv6 addresses now that the endpoint is dual-stack.
func TestIPIPNetRegexParse(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "ipv4",
			body: "当前 IP：1.2.3.4  来自于：中国 广东 广州  电信",
			want: "1.2.3.4",
		},
		{
			name: "ipv6",
			body: "当前 IP：2606:4700:4700::1111  来自于：美国",
			want: "2606:4700:4700::1111",
		},
		{
			name: "ipv6 compressed",
			body: "当前 IP：240e::1  来自于：中国",
			want: "240e::1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := ipipRegex.FindStringSubmatch(tc.body)
			if len(m) < 2 {
				t.Fatalf("failed to parse IP from %q", tc.body)
			}
			if m[1] != tc.want {
				t.Errorf("parsed IP = %q, want %q", m[1], tc.want)
			}
		})
	}
}

// TestIPIPNetParseOverHTTP exercises the UA header and regex parse over a real
// HTTP roundtrip against a stub server.
func TestIPIPNetParseOverHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != meta.UserAgent {
			t.Errorf("User-Agent = %q, want %q", got, meta.UserAgent)
		}
		_, _ = w.Write([]byte("当前 IP：203.0.113.7  来自于：测试"))
	}))
	defer srv.Close()

	resp, err := httpGet(srv.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("httpGet error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	m := ipipRegex.FindStringSubmatch(string(body))
	if len(m) < 2 || m[1] != "203.0.113.7" {
		t.Errorf("parsed = %v, want 203.0.113.7", m)
	}
}
