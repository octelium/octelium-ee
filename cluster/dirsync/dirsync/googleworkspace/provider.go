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

	"github.com/gosimple/slug"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	core "github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	admin "google.golang.org/api/admin/directory/v1"
)

type Opts struct {
	DirectorProviderRef *metav1.ObjectReference
}

type Provider struct {
	octeliumC           octeliumc.ClientInterface
	directorProviderRef *metav1.ObjectReference
	dp                  *enterprisev1.DirectoryProvider
	id                  string
	coreSrv             *core.Server
	srv                 *admin.Service
}

const maxResults = 500

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, o *Opts) (*Provider, error) {
	dp, err := octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: o.DirectorProviderRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	/*
		scopes := []string{
			admin.AdminDirectoryUserReadonlyScope,
			admin.AdminDirectoryGroupReadonlyScope,
		}

		srv, err := admin.NewService(ctx,
			option.WithScopes(scopes...),
		)
		if err != nil {
			return nil, err
		}
	*/

	if dp.Spec.GetGoogleWorkspace() == nil {
		return nil, errors.Errorf("")
	}

	return &Provider{
		octeliumC:           octeliumC,
		directorProviderRef: o.DirectorProviderRef,
		dp:                  dp,
		id:                  dp.Status.Id,
		coreSrv: core.NewServer(&core.Opts{
			OcteliumC:  octeliumC,
			IsEmbedded: true,
		}),
		// srv: srv,
	}, nil
}

func (p *Provider) Synchronize(ctx context.Context) error {

	provider, err := p.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: p.directorProviderRef.Uid,
	})
	if err != nil {
		return err
	}

	if provider.Spec.IsDisabled {
		return nil
	}

	{
		usrList, err := p.listUsers(ctx)
		if err != nil {
			return err
		}

		for _, u := range usrList {
			if err := p.setUser(ctx, u, provider); err != nil {
				zap.L().Warn("Could not setUser", zap.Any("usr", u), zap.Error(err))
			}
		}
	}

	return nil
}

func (p *Provider) setUser(ctx context.Context, u *admin.User, dp *enterprisev1.DirectoryProvider) error {
	if u == nil {
		return nil
	}

	doUpdate := func(usr *corev1.User) error {
		cUsr := p.toUser(u, dp)

		usr.Spec = cUsr.Spec

		usr, err := p.coreSrv.UpdateUser(ctx, usr)
		if err != nil {
			return err
		}

		return nil
	}

	if usr, err := p.getUser(ctx, u); err == nil {
		return doUpdate(usr)
	} else {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	if pUser, err := p.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
		Name: p.genUserName(u),
	}); err == nil {
		usr, err := p.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: pUser.Status.UserRef.Uid,
		})
		if err != nil {
			return err
		}

		return doUpdate(usr)
	} else if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	usr, err := p.coreSrv.CreateUser(ctx, p.toUser(u, dp))
	if err != nil {
		return err
	}

	if _, err := p.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, &enterprisev1.DirectoryProviderUser{
		Metadata: &metav1.Metadata{
			Name: p.genUserName(u),
		},
		Spec: &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			UserRef:              umetav1.GetObjectReference(usr),
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		},
	}); err != nil {
		return err
	}

	return nil
}

func (p *Provider) genUserName(u *admin.User) string {
	return fmt.Sprintf("%s-%s", p.id, slug.Make(u.Id))
}

func (p *Provider) genGroupName(u *admin.Group) string {
	return fmt.Sprintf("%s-%s", p.id, slug.Make(u.Id))
}

func (p *Provider) deleteUser(ctx context.Context, u *admin.User) error {

	dpUsr, err := p.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
		Name: p.genUserName(u),
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}

		return err
	}

	if _, err := p.octeliumC.EnterpriseC().DeleteDirectoryProviderUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	if _, err := p.octeliumC.CoreC().DeleteUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (p *Provider) getUser(ctx context.Context, u *admin.User) (*corev1.User, error) {
	usrList, err := p.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.email", u.PrimaryEmail),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(usrList.Items) != 1 {
		return nil, grpcutils.NotFound("")
	}

	return usrList.Items[0], nil
}

