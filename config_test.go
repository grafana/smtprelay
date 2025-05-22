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
