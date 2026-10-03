package dto

import "strings"

// proxyMaskFill is the fixed run of asterisks that replaces the middle of a
// proxy endpoint value. A fixed width hides how many characters were removed.
const proxyMaskFill = "****"

// MaskProxyValue keeps roughly the first and last third of a proxy host,
// username or exit IP and replaces the middle with four asterisks. Viewers who
// may read the pool but not manage proxies can still tell proxies apart
// without being able to copy a working endpoint.
func MaskProxyValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= len(proxyMaskFill) {
		return proxyMaskFill
	}
	keep := len(runes) / 3
	if keep < 1 {
		keep = 1
	}
	return string(runes[:keep]) + proxyMaskFill + string(runes[len(runes)-keep:])
}

// MaskedProxy returns a copy of p with its endpoint identifiers masked and the
// password cleared. The original is left untouched.
func MaskedProxy(p *Proxy) *Proxy {
	if p == nil {
		return nil
	}
	out := *p
	out.Host = MaskProxyValue(p.Host)
	out.Username = MaskProxyValue(p.Username)
	out.Password = ""
	return &out
}

// MaskedAdminProxy masks an admin proxy DTO for a viewer without proxy write
// access: endpoint identifiers are masked and the password is dropped.
func MaskedAdminProxy(p *AdminProxy) *AdminProxy {
	if p == nil {
		return nil
	}
	out := *p
	out.Proxy = *MaskedProxy(&p.Proxy)
	out.Password = ""
	return &out
}

// MaskedAdminProxyWithAccountCount also masks the probed exit IP, which would
// otherwise reveal the endpoint of a direct proxy.
func MaskedAdminProxyWithAccountCount(p *AdminProxyWithAccountCount) *AdminProxyWithAccountCount {
	if p == nil {
		return nil
	}
	out := *p
	out.AdminProxy = *MaskedAdminProxy(&p.AdminProxy)
	out.IPAddress = MaskProxyValue(p.IPAddress)
	// Probe failures quote the dialed address verbatim.
	out.LatencyMessage = ""
	return &out
}
