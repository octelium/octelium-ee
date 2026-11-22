// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/cluster"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const tstOrgUID = "6f1b2c4d-8e9f-4a0b-9c1d-2e3f4a5b6c7d"

type tstPortalRequest struct {
	method string
	path   string
	query  string
	header http.Header
	body   []byte
}

type tstPortal struct {
	t   *testing.T
	srv *httptest.Server

	mu             sync.Mutex
	exchanges      int
	tokens         map[string]bool
	tokenStatus    int
	describeStatus int
	entitled       bool
	requests       []tstPortalRequest
	tickets        map[string]map[string]any
	events         map[string][]map[string]any
	attachments    map[string][]byte
	order          []string
}

func newTstPortal(t *testing.T) *tstPortal {
	ret := &tstPortal{
		t:           t,
		tokens:      make(map[string]bool),
		entitled:    true,
		tickets:     make(map[string]map[string]any),
		events:      make(map[string][]map[string]any),
		attachments: make(map[string][]byte),
	}
	ret.srv = httptest.NewServer(http.HandlerFunc(ret.serve))
	t.Cleanup(ret.srv.Close)
	return ret
}

func (p *tstPortal) set(fn func(p *tstPortal)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(p)
}

func (p *tstPortal) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (p *tstPortal) writeError(w http.ResponseWriter, code int, errCode, msg string) {
	p.writeJSON(w, code, map[string]any{
		"error": map[string]any{
			"code":    errCode,
			"message": msg,
		},
	})
}

func (p *tstPortal) getRequests(path string) []tstPortalRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	var ret []tstPortalRequest
	for _, req := range p.requests {
		if req.path == path {
			ret = append(ret, req)
		}
	}
	return ret
}

