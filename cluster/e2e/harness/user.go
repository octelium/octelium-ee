// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	octelium "github.com/octelium/octelium/octelium-go"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	accessUserService     = "octelium.api.main.access.v1.UserService"
	accessReviewerService = "octelium.api.main.access.v1.ReviewerService"
)

type Actor struct {
	User  *corev1.User
	Conn  *grpc.ClientConn
	Token string
}

func APIPolicy(name string, services ...string) []*corev1.InlinePolicy {
	quoted := make([]string, 0, len(services))
	for _, svc := range services {
		quoted = append(quoted, strconv.Quote(svc))
	}

	return []*corev1.InlinePolicy{
		{
			Name: name,
			Spec: &corev1.Policy_Spec{
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Name:     "allow",
						Effect:   corev1.Policy_Spec_Rule_ALLOW,
						Priority: -1,
						Condition: &corev1.Condition{
							Type: &corev1.Condition_Match{
								Match: fmt.Sprintf(
									`ctx.namespace.metadata.name == "octelium-api" && `+
										`ctx.request.grpc.serviceFullName in [%s]`,
									strings.Join(quoted, ", ")),
							},
						},
					},
				},
			},
		},
	}
}

func ServicePolicy(name string, svc *corev1.Service) []*corev1.InlinePolicy {
	return []*corev1.InlinePolicy{
		{
			Name: name,
			Spec: &corev1.Policy_Spec{
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Name:     "allow",
						Effect:   corev1.Policy_Spec_Rule_ALLOW,
						Priority: -1,
						Condition: &corev1.Condition{
							Type: &corev1.Condition_Match{
								Match: fmt.Sprintf(`ctx.service.metadata.uid == %q`,
									svc.Metadata.Uid),
							},
						},
					},
				},
			},
		},
	}
}

func (h *H) NewActor(t *testing.T, groups ...string) *Actor {
	t.Helper()

	return h.NewActorWithAuthorization(t, &corev1.User_Spec_Authorization{
		InlinePolicies: APIPolicy("access-api", accessUserService, accessReviewerService),
	}, groups...)
}

func (h *H) NewActorWithAuthorization(t *testing.T,
	authz *corev1.User_Spec_Authorization, groups ...string) *Actor {
	t.Helper()

	usr := h.CreateUser(t, &corev1.User{
		Spec: &corev1.User_Spec{
			Type:          corev1.User_Spec_WORKLOAD,
			Groups:        groups,
			Authorization: authz,
		},
	})

	return &Actor{
		User:  usr,
		Conn:  h.UserConn(t, usr),
		Token: h.AccessToken(t, usr),
	}
}

func (h *H) UserConn(t *testing.T, usr *corev1.User) *grpc.ClientConn {
	t.Helper()

	cred := h.CreateCredential(t, harness.CredentialOpts{
		User:        usr.Metadata.Name,
		Type:        corev1.Credential_Spec_AUTH_TOKEN,
		SessionType: corev1.Session_Status_CLIENTLESS,
	})

	tkn := h.CredentialToken(t, cred)
	if tkn.GetAuthenticationToken() == nil {
		t.Fatalf("The Credential %s did not yield an authentication token", cred.Metadata.Name)
	}

	oC, err := octelium.NewClient(t.Context(),
		octelium.WithDomain(h.Domain),
		octelium.WithAuthenticator(
			octelium.AuthenticationToken(tkn.GetAuthenticationToken().AuthenticationToken)))
	if err != nil {
		t.Fatalf("Could not build a Client for the User %s: %+v", usr.Metadata.Name, err)
	}

	conn, err := oC.Conn(t.Context())
	if err != nil {
		t.Fatalf("Could not dial the Cluster as the User %s: %+v", usr.Metadata.Name, err)
	}

	t.Cleanup(func() { conn.Close() })

	return conn
}

type Probe struct {
	h   *H
	c   *resty.Client
	svc *corev1.Service
	usr *corev1.User
}

func (h *H) Probe(t *testing.T, usr *corev1.User, svc *corev1.Service) *Probe {
	t.Helper()

	return &Probe{
		h:   h,
		c:   h.ServiceClient(svc, h.AccessToken(t, usr)),
		svc: svc,
		usr: usr,
	}
}

func (p *Probe) Status(ctx context.Context) (int, error) {
	return p.h.StatusOf(ctx, p.c, "/")
}

func (p *Probe) MustBeAllowed(t *testing.T) time.Duration {
	t.Helper()
	return p.h.WaitAllowed(t, p.c)
}

func (p *Probe) MustBeDenied(t *testing.T) time.Duration {
	t.Helper()
	return p.h.WaitDenied(t, p.c)
}

func (p *Probe) MustStayDenied(t *testing.T, window time.Duration) {
	t.Helper()

	p.h.Consistently(t, "the access to stay denied", window, func(ctx context.Context) error {
		got, err := p.Status(ctx)
		if err != nil {
			return nil
		}
		if got == http.StatusOK {
			return errors.Errorf("the request is still allowed")
		}
		return nil
	})
}
