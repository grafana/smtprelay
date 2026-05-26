package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupAllowedNetworks(t *testing.T) {
	t.Parallel()

	// obvious non-CIDR
	_, err := setupAllowedNetworks("bogus")
	require.Error(t, err)

	// can happen if you set -allowed_nets="" rather than -allowed_nets= in a
	// k8s pod spec
	_, err = setupAllowedNetworks(`""`)
	require.Error(t, err)

	// empty's totally OK
	nets, err := setupAllowedNetworks("")
	require.NoError(t, err)
	assert.Empty(t, nets)

	// single CIDR
	nets, err = setupAllowedNetworks("10.2.3.0/24")
	require.NoError(t, err)
	assert.Equal(t, []*net.IPNet{
		{IP: net.IP{10, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}},
	}, nets)

	// host IP isn't allowed
	_, err = setupAllowedNetworks("1.2.3.4/16")
	require.Error(t, err)
}

func FuzzSetupAllowedNetworks(f *testing.F) {
	f.Add("127.0.0.0/8 ::1/128")
	f.Add("10.0.0.0/8")
	f.Add("")
	f.Add("not-a-cidr")
	f.Add("192.168.1.0/24 10.0.0.0/8 172.16.0.0/12")
	f.Add("192.168.1.100/24")

	f.Fuzz(func(_ *testing.T, s string) {
		_, _ = setupAllowedNetworks(s)
	})
}

func FuzzParseLogHeaders(f *testing.F) {
	f.Add("field1=X-Header-1 field2=X-Header-2")
	f.Add("")
	f.Add("noequals")
	f.Add("=value")
	f.Add("key=")

	f.Fuzz(func(_ *testing.T, s string) {
		parseLogHeaders(s)
	})
}