func (p *tstPortal) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.requests = append(p.requests, tstPortalRequest{
		method: r.Method,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
		header: r.Header.Clone(),
		body:   body,
	})

	if r.URL.Path == pathSupportToken {
		if p.tokenStatus != 0 {
			p.writeError(w, p.tokenStatus, "unauthenticated", "the License has been revoked")
			return
		}
		p.exchanges++
		tkn := fmt.Sprintf("tkn-%d", p.exchanges)
		p.tokens[tkn] = true
		p.writeJSON(w, http.StatusCreated, map[string]any{
			"accessToken": tkn,
			"tokenType":   "Bearer",
			"expiresAt":   time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339Nano),
			"organization": map[string]any{
				"uid":         tstOrgUID,
				"displayName": "Acme Inc.",
			},
		})
		return
	}

	if !p.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")] {
		p.writeError(w, http.StatusUnauthorized, "unauthenticated", "the support access token is invalid or has expired")
		return
	}

	segments := strings.Split(strings.TrimPrefix(r.URL.Path, pathTickets), "/")

	switch {
	case r.URL.Path == pathSupport:
		if p.describeStatus != 0 {
			p.writeError(w, p.describeStatus, "not_found", "route not found")
			return
		}
		level := "STANDARD"
		if !p.entitled {
			level = "NONE"
		}
		p.writeJSON(w, http.StatusOK, map[string]any{
			"organization":   map[string]any{"uid": tstOrgUID, "displayName": "Acme Inc."},
			"clusterDomain":  "cluster.acme.example",
			"installationID": "3f3f3f3f3f3f3f3f",
			"supportLevel":   level,
			"entitled":       p.entitled,
			"attachments": map[string]any{
				"maxBytes":     25 << 20,
				"maxPerTicket": 100,
				"extensions":   []string{"txt", "log", "gz", "png"},
			},
			"maxBodyLength": 16000,
			"autoCloseDays": 5,
		})
	case r.URL.Path == pathTickets && r.Method == http.MethodGet:
		var items []map[string]any
		for _, id := range p.order {
			items = append(items, p.tickets[id])
		}
		p.writeJSON(w, http.StatusOK, map[string]any{"tickets": items, "cursor": "next-page"})
	case r.URL.Path == pathTickets && r.Method == http.MethodPost:
		in := map[string]any{}
		json.Unmarshal(body, &in)
		id := uuid.NewString()
		ticket := map[string]any{
			"id":                id,
			"number":            1001 + len(p.order),
			"organization":      map[string]any{"uid": tstOrgUID, "displayName": "Acme Inc."},
			"subject":           in["subject"],
			"category":          "TECHNICAL",
			"priority":          in["priority"],
			"requestedPriority": in["priority"],
			"status":            "NEW",
			"channel":           "CLUSTER",
			"requester":         map[string]any{"name": "Alice", "source": "cluster", "verified": false},
			"supportLevel":      "STANDARD",
			"cluster":           map[string]any{"domain": "cluster.acme.example"},
			"environment":       in["environment"],
			"sla": map[string]any{
				"coverage":           "BUSINESS",
				"startedAt":          "2026-11-16T14:00:00.000Z",
				"firstResponseDueAt": "2026-11-17T14:00:00.000Z",
				"paused":             false,
			},
			"escalated":       false,
			"createdAt":       "2026-11-16T14:00:00.000Z",
			"updatedAt":       "2026-11-16T14:00:00.000Z",
			"lastActivityAt":  "2026-11-16T14:00:00.000Z",
			"messageCount":    1,
			"attachmentCount": 0,
			"revision":        1,
		}
		p.tickets[id] = ticket
		p.order = append(p.order, id)
		p.events[id] = []map[string]any{{
			"id":       uuid.NewString(),
			"sequence": 1,
			"type":     "created",
			"actor":    map[string]any{"type": "cluster", "name": "Alice"},
			"channel":  "CLUSTER",
			"at":       "2026-11-16T14:00:00.000Z",
			"body":     in["description"],
		}}
		p.writeJSON(w, http.StatusCreated, map[string]any{"ticket": ticket, "replayed": false})
	case len(segments) >= 2:
		ticket, ok := p.tickets[segments[1]]
		if !ok {
			p.writeError(w, http.StatusNotFound, "not_found", "request not found")
			return
		}
		id := segments[1]
		action := ""
		if len(segments) >= 3 {
			action = segments[2]
		}

		switch {
		case action == "" && r.Method == http.MethodGet:
			attachments := []map[string]any{}
			for aid, data := range p.attachments {
				if strings.HasPrefix(aid, id+"/") {
					attachments = append(attachments, map[string]any{
						"id":          strings.TrimPrefix(aid, id+"/"),
						"name":        "logs.ndjson.gz",
						"contentType": "application/gzip",
						"size":        len(data),
						"purpose":     "DIAGNOSTIC",
						"uploadedBy":  map[string]any{"type": "cluster", "name": "Alice"},
						"channel":     "CLUSTER",
						"createdAt":   "2026-11-16T14:05:00.000Z",
						"available":   true,
					})
				}
			}
			open := ticket["status"] != "RESOLVED" && ticket["status"] != "CLOSED"
			p.writeJSON(w, http.StatusOK, map[string]any{
				"ticket":       ticket,
				"events":       p.events[id],
				"eventsCursor": "earlier",
				"attachments":  attachments,
				"actions": map[string]any{
					"reply":    ticket["status"] != "CLOSED",
					"resolve":  open,
					"escalate": open && ticket["escalated"] != true,
					"followUp": ticket["status"] == "CLOSED",
				},
			})
		case action == "events":
			p.writeJSON(w, http.StatusOK, map[string]any{
				"events": []map[string]any{{
					"id":       uuid.NewString(),
					"sequence": 0,
					"type":     "message",
					"actor":    map[string]any{"type": "agent", "name": "Sam Rivera"},
					"channel":  "AGENT",
					"at":       "2026-11-15T14:00:00.000Z",
					"body":     "An earlier message",
				}},
			})
		case action == "messages":
			in := map[string]any{}
			json.Unmarshal(body, &in)
			p.events[id] = append(p.events[id], map[string]any{
				"id":            uuid.NewString(),
				"sequence":      len(p.events[id]) + 1,
				"type":          "message",
				"actor":         map[string]any{"type": "cluster", "name": "Alice"},
				"channel":       "CLUSTER",
				"at":            "2026-11-16T15:00:00.000Z",
				"body":          in["body"],
				"attachmentIDs": in["attachmentIDs"],
			})
			p.writeJSON(w, http.StatusCreated, map[string]any{"ticket": ticket})
		case action == "attachments" && r.Method == http.MethodPost:
			aid := uuid.NewString()
			p.attachments[id+"/"+aid] = body
			p.writeJSON(w, http.StatusCreated, map[string]any{
				"attachment": map[string]any{
					"id":          aid,
					"name":        r.Header.Get("X-Filename"),
					"contentType": "application/gzip",
					"size":        len(body),
					"sha256":      r.Header.Get("X-Content-SHA256"),
					"purpose":     r.Header.Get("X-Attachment-Purpose"),
					"uploadedBy":  map[string]any{"type": "cluster"},
					"channel":     "CLUSTER",
					"createdAt":   "2026-11-16T14:05:00.000Z",
					"available":   true,
				},
				"ticket": ticket,
			})
		case action == "attachments" && r.Method == http.MethodGet && len(segments) == 4:
			data, ok := p.attachments[id+"/"+segments[3]]
			if !ok {
				p.writeError(w, http.StatusNotFound, "not_found", "attachment not found")
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="logs_x.ndjson.gz"; filename*=UTF-8''logs%20%C3%A9.ndjson.gz`)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		case action == "resolve":
			if ticket["status"] == "RESOLVED" {
				p.writeError(w, http.StatusConflict, "ticket_closed", "this request is already resolved or closed")
				return
			}
			ticket["status"] = "RESOLVED"
			p.writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket})
		case action == "escalate":
			ticket["escalated"] = true
			p.writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket})
		default:
			p.writeError(w, http.StatusNotFound, "not_found", "route not found")
		}
	default:
		p.writeError(w, http.StatusNotFound, "not_found", "route not found")
	}
}

type tstDownloadStream struct {
	grpc.ServerStream
	ctx      context.Context
	messages []*enterprisev1.DownloadSupportRequestAttachmentResponse
}

func (s *tstDownloadStream) Context() context.Context {
	return s.ctx
}

func (s *tstDownloadStream) Send(msg *enterprisev1.DownloadSupportRequestAttachmentResponse) error {
	s.messages = append(s.messages, msg)
	return nil
}

func tstSetLicense(t *testing.T, fn func(ctx context.Context, octeliumC octeliumc.ClientInterface) (*cluster.LicenseCredential, error)) {
	orig := getLicenseCredentialFunc
	getLicenseCredentialFunc = fn
	t.Cleanup(func() {
		getLicenseCredentialFunc = orig
	})
}

func tstValidLicense(ctx context.Context, octeliumC octeliumc.ClientInterface) (*cluster.LicenseCredential, error) {
	return &cluster.LicenseCredential{
		JWT: "eyJhbGciOiJFZERTQSJ9.license.signature",
		License: &enterprisev1.License{
			Uid:  "8b3d4e5f-0a1b-4c2d-9e3f-4a5b6c7d8e9f",
			Type: enterprisev1.License_SUBSCRIPTION,
			Plan: &enterprisev1.License_Plan{
				Type: enterprisev1.License_Plan_ENTERPRISE,
				Tier: "E2",
			},
		},
		ClusterDomain: "cluster.acme.example",
	}, nil
}

func tstUserCtx() context.Context {
	return context.WithValue(context.Background(), "octelium-user-ctx", &userctx.UserCtx{
		User: &corev1.User{
			Metadata: &metav1.Metadata{
				Uid:         "c0a8e1f2-0000-4000-8000-000000000001",
				Name:        "alice",
				DisplayName: "Alice",
			},
			Spec: &corev1.User_Spec{
				Email: "Alice@Acme.Example",
			},
		},
	})
}

func TestGetPortalURL(t *testing.T) {
	assert.Equal(t, devPortalURL, getPortalURL())

	for _, tc := range []struct {
		val      string
		expected string
	}{
		{val: "https://portal.octelium.com", expected: "https://portal.octelium.com"},
		{val: "https://portal.octelium.com/", expected: "https://portal.octelium.com"},
		{val: "http://localhost:8790", expected: "http://localhost:8790"},
		{val: "https://portal.octelium.com/api", expected: devPortalURL},
		{val: "https://user:pass@portal.octelium.com", expected: devPortalURL},
		{val: "https://portal.octelium.com?x=y", expected: devPortalURL},
		{val: "ftp://portal.octelium.com", expected: devPortalURL},
		{val: "portal.octelium.com", expected: devPortalURL},
	} {
		t.Setenv(envPortalURL, tc.val)
		assert.Equal(t, tc.expected, getPortalURL(), tc.val)
	}

	t.Setenv("OCTELIUM_DEV", "")
	t.Setenv("OCTELIUM_PRODUCTION", "true")
	t.Setenv(envPortalURL, "")
	assert.Equal(t, defaultPortalURL, getPortalURL())
	t.Setenv(envPortalURL, "http://localhost:8790")
	assert.Equal(t, defaultPortalURL, getPortalURL())
	t.Setenv(envPortalURL, "https://portal-test.octelium.com")
	assert.Equal(t, "https://portal-test.octelium.com", getPortalURL())
}

func TestServerDisabledInProduction(t *testing.T) {
	t.Setenv("OCTELIUM_DEV", "")
	t.Setenv("OCTELIUM_PRODUCTION", "true")

	srv := &Server{}
	ctx := context.Background()
	id := uuid.NewString()

	errs := []error{}
	_, err := srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
	errs = append(errs, err)
	_, err = srv.ListSupportRequest(ctx, &enterprisev1.ListSupportRequestOptions{})
	errs = append(errs, err)
	_, err = srv.GetSupportRequest(ctx, &enterprisev1.GetSupportRequestRequest{Id: id})
	errs = append(errs, err)
	_, err = srv.ListSupportRequestEvent(ctx, &enterprisev1.ListSupportRequestEventOptions{Id: id, Cursor: "x"})
	errs = append(errs, err)
	_, err = srv.CreateSupportRequest(ctx, &enterprisev1.CreateSupportRequestRequest{Subject: "Gateway", Description: "Down"})
	errs = append(errs, err)
	_, err = srv.AddSupportRequestMessage(ctx, &enterprisev1.AddSupportRequestMessageRequest{Id: id, Body: "Hello"})
	errs = append(errs, err)
	_, err = srv.AddSupportRequestAttachment(ctx, &enterprisev1.AddSupportRequestAttachmentRequest{Id: id, Name: "a.txt", Data: []byte("a")})
	errs = append(errs, err)
	_, err = srv.ResolveSupportRequest(ctx, &enterprisev1.ResolveSupportRequestRequest{Id: id})
	errs = append(errs, err)
	_, err = srv.EscalateSupportRequest(ctx, &enterprisev1.EscalateSupportRequestRequest{Id: id, Reason: "Production is down"})
	errs = append(errs, err)
	errs = append(errs, srv.DownloadSupportRequestAttachment(&enterprisev1.DownloadSupportRequestAttachmentRequest{Id: id, AttachmentID: id}, nil))

	for _, err := range errs {
		assert.True(t, grpcerr.IsUnimplemented(err), "%+v", err)
	}
}

func TestValidation(t *testing.T) {
	{
		_, err := getSubject("ab")
		assert.True(t, grpcerr.IsInvalidArg(err))
		_, err = getSubject("Gateway\ncrashes")
		assert.True(t, grpcerr.IsInvalidArg(err))
		_, err = getSubject(strings.Repeat("é", maxSubjectLen+1))
		assert.True(t, grpcerr.IsInvalidArg(err))
		ret, err := getSubject("  Gateway crashes\x00 ")
		assert.Nil(t, err)
		assert.Equal(t, "Gateway crashes", ret)
	}

	{
		_, err := getText("   ", "Message", 1, maxBodyLen)
		assert.True(t, grpcerr.IsInvalidArg(err))
		_, err = getText(string([]byte{0xff, 0xfe}), "Message", 1, maxBodyLen)
		assert.True(t, grpcerr.IsInvalidArg(err))
		_, err = getText("Too short", "Reason", minReasonLen, maxReasonLen)
		assert.True(t, grpcerr.IsInvalidArg(err))
		ret, err := getText("Line 1\r\nLine 2\x07", "Message", 1, maxBodyLen)
		assert.Nil(t, err)
		assert.Equal(t, "Line 1\nLine 2", ret)
	}

	{
		_, err := getIdempotencyKey("short")
		assert.True(t, grpcerr.IsInvalidArg(err))
		_, err = getIdempotencyKey("has spaces in it")
		assert.True(t, grpcerr.IsInvalidArg(err))
		ret, err := getIdempotencyKey("")
		assert.Nil(t, err)
		assert.Equal(t, "", ret)
		ret, err = getIdempotencyKey("support:0123456789")
		assert.Nil(t, err)
		assert.Equal(t, "support:0123456789", ret)
	}

	{
		assert.True(t, grpcerr.IsInvalidArg(checkID("not-an-id")))
		assert.True(t, grpcerr.IsInvalidArg(checkID("../../tickets")))
		assert.Nil(t, checkID(uuid.NewString()))
		assert.Nil(t, checkID(strings.ToUpper(uuid.NewString())))
	}

	for _, tc := range []struct {
		name     string
		expected string
	}{
		{name: "logs.txt", expected: "logs.txt"},
		{name: "../../etc/passwd.log", expected: "passwd.log"},
		{name: `C:\Users\alice\trace.HAR`, expected: "trace.HAR"},
		{name: "screen<shot>.png", expected: "screenshot.png"},
		{name: "component-logs.ndjson.gz", expected: "component-logs.ndjson.gz"},
	} {
		ret, err := getAttachmentName(tc.name)
		assert.Nil(t, err, tc.name)
		assert.Equal(t, tc.expected, ret)
	}

	for _, name := range []string{"", "run.exe", "noextension", ".", strings.Repeat("a", 300) + ".txt"} {
		_, err := getAttachmentName(name)
		assert.True(t, grpcerr.IsInvalidArg(err), name)
	}

	assert.Equal(t, "logs é.ndjson.gz", getDownloadName(`attachment; filename="logs_.ndjson.gz"; filename*=UTF-8''logs%20%C3%A9.ndjson.gz`))
	assert.Equal(t, "logs.txt", getDownloadName(`attachment; filename="logs.txt"`))
	assert.Equal(t, "attachment", getDownloadName(`attachment; filename="run.exe"`))
	assert.Equal(t, "attachment", getDownloadName(""))

	{
		assert.Nil(t, getReporter(nil))
		assert.Nil(t, getReporter(&userctx.UserCtx{User: &corev1.User{}}))

		ret := getReporter(&userctx.UserCtx{User: &corev1.User{
			Metadata: &metav1.Metadata{Name: "alice"},
			Spec:     &corev1.User_Spec{Email: " Alice@Acme.Example "},
		}})
		assert.Equal(t, map[string]string{"name": "alice", "email": "alice@acme.example"}, ret)

		ret = getReporter(&userctx.UserCtx{User: &corev1.User{
			Metadata: &metav1.Metadata{Name: "svc", DisplayName: "Deploy Bot"},
			Spec:     &corev1.User_Spec{Email: "Deploy Bot <bot@acme.example>"},
		}})
		assert.Equal(t, map[string]string{"name": "Deploy Bot"}, ret)
	}

	assert.Equal(t, "This request is closed", getMessage("this request is closed", "fallback"))
	assert.Equal(t, "fallback", getMessage(" ", "fallback"))
	assert.Equal(t, "fallback", getMessage(strings.Repeat("a", 600), "fallback"))
	assert.Equal(t, strings.Repeat("a", maxEnvironmentValueLen), truncateEnvironmentValue(strings.Repeat("a", 600)))
	assert.LessOrEqual(t, len(truncateEnvironmentValue(strings.Repeat("é", 600))), maxEnvironmentValueLen)
}

func TestToGRPCError(t *testing.T) {
	assert.Nil(t, toGRPCError(nil))

	orig := status.Error(codes.FailedPrecondition, "no license")
	assert.Equal(t, orig, toGRPCError(orig))

	for _, tc := range []struct {
		status int
		code   codes.Code
	}{
		{status: http.StatusBadRequest, code: codes.InvalidArgument},
		{status: http.StatusRequestEntityTooLarge, code: codes.InvalidArgument},
		{status: http.StatusUnauthorized, code: codes.FailedPrecondition},
		{status: http.StatusForbidden, code: codes.FailedPrecondition},
		{status: http.StatusNotFound, code: codes.NotFound},
		{status: http.StatusConflict, code: codes.FailedPrecondition},
		{status: http.StatusGone, code: codes.FailedPrecondition},
		{status: http.StatusTooManyRequests, code: codes.ResourceExhausted},
		{status: http.StatusInternalServerError, code: codes.Unavailable},
		{status: http.StatusBadGateway, code: codes.Unavailable},
	} {
		err := toGRPCError(&portalError{status: tc.status, message: "this request is closed"})
		assert.Equal(t, tc.code, status.Code(err), "%d", tc.status)
		assert.NotEqual(t, codes.Unauthenticated, status.Code(err))
	}

	assert.Equal(t, "This request is closed",
		status.Convert(toGRPCError(&portalError{status: http.StatusConflict, message: "this request is closed"})).Message())
	assert.Equal(t, codes.Canceled, status.Code(toGRPCError(context.Canceled)))
	assert.Equal(t, codes.Unavailable, status.Code(toGRPCError(context.DeadlineExceeded)))
	assert.Equal(t, codes.Unavailable, status.Code(toGRPCError(io.ErrUnexpectedEOF)))

	for _, tc := range []struct {
		err   error
		state enterprisev1.SupportInfo_State
	}{
		{err: &portalError{status: http.StatusUnauthorized}, state: enterprisev1.SupportInfo_UNAUTHORIZED},
		{err: &portalError{status: http.StatusForbidden, message: "the License has been revoked"}, state: enterprisev1.SupportInfo_UNAUTHORIZED},
		{err: &portalError{status: http.StatusNotFound}, state: enterprisev1.SupportInfo_DISABLED},
		{err: &portalError{status: http.StatusTooManyRequests}, state: enterprisev1.SupportInfo_UNREACHABLE},
		{err: &portalError{status: http.StatusServiceUnavailable}, state: enterprisev1.SupportInfo_UNREACHABLE},
		{err: context.DeadlineExceeded, state: enterprisev1.SupportInfo_UNREACHABLE},
	} {
		state, msg := getSupportInfoState(tc.err)
		assert.Equal(t, tc.state, state)
		assert.NotEmpty(t, msg)
	}

	_, msg := getSupportInfoState(&portalError{status: http.StatusForbidden, message: "the License has been revoked"})
	assert.Equal(t, "The License has been revoked", msg)
}

func TestPortalClientToken(t *testing.T) {
	portal := newTstPortal(t)
	c := newPortalClient(portal.srv.URL)
	cred := &credential{
		license:         "jwt",
		clusterDomain:   "cluster.acme.example",
		installationKey: strings.Repeat("k", 43),
	}
	ctx := context.Background()

	{
		var wg sync.WaitGroup
		for range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := c.getToken(ctx, cred)
				assert.Nil(t, err)
			}()
		}
		wg.Wait()
		assert.Equal(t, 1, portal.exchanges)

		reqs := portal.getRequests(pathSupportToken)
		require.Len(t, reqs, 1)
		in := map[string]string{}
		require.Nil(t, json.Unmarshal(reqs[0].body, &in))
		assert.Equal(t, map[string]string{
			"license":         "jwt",
			"clusterDomain":   "cluster.acme.example",
			"installationKey": strings.Repeat("k", 43),
		}, in)
		assert.Empty(t, reqs[0].header.Get("Authorization"))
	}

	{
		c.now = func() time.Time { return time.Now().Add(14 * time.Minute) }
		tkn, err := c.getToken(ctx, cred)
		assert.Nil(t, err)
		assert.Equal(t, "tkn-2", tkn.value)
		c.now = time.Now
	}

	{
		other := *cred
		other.license = "renewed-jwt"
		tkn, err := c.getToken(ctx, &other)
		assert.Nil(t, err)
		assert.Equal(t, "tkn-3", tkn.value)
	}

	{
		portal.set(func(p *tstPortal) { p.tokens = make(map[string]bool) })

		out := &portalDescription{}
		err := c.do(ctx, cred, &portalRequest{method: http.MethodGet, path: pathSupport}, out)
		assert.Nil(t, err)
		assert.Equal(t, "Acme Inc.", out.Organization.DisplayName)
		assert.Equal(t, 4, portal.exchanges)
	}

	{
		portal.set(func(p *tstPortal) {
			p.tokens = make(map[string]bool)
			p.tokenStatus = http.StatusForbidden
		})

		err := c.do(ctx, cred, &portalRequest{method: http.MethodGet, path: pathSupport}, &portalDescription{})
		perr := &portalError{}
		require.ErrorAs(t, err, &perr)
		assert.Equal(t, http.StatusForbidden, perr.status)
		assert.Equal(t, "the License has been revoked", perr.message)
	}
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	require.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	portal := newTstPortal(t)
	t.Setenv(envPortalURL, portal.srv.URL)
	tstSetLicense(t, tstValidLicense)

	srv := NewServer(tst.C.OcteliumC)
	usrCtx := tstUserCtx()

	{
		_, err := srv.GetSupportInfo(ctx, nil)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		res, err := srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{SkipPortal: true})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_STATE_UNKNOWN, res.State)
		assert.Equal(t, "cluster.acme.example", res.ClusterDomain)
		assert.Equal(t, portal.srv.URL, res.PortalURL)
		assert.Equal(t, "SUBSCRIPTION", res.Environment["license.type"])
		assert.Equal(t, "ENTERPRISE E2", res.Environment["license.plan"])
		assert.Contains(t, res.Environment, "regions")
		assert.Equal(t, 0, portal.exchanges)

		_, err = tst.C.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: installationSecretName})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		res, err := srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_AVAILABLE, res.State)
		assert.Equal(t, "Acme Inc.", res.Organization.DisplayName)
		assert.Equal(t, enterprisev1.SupportRequest_STANDARD, res.SupportLevel)
		assert.Equal(t, "3f3f3f3f3f3f3f3f", res.InstallationID)
		assert.Equal(t, uint64(maxAttachmentSize), res.Limits.MaxAttachmentSize)
		assert.Equal(t, []string{"txt", "log", "gz", "png"}, res.Limits.AttachmentExtensions)
		assert.Equal(t, uint32(5), res.Limits.AutoCloseDays)

		sec, err := tst.C.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: installationSecretName})
		require.Nil(t, err, "%+v", err)
		assert.True(t, sec.Metadata.IsSystem && sec.Metadata.IsUserHidden && sec.Metadata.IsSystemHidden)

		reqs := portal.getRequests(pathSupportToken)
		require.Len(t, reqs, 1)
		in := map[string]string{}
		require.Nil(t, json.Unmarshal(reqs[0].body, &in))
		assert.Equal(t, sec.Data.GetValue(), in["installationKey"])
		assert.Equal(t, "cluster.acme.example", in["clusterDomain"])

		other := NewServer(tst.C.OcteliumC)
		key, err := other.getInstallationKey(ctx)
		require.Nil(t, err)
		assert.Equal(t, sec.Data.GetValue(), key)

		_, err = srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err)
		assert.Len(t, portal.getRequests(pathSupport), 1)
	}

	{
		srv.info = nil
		portal.set(func(p *tstPortal) { p.entitled = false })
		res, err := srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_NOT_ENTITLED, res.State)
		assert.Equal(t, enterprisev1.SupportRequest_NONE, res.SupportLevel)
		portal.set(func(p *tstPortal) { p.entitled = true })

		srv.info = nil
		portal.set(func(p *tstPortal) { p.describeStatus = http.StatusNotFound })
		res, err = srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_DISABLED, res.State)
		portal.set(func(p *tstPortal) { p.describeStatus = 0 })

		srv.info = nil
		srv.portal.invalidateToken(srv.portal.token.key)
		portal.set(func(p *tstPortal) { p.tokenStatus = http.StatusForbidden })
		res, err = srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_UNAUTHORIZED, res.State)
		assert.Equal(t, "The License has been revoked", res.Message)
		portal.set(func(p *tstPortal) { p.tokenStatus = 0 })
		srv.info = nil
	}

	{
		unreachable := NewServer(tst.C.OcteliumC)
		unreachable.portal = newPortalClient("http://127.0.0.1:9")
		res, err := unreachable.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_UNREACHABLE, res.State)

		_, err = unreachable.ListSupportRequest(ctx, &enterprisev1.ListSupportRequestOptions{})
		assert.True(t, grpcerr.IsUnavailable(err), "%+v", err)
	}

	{
		_, err := srv.CreateSupportRequest(usrCtx, &enterprisev1.CreateSupportRequestRequest{
			Subject:     "No",
			Description: "The gateway restarts",
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		_, err = srv.CreateSupportRequest(usrCtx, &enterprisev1.CreateSupportRequestRequest{
			Subject:     "Gateway restarts",
			Description: "The gateway restarts",
			Priority:    enterprisev1.SupportRequest_Priority(42),
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	var ticket *enterprisev1.SupportRequest
	{
		res, err := srv.CreateSupportRequest(usrCtx, &enterprisev1.CreateSupportRequestRequest{
			Subject:        "Gateway restarts",
			Description:    "Every gateway restarts after the upgrade.",
			Priority:       enterprisev1.SupportRequest_P2,
			IdempotencyKey: "support-0123456789",
		})
		require.Nil(t, err, "%+v", err)
		ticket = res
		assert.Equal(t, uint32(1001), res.Number)
		assert.Equal(t, enterprisev1.SupportRequest_NEW, res.Status)
		assert.Equal(t, enterprisev1.SupportRequest_P2, res.Priority)
		assert.Equal(t, enterprisev1.SupportRequest_CLUSTER, res.Channel)
		assert.Equal(t, "cluster.acme.example", res.ClusterDomain)
		assert.False(t, res.Reporter.IsVerified)
		assert.Equal(t, "BUSINESS", res.Sla.Coverage)
		assert.NotNil(t, res.Sla.FirstResponseDueAt)
		assert.Equal(t, fmt.Sprintf("%s/organizations/%s/support/%s", portal.srv.URL, tstOrgUID, res.Id), res.PortalURL)
		require.Len(t, res.Events, 1)
		assert.Equal(t, enterprisev1.SupportRequest_Event_CREATED, res.Events[0].Type)
		assert.Equal(t, enterprisev1.SupportRequest_Actor_CLUSTER, res.Events[0].Actor.Type)
		assert.Equal(t, "earlier", res.EventsCursor)
		assert.True(t, res.Actions.CanReply && res.Actions.CanResolve && res.Actions.CanEscalate)

		reqs := portal.getRequests(pathTickets)
		var create tstPortalRequest
		for _, req := range reqs {
			if req.method == http.MethodPost {
				create = req
			}
		}
		assert.Equal(t, "support-0123456789", create.header.Get("Idempotency-Key"))
		in := map[string]any{}
		require.Nil(t, json.Unmarshal(create.body, &in))
		assert.Equal(t, "P2", in["priority"])
		assert.Equal(t, map[string]any{"name": "Alice", "email": "alice@acme.example"}, in["reporter"])
		assert.Equal(t, "8b3d4e5f-0a1b-4c2d-9e3f-4a5b6c7d8e9f", in["environment"].(map[string]any)["license.uid"])
		assert.NotContains(t, string(create.body), "eyJhbGciOiJFZERTQSJ9")
	}

	{
		res, err := srv.CreateSupportRequest(usrCtx, &enterprisev1.CreateSupportRequestRequest{
			Subject:     "Second request",
			Description: "Another issue",
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportRequest_P3, res.RequestedPriority)
	}

	{
		res, err := srv.ListSupportRequest(ctx, &enterprisev1.ListSupportRequestOptions{
			Filter: enterprisev1.ListSupportRequestOptions_OPEN,
			Cursor: "page-2",
		})
		require.Nil(t, err, "%+v", err)
		assert.Len(t, res.Items, 2)
		assert.Equal(t, "next-page", res.NextCursor)
		assert.Empty(t, res.Items[0].Events)

		reqs := portal.getRequests(pathTickets)
		assert.Equal(t, "cursor=page-2&status=open", reqs[len(reqs)-1].query)
	}

	{
		_, err := srv.GetSupportRequest(ctx, &enterprisev1.GetSupportRequestRequest{Id: "../support"})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		_, err = srv.GetSupportRequest(ctx, &enterprisev1.GetSupportRequestRequest{Id: uuid.NewString()})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

		res, err := srv.ListSupportRequestEvent(ctx, &enterprisev1.ListSupportRequestEventOptions{
			Id:     ticket.Id,
			Cursor: "earlier",
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, enterprisev1.SupportRequest_Actor_AGENT, res.Items[0].Actor.Type)
		assert.Equal(t, "Sam Rivera", res.Items[0].Actor.Name)
	}

	var attachment *enterprisev1.SupportRequest_Attachment
	data := bytes.Repeat([]byte("diagnostics "), downloadChunkSize/4)
	{
		_, err := srv.AddSupportRequestAttachment(usrCtx, &enterprisev1.AddSupportRequestAttachmentRequest{
			Id:   ticket.Id,
			Name: "run.exe",
			Data: []byte("x"),
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		_, err = srv.AddSupportRequestAttachment(usrCtx, &enterprisev1.AddSupportRequestAttachmentRequest{
			Id:   ticket.Id,
			Name: "large.log",
			Data: make([]byte, maxAttachmentSize+1),
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		res, err := srv.AddSupportRequestAttachment(usrCtx, &enterprisev1.AddSupportRequestAttachmentRequest{
			Id:      ticket.Id,
			Name:    "component logs é.ndjson.gz",
			Data:    data,
			Purpose: enterprisev1.SupportRequest_Attachment_DIAGNOSTIC,
		})
		require.Nil(t, err, "%+v", err)
		attachment = res
		assert.Equal(t, enterprisev1.SupportRequest_Attachment_DIAGNOSTIC, res.Purpose)
		assert.Equal(t, getSHA256(data), res.Sha256)
		assert.Equal(t, uint64(len(data)), res.Size)

		reqs := portal.getRequests(getTicketPath(ticket.Id, "attachments"))
		require.Len(t, reqs, 1)
		assert.Equal(t, "component%20logs%20%C3%A9.ndjson.gz", reqs[0].header.Get("X-Filename"))
		assert.Equal(t, "application/octet-stream", reqs[0].header.Get("Content-Type"))
		assert.Equal(t, fmt.Sprintf("%d", len(data)), reqs[0].header.Get("Content-Length"))
	}

	{
		_, err := srv.AddSupportRequestMessage(usrCtx, &enterprisev1.AddSupportRequestMessageRequest{
			Id:            ticket.Id,
			Body:          "Logs attached",
			AttachmentIDs: []string{"not-an-id"},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		res, err := srv.AddSupportRequestMessage(usrCtx, &enterprisev1.AddSupportRequestMessageRequest{
			Id:             ticket.Id,
			Body:           "Logs attached",
			AttachmentIDs:  []string{attachment.Id},
			IdempotencyKey: "support-message-0001",
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Events, 2)
		assert.Equal(t, "Logs attached", res.Events[1].Body)
		assert.Equal(t, []string{attachment.Id}, res.Events[1].AttachmentIDs)
		require.Len(t, res.Attachments, 1)
		assert.True(t, res.Attachments[0].IsAvailable)

		reqs := portal.getRequests(getTicketPath(ticket.Id, "messages"))
		require.Len(t, reqs, 1)
		assert.Equal(t, "support-message-0001", reqs[0].header.Get("Idempotency-Key"))
	}

	{
		stream := &tstDownloadStream{ctx: ctx}
		err := srv.DownloadSupportRequestAttachment(&enterprisev1.DownloadSupportRequestAttachmentRequest{
			Id:           ticket.Id,
			AttachmentID: attachment.Id,
		}, stream)
		require.Nil(t, err, "%+v", err)
		require.Greater(t, len(stream.messages), 2)

		header := stream.messages[0].GetHeader()
		require.NotNil(t, header)
		assert.Equal(t, "logs é.ndjson.gz", header.Name)
		assert.Equal(t, uint64(len(data)), header.Size)

		var got []byte
		for _, msg := range stream.messages[1:] {
			assert.LessOrEqual(t, len(msg.GetChunk()), downloadChunkSize)
			got = append(got, msg.GetChunk()...)
		}
		assert.Equal(t, data, got)

		err = srv.DownloadSupportRequestAttachment(&enterprisev1.DownloadSupportRequestAttachmentRequest{
			Id:           ticket.Id,
			AttachmentID: uuid.NewString(),
		}, &tstDownloadStream{ctx: ctx})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.EscalateSupportRequest(ctx, &enterprisev1.EscalateSupportRequestRequest{
			Id:     ticket.Id,
			Reason: "Urgent",
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		res, err := srv.EscalateSupportRequest(ctx, &enterprisev1.EscalateSupportRequestRequest{
			Id:     ticket.Id,
			Reason: "Production traffic is down for every user.",
		})
		require.Nil(t, err, "%+v", err)
		assert.True(t, res.IsEscalated)
		assert.False(t, res.Actions.CanEscalate)
	}

	{
		res, err := srv.ResolveSupportRequest(ctx, &enterprisev1.ResolveSupportRequestRequest{Id: ticket.Id})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportRequest_RESOLVED, res.Status)
		assert.False(t, res.Actions.CanResolve)

		_, err = srv.ResolveSupportRequest(ctx, &enterprisev1.ResolveSupportRequestRequest{Id: ticket.Id})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
		assert.Equal(t, "This request is already resolved or closed", status.Convert(err).Message())
	}

	{
		tstSetLicense(t, func(ctx context.Context, octeliumC octeliumc.ClientInterface) (*cluster.LicenseCredential, error) {
			return nil, &cluster.LicenseUnavailableError{Reason: "The License of the Cluster has expired"}
		})

		res, err := srv.GetSupportInfo(ctx, &enterprisev1.GetSupportInfoRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SupportInfo_NO_LICENSE, res.State)
		assert.Equal(t, "The License of the Cluster has expired", res.Message)

		_, err = srv.ListSupportRequest(ctx, &enterprisev1.ListSupportRequestOptions{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

		_, err = srv.CreateSupportRequest(usrCtx, &enterprisev1.CreateSupportRequestRequest{
			Subject:     "Gateway restarts",
			Description: "Every gateway restarts",
		})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
	}
}

func TestInstallationSecretOwnership(t *testing.T) {
	_, err := getInstallationKeyFromSecret(nil)
	assert.NotNil(t, err)

	sec := &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name:           installationSecretName,
			IsSystem:       true,
			IsUserHidden:   true,
			IsSystemHidden: true,
			SystemLabels: map[string]string{
				installationSecretLabel: "true",
			},
		},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: strings.Repeat("k", 43),
			},
		},
	}

	key, err := getInstallationKeyFromSecret(sec)
	assert.Nil(t, err)
	assert.Equal(t, strings.Repeat("k", 43), key)

	sec.Metadata.IsUserHidden = false
	_, err = getInstallationKeyFromSecret(sec)
	assert.NotNil(t, err)
	sec.Metadata.IsUserHidden = true

	sec.Metadata.SystemLabels = nil
	_, err = getInstallationKeyFromSecret(sec)
	assert.NotNil(t, err)
	sec.Metadata.SystemLabels = map[string]string{installationSecretLabel: "true"}

	sec.Data.Type = &enterprisev1.Secret_Data_Value{Value: "short"}
	_, err = getInstallationKeyFromSecret(sec)
	assert.NotNil(t, err)
}
