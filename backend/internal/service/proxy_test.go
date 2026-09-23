package service

import (
	"net/url"
	"strings"
	"testing"
)

func TestProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		proxy Proxy
		want  string
	}{
		{
			name: "without auth",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
			},
			want: "http://proxy.example.com:8080",
		},
		{
			name: "with auth",
			proxy: Proxy{
				Protocol: "socks5",
				Host:     "socks.example.com",
				Port:     1080,
				Username: "user",
				Password: "pass",
			},
			want: "socks5://user:pass@socks.example.com:1080",
		},
		{
			name: "username only keeps no auth for compatibility",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
				Username: "user-only",
			},
			want: "http://proxy.example.com:8080",
		},
		{
			name: "with special characters in credentials",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     3128,
				Username: "first last@corp",
				Password: "p@ ss:#word",
			},
			want: "http://first%20last%40corp:p%40%20ss%3A%23word@proxy.example.com:3128",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.proxy.URL(); got != tc.want {
				t.Fatalf("Proxy.URL() mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestProxyURL_SpecialCharactersRoundTrip(t *testing.T) {
	t.Parallel()

	proxy := Proxy{
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     3128,
		Username: "first last@corp",
		Password: "p@ ss:#word",
	}

	parsed, err := url.Parse(proxy.URL())
	if err != nil {
		t.Fatalf("parse proxy URL failed: %v", err)
	}
	if got := parsed.User.Username(); got != proxy.Username {
		t.Fatalf("username mismatch after parse: got=%q want=%q", got, proxy.Username)
	}
	pass, ok := parsed.User.Password()
	if !ok {
		t.Fatal("password missing after parse")
	}
	if pass != proxy.Password {
		t.Fatalf("password mismatch after parse: got=%q want=%q", pass, proxy.Password)
	}
}

// Scenario: 代理 URL 进日志前必须抹掉 userinfo。凭证泄在 INFO 日志里，
// 等于把账号密码写进磁盘和日志采集链路，和面板上的脱敏要求是同一条线。
func TestRedactProxyURLForLogStripsCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty stays empty", raw: "", want: ""},
		{
			name: "socks5h with credentials",
			raw:  "socks5h://fLubk6sZ:zSYfjFV9@38.96.16.7:42456",
			want: "socks5h://***@38.96.16.7:42456",
		},
		{
			name: "http without credentials keeps host",
			raw:  "http://proxy.example.com:8080",
			want: "http://proxy.example.com:8080",
		},
		{
			name: "username only still marked",
			raw:  "http://someuser@proxy.example.com:8080",
			want: "http://***@proxy.example.com:8080",
		},
		{
			name: "unparsable is redacted wholesale",
			raw:  "not a url at all",
			want: "[unparsable-proxy-url-redacted]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RedactProxyURLForLog(tt.raw); got != tt.want {
				t.Fatalf("RedactProxyURLForLog(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// Scenario: 带特殊字符的真实凭证（见 TestProxyURLRoundTripsSpecialCharacters）
// 经过 URL 转义后依然不能出现在日志串里。
func TestRedactProxyURLForLogHidesEscapedCredentials(t *testing.T) {
	t.Parallel()

	proxy := Proxy{
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     3128,
		Username: "first last@corp",
		Password: "p@ ss:#word",
	}

	got := RedactProxyURLForLog(proxy.URL())
	if got != "http://***@proxy.example.com:3128" {
		t.Fatalf("unexpected redaction: %q", got)
	}
	for _, secret := range []string{"first", "corp", "ss", "word"} {
		if strings.Contains(got, secret) {
			t.Fatalf("redacted URL %q still leaks %q", got, secret)
		}
	}
}
