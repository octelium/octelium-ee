package cloudmanutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterDomainBoundaries(t *testing.T) {
	for _, tc := range []struct {
		domain string
		want   bool
	}{{"example.com", true}, {"api.example.com", true}, {"example.com.evil", false}, {"notexample.com", false}, {"", false}} {
		assert.Equal(t, tc.want, IsSameClusterDomain(tc.domain, "example.com"))
	}
}
