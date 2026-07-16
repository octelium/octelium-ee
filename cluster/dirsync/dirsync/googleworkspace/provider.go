// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package googleworkspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/syncprovider"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

const maxResults = 500
const memberMaxResults = 200

type Provider struct {
	octeliumC octeliumc.ClientInterface
	dp        *enterprisev1.DirectoryProvider
	srv       *admin.Service
	customer  string
}

func NewProvider(ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	dp *enterprisev1.DirectoryProvider) (*Provider, error) {

	spec := dp.Spec.GetGoogleWorkspace()
	if spec == nil {
		return nil, errors.Errorf("DirectoryProvider is not of type GoogleWorkspace")
	}

	subject := spec.GetImpersonateSubject()
	if subject == "" {
		return nil, errors.Errorf("GoogleWorkspace requires impersonateSubject")
	}

	saJSON, err := syncprovider.GetSecretValue(ctx, octeliumC, spec.GetServiceAccount().GetFromSecret())
	if err != nil {
		return nil, err
	}

	cfg, err := google.JWTConfigFromJSON([]byte(saJSON),
		admin.AdminDirectoryUserReadonlyScope,
		admin.AdminDirectoryGroupReadonlyScope,
		admin.AdminDirectoryGroupMemberReadonlyScope,
	)
	if err != nil {
		return nil, err
	}
	cfg.Subject = subject

	srv, err := admin.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx)))
	if err != nil {
		return nil, err
	}

	customer := spec.GetCustomer()
	if customer == "" {
		customer = "octelium"
	}

	return &Provider{
		octeliumC: octeliumC,
		dp:        dp,
		srv:       srv,
		customer:  customer,
	}, nil
}

func (p *Provider) Synchronize(ctx context.Context) error {
	return syncprovider.NewReconciler(p.octeliumC, p.dp).Sync(ctx, p)
}

func (p *Provider) ListUsers(ctx context.Context) ([]*syncprovider.User, error) {
	var ret []*syncprovider.User
	pageToken := ""
	for {
		req := p.srv.Users.List().Context(ctx).Customer(p.customer).MaxResults(maxResults)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			return nil, err
		}
		for _, u := range resp.Users {
			if u == nil || u.Id == "" {
				continue
			}
			ret = append(ret, toUser(u))
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return ret, nil
}

func (p *Provider) ListGroups(ctx context.Context) ([]*syncprovider.Group, error) {
	var groups []*admin.Group
	pageToken := ""
	for {
		req := p.srv.Groups.List().Context(ctx).Customer(p.customer).MaxResults(maxResults)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			return nil, err
		}
		groups = append(groups, resp.Groups...)
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	var ret []*syncprovider.Group
	for _, g := range groups {
		if g == nil || g.Id == "" {
			continue
		}
		members, err := p.listGroupMembers(ctx, g.Id)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &syncprovider.Group{
			ExternalID:        g.Id,
			DisplayName:       groupDisplayName(g),
			MemberExternalIDs: members,
		})
	}
	return ret, nil
}

func (p *Provider) listGroupMembers(ctx context.Context, groupKey string) ([]string, error) {
	var ret []string
	pageToken := ""
	for {
		req := p.srv.Members.List(groupKey).Context(ctx).MaxResults(memberMaxResults)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			return nil, err
		}
		for _, m := range resp.Members {
			if m == nil || m.Id == "" {
				continue
			}
			if strings.EqualFold(m.Type, "USER") {
				ret = append(ret, m.Id)
			}
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return ret, nil
}

func toUser(u *admin.User) *syncprovider.User {
	ret := &syncprovider.User{
		ExternalID:  u.Id,
		Email:       u.PrimaryEmail,
		DisplayName: u.PrimaryEmail,
		IsDisabled:  u.Suspended,
		PicURL:      u.ThumbnailPhotoUrl,
	}

	if u.Name != nil {
		ret.FirstName = u.Name.GivenName
		ret.LastName = u.Name.FamilyName
		if fn := strings.TrimSpace(u.Name.FullName); fn != "" {
			ret.DisplayName = fn
		} else if dn := strings.TrimSpace(fmt.Sprintf("%s %s", u.Name.GivenName, u.Name.FamilyName)); dn != "" {
			ret.DisplayName = dn
		}
	}

	return ret
}

func groupDisplayName(g *admin.Group) string {
	if g.Name != "" {
		return g.Name
	}
	return g.Email
}
