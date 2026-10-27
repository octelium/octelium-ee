package route53

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
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

func TestRoute53RecordNamesAndValues(t *testing.T) {
	assert.Equal(t, "Z123", trimHostedZoneID("/hostedzone/Z123"))
	assert.Equal(t, "Z123", trimHostedZoneID("Z123"))
	assert.Equal(t, "*.example.com.", normalizeRecordName(`\052.Example.COM.`))
	assert.Equal(t, []string{"one", "two"}, uniqueValues([]string{"", "one", "two", "one"}))
}

func TestRoute53ZoneSelectionAndRecordChanges(t *testing.T) {
	var mu sync.Mutex
	var changes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch {
		case strings.HasSuffix(r.URL.Path, "/hostedzone"):
			io.WriteString(w, `<ListHostedZonesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZones><HostedZone><Id>/hostedzone/PARENT</Id><Name>example.com.</Name><Config><PrivateZone>false</PrivateZone></Config></HostedZone><HostedZone><Id>/hostedzone/CHILD</Id><Name>sub.example.com.</Name><Config><PrivateZone>false</PrivateZone></Config></HostedZone><HostedZone><Id>/hostedzone/PRIVATE</Id><Name>app.sub.example.com.</Name><Config><PrivateZone>true</PrivateZone></Config></HostedZone></HostedZones><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListHostedZonesResponse>`)
		case r.Method == http.MethodPost:
			assert.Contains(t, r.URL.Path, "/hostedzone/CHILD/")
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			mu.Lock()
			changes = append(changes, string(body))
			mu.Unlock()
			io.WriteString(w, `<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeInfo><Id>change</Id><Status>INSYNC</Status><SubmittedAt>2026-01-01T00:00:00Z</SubmittedAt></ChangeInfo></ChangeResourceRecordSetsResponse>`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	client := awsroute53.NewFromConfig(aws.Config{
		Region:      "us-east-1",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("access", "secret", "")),
		HTTPClient:  srv.Client(),
	}, func(o *awsroute53.Options) { o.BaseEndpoint = aws.String(srv.URL); o.RetryMaxAttempts = 1 })
	p := &Provider{c: client, domain: "app.sub.example.com"}
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.1", "2001:db8::1"}))
	require.NoError(t, p.SetCNAME(t.Context(), "portal.sub.example.com", "app.sub.example.com"))
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, changes, 3)
	assert.Contains(t, changes[0], "UPSERT")
	assert.Contains(t, changes[0], "192.0.2.1")
	assert.Contains(t, changes[1], "AAAA")
	assert.Contains(t, changes[2], "CNAME")
}
