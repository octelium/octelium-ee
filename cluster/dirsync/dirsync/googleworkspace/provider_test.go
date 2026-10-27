package googleworkspace

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func newGoogleWorkspaceHTTPTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	client, err := admin.NewService(t.Context(), option.WithEndpoint(srv.URL+"/"), option.WithHTTPClient(srv.Client()), option.WithoutAuthentication())
	require.NoError(t, err)
	return &Provider{srv: client, customer: "my_customer"}
}

func TestGoogleWorkspaceUsersPaginationAndConversion(t *testing.T) {
	p := newGoogleWorkspaceHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my_customer", r.URL.Query().Get("customer"))
		if r.URL.Query().Get("pageToken") == "" {
			io.WriteString(w, `{"users":[{"id":"b","primaryEmail":"b@example.com","suspended":true}],"nextPageToken":"next"}`)
		} else {
			io.WriteString(w, `{"users":[{"id":"a","primaryEmail":"a@example.com","name":{"givenName":"First","familyName":"Last"}}]}`)
		}
	})
	users, err := p.ListUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "a", users[0].ExternalID)
	assert.Equal(t, "First Last", users[0].DisplayName)
	assert.True(t, users[1].IsDisabled)
}

func TestGoogleWorkspaceRejectsInvalidPages(t *testing.T) {
	for _, body := range []string{`{"users":[{}]}`, `{"users":[{"id":"same"},{"id":"same"}]}`, `{"users":[],"nextPageToken":"same"}`} {
		p := newGoogleWorkspaceHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
		users, err := p.ListUsers(t.Context())
		assert.Nil(t, users)
		assert.Error(t, err)
	}
}

func TestGoogleWorkspaceGroupsAndDirectMembership(t *testing.T) {
	p := newGoogleWorkspaceHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/members") {
			io.WriteString(w, `{"members":[{"id":"z","type":"USER"},{"id":"nested","type":"GROUP"},{"id":"a","type":"user"}]}`)
		} else {
			io.WriteString(w, `{"groups":[{"id":"team","name":"Engineering","email":"team@example.com"}]}`)
		}
	})
	groups, err := p.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "Engineering", groups[0].DisplayName)
	assert.Equal(t, []string{"a", "z"}, groups[0].MemberExternalIDs)
}

func TestGoogleWorkspaceRejectsInvalidMembership(t *testing.T) {
	for _, body := range []string{`{"members":[{"type":"USER"}]}`, `{"members":[{"id":"same","type":"USER"},{"id":"same","type":"USER"}]}`, `{"members":[],"nextPageToken":"repeat"}`} {
		p := newGoogleWorkspaceHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
		members, err := p.listGroupMembers(t.Context(), "group")
		assert.Nil(t, members)
		assert.Error(t, err)
	}
}

func TestGoogleWorkspaceResponseBodyLimit(t *testing.T) {
	for _, tc := range []struct {
		size    int
		exceeds bool
	}{{0, false}, {4, false}, {5, true}} {
		reader := &responseLimitReadCloser{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", tc.size))), remaining: 4}
		data, err := io.ReadAll(reader)
		if tc.exceeds {
			assert.ErrorIs(t, err, errResponseTooLarge)
			_, err = reader.Read(make([]byte, 1))
			assert.ErrorIs(t, err, errResponseTooLarge)
		} else {
			assert.NoError(t, err)
			assert.Len(t, data, tc.size)
		}
		assert.NoError(t, reader.Close())
	}
}

func TestGoogleWorkspaceRetryClassificationAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		err   error
		retry bool
	}{
		{nil, false}, {context.Canceled, false}, {context.DeadlineExceeded, false}, {errResponseTooLarge, false},
		{&googleapi.Error{Code: 400}, false}, {&googleapi.Error{Code: 401}, false}, {&googleapi.Error{Code: 403}, false},
		{&googleapi.Error{Code: 429}, true}, {&googleapi.Error{Code: 503}, true},
		{&googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}}, true},
	} {
		assert.Equal(t, tc.retry, isRetryableGoogleError(tc.err))
	}
	assert.Equal(t, 2*time.Second, googleRetryAfter(&googleapi.Error{Header: http.Header{"Retry-After": []string{"2"}}}))
	assert.Zero(t, googleRetryAfter(errors.New("no header")))
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	err := (&Provider{}).doWithRetry(ctx, "list", func() error { calls++; cancel(); return &googleapi.Error{Code: 503} })
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestGoogleWorkspaceDisplayNameFallbacks(t *testing.T) {
	assert.Equal(t, "a@example.com", toUser(&admin.User{PrimaryEmail: "a@example.com"}).DisplayName)
	assert.Equal(t, "Full Name", toUser(&admin.User{Name: &admin.UserName{FullName: " Full Name "}}).DisplayName)
	assert.Equal(t, "First Last", toUser(&admin.User{Name: &admin.UserName{GivenName: "First", FamilyName: "Last"}}).DisplayName)
	assert.Equal(t, "team@example.com", groupDisplayName(&admin.Group{Email: "team@example.com"}))
}