type resourceUser struct {
	Id           string `json:"id,omitempty"`
	PrimaryEmail string `json:"primaryEmail,omitempty"`
	Suspended    bool   `json:"suspended,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	ProfileURL   string `json:"profileUrl,omitempty"`
}

func (p *Provider) toUser(u *admin.User, dp *enterprisev1.DirectoryProvider) *corev1.User {
	return &corev1.User{
		Metadata: &metav1.Metadata{
			Name: p.genUserName(u),
		},
		Spec: &corev1.User_Spec{
			Type:       corev1.User_Spec_HUMAN,
			Email:      u.PrimaryEmail,
			IsDisabled: u.Suspended,
			Info: func() *corev1.User_Spec_Info {
				if u.Name == nil {
					return nil
				}
				return &corev1.User_Spec_Info{
					FirstName: u.Name.GivenName,
					LastName:  u.Name.FamilyName,
				}
			}(),
		},
	}
}

func (p *Provider) listUsers(ctx context.Context) ([]*admin.User, error) {
	var ret []*admin.User

	spec := p.dp.Spec.GetGoogleWorkspace()
	pageToken := ""
	for {
		usersReq := p.srv.Users.List().Context(ctx).Customer(spec.Customer).MaxResults(maxResults)
		if pageToken != "" {
			usersReq = usersReq.PageToken(pageToken)
		}
		usersResp, err := usersReq.Do()
		if err != nil {
			return nil, err
		}

		ret = append(ret, usersResp.Users...)

		pageToken = usersResp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return ret, nil
}

func (p *Provider) listGroup(ctx context.Context) ([]*admin.Group, error) {
	var ret []*admin.Group

	spec := p.dp.Spec.GetGoogleWorkspace()
	pageToken := ""
	for {
		req := p.srv.Groups.List().Context(ctx).Customer(spec.Customer).MaxResults(maxResults)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			return nil, err
		}

		ret = append(ret, resp.Groups...)

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return ret, nil
}

func (p *Provider) setGroup(ctx context.Context, g *admin.Group, dp *enterprisev1.DirectoryProvider) error {
	if g == nil {
		return nil
	}

	doUpdate := func(grp *corev1.Group) error {
		cGroup := p.toGroup(g, dp)

		grp.Spec = cGroup.Spec

		grp, err := p.coreSrv.UpdateGroup(ctx, grp)
		if err != nil {
			return err
		}

		return nil
	}

	if grp, err := p.getGroup(ctx, g); err == nil {
		return doUpdate(grp)
	} else {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	if pUser, err := p.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{
		Name: p.genGroupName(g),
	}); err == nil {
		usr, err := p.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
			Uid: pUser.Status.GroupRef.Uid,
		})
		if err != nil {
			return err
		}

		return doUpdate(usr)
	} else if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	usr, err := p.coreSrv.CreateGroup(ctx, p.toGroup(g, dp))
	if err != nil {
		return err
	}

	if _, err := p.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, &enterprisev1.DirectoryProviderUser{
		Metadata: &metav1.Metadata{
			Name: p.genGroupName(g),
		},
		Spec: &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			UserRef:              umetav1.GetObjectReference(usr),
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		},
	}); err != nil {
		return err
	}

	return nil
}

func (p *Provider) toGroup(u *admin.Group, dp *enterprisev1.DirectoryProvider) *corev1.Group {
	return &corev1.Group{
		Metadata: &metav1.Metadata{
			Name: p.genGroupName(u),
		},
		Spec: &corev1.Group_Spec{},
	}
}

func (p *Provider) getGroup(ctx context.Context, u *admin.Group) (*corev1.Group, error) {
	return p.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
		Name: p.genGroupName(u),
	})
}
