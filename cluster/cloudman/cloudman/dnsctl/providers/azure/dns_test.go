package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSAddressFamiliesAndInvalidInput(t *testing.T) {
	v4, v6, err := splitIPAddresses([]string{"192.0.2.1", "2001:db8::1", "192.0.2.2"})
	require.NoError(t, err)
	assert.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, v4)
	assert.Equal(t, []string{"2001:db8::1"}, v6)
	v4, v6, err = splitIPAddresses(nil)
	assert.NoError(t, err)
	assert.Empty(t, v4)
	assert.Empty(t, v6)
	for _, invalid := range []string{"", "hostname.example.com", "192.0.2.1/24", "999.0.0.1", "[2001:db8::1]"} {
		_, _, err := splitIPAddresses([]string{"192.0.2.1", invalid})
		assert.Error(t, err, invalid)
	}
}

func TestDNSZoneBoundaries(t *testing.T) {
	for _, tc := range []struct {
		domain, zone string
		want         bool
	}{
		{"example.com", "example.com", true}, {"api.example.com", "example.com", true}, {"notexample.com", "example.com", false}, {"example.com.evil", "example.com", false}, {"other.com", "example.com", false},
	} {
		assert.Equal(t, tc.want, domainMatchesZone(tc.domain, tc.zone))
	}
	assert.Equal(t, "api.example.com", normalizeDomain(" API.Example.COM. "))
}

func TestAzureRecordNamesAndClouds(t *testing.T) {
	assert.Equal(t, "@", trimDomain("EXAMPLE.com.", "example.com"))
	assert.Equal(t, "API", trimDomain("API.EXAMPLE.com.", "example.com"))
	assert.Equal(t, "unrelated.com", trimDomain("unrelated.com", "example.com"))
	for _, name := range []string{"", "public", "china", "usgovernment"} {
		_, err := getCloudConfig(name)
		assert.NoError(t, err)
	}
	for _, name := range []string{"german", "unknown"} {
		_, err := getCloudConfig(name)
		assert.Error(t, err)
	}
}
