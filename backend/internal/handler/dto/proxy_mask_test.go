package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskProxyValueKeepsEndsAndHidesMiddle(t *testing.T) {
	require.Equal(t, "185.1****7.205", MaskProxyValue("185.115.207.205"))
	require.Equal(t, "u****1", MaskProxyValue("user1"))
	require.Equal(t, "****", MaskProxyValue("abcd"))
	require.Equal(t, "", MaskProxyValue("  "))
}

func TestMaskedProxyDropsPasswordAndLeavesOriginal(t *testing.T) {
	original := &Proxy{ID: 3, Name: "hk-1", Host: "proxy.example.com", Port: 8080, Username: "alice-01", Password: "secret"}
	masked := MaskedProxy(original)

	require.Equal(t, "proxy****e.com", masked.Host)
	require.Equal(t, "al****01", masked.Username)
	require.Empty(t, masked.Password)
	require.Equal(t, "hk-1", masked.Name)
	require.Equal(t, 8080, masked.Port)
	require.Equal(t, "proxy.example.com", original.Host)
	require.Nil(t, MaskedProxy(nil))
}

func TestMaskedAdminProxyWithAccountCountHidesCopyableFields(t *testing.T) {
	in := &AdminProxyWithAccountCount{
		AdminProxy:     AdminProxy{Proxy: Proxy{Host: "10.20.30.40", Username: "bob-user"}, Password: "pw"},
		AccountCount:   2,
		IPAddress:      "203.0.113.77",
		LatencyMessage: "dial tcp 10.20.30.40:1080: timeout",
		Country:        "HK",
	}
	out := MaskedAdminProxyWithAccountCount(in)

	require.Equal(t, "10.****.40", out.Host)
	require.Empty(t, out.Password)
	require.Empty(t, out.Proxy.Password)
	require.Equal(t, "203.****3.77", out.IPAddress)
	require.Empty(t, out.LatencyMessage)
	require.Equal(t, "HK", out.Country)
	require.Equal(t, int64(2), out.AccountCount)
	require.Equal(t, "pw", in.Password)
}
