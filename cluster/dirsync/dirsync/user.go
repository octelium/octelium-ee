// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dirsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/gosimple/slug"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func (s *server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()

	reqCtx := middlewares.GetCtxRequestContext(ctx)

	scimUser, err := s.unmarshalUserSCIMFromReqBody(r)
	if err != nil {
		s.setErrorBadRequestWithErr(w, err)
		return
	}

	{
		_, err := s.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
			Name: s.genUserName(scimUser, reqCtx.DirectoryProvider),
		})
		if err == nil {
			s.setErrorAlreadyExists(w, err)
			return
		}
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
		}
	}

	zap.L().Debug("got req", zap.Any("scimUser", scimUser), zap.Any("idp", reqCtx.DirectoryProvider))

	dpUser := &enterprisev1.DirectoryProviderUser{
		Metadata: &metav1.Metadata{
			Name: s.genUserName(scimUser, reqCtx.DirectoryProvider),
		},
		Spec: &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			DirectoryProviderRef: umetav1.GetObjectReference(reqCtx.DirectoryProvider),
		},
	}

	usr := &corev1.User{
		Metadata: &metav1.Metadata{
			Name: s.genUserName(scimUser, reqCtx.DirectoryProvider),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	}

	if err := s.setUser(ctx, scimUser, dpUser, usr, reqCtx.DirectoryProvider); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	if usr.Spec.Email != "" {
		usrList, err := s.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("spec.email", usr.Spec.Email),
			},
		})
		if err != nil {
			s.setErrorInternal(w, err)
			return
		}

		coreSrv := admin.NewServer(&admin.Opts{
			OcteliumC:  s.octeliumC,
			IsEmbedded: true,
		})

		if len(usrList.Items) > 0 {
			oUsr := usrList.Items[0]
			oUsr.Spec = usr.Spec
			apisrvcommon.MetadataUpdate(oUsr.Metadata, usr.Metadata)
			if err := coreSrv.CheckAndSetUser(ctx, s.octeliumC, oUsr, false); err != nil {
				s.setErrorInternal(w, err)
				return
			}

			usr, err = s.octeliumC.CoreC().UpdateUser(ctx, oUsr)
			if err != nil {
				s.setErrorInternal(w, err)
				return
			}
		} else {

			if err := coreSrv.CheckAndSetUser(ctx, s.octeliumC, usr, false); err != nil {
				s.setErrorInternal(w, err)
				return
			}

			usr, err = s.octeliumC.CoreC().CreateUser(ctx, usr)
			if err != nil {
				s.setErrorInternal(w, err)
				return
			}
		}
	}

	dpUser.Status.UserRef = umetav1.GetObjectReference(usr)

	dpUser, err = s.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, dpUser)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	res, err := s.toUserSCIM(dpUser)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(resBytes)
}

func (s *server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)
	dpUsr, err := s.getDPUserFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}

		s.setErrorInternal(w, err)
		return
	}

	scimUser, err := s.unmarshalUserSCIMFromReqBody(r)
	if err != nil {
		s.setErrorBadRequestWithErr(w, err)
		return
	}

	usr, err := s.getUserFromDPUser(ctx, dpUsr)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}

		s.setErrorInternal(w, err)
		return
	}

	if err := s.setUser(ctx, scimUser, dpUsr, usr, reqCtx.DirectoryProvider); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	{
		coreSrv := admin.NewServer(&admin.Opts{
			OcteliumC:  s.octeliumC,
			IsEmbedded: true,
		})
		if coreSrv.CheckAndSetUser(ctx, s.octeliumC, usr, false); err != nil {
			s.setErrorBadRequestWithErr(w, err)
			return
		}
	}

	usr, err = s.octeliumC.CoreC().UpdateUser(ctx, usr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	dpUsr, err = s.octeliumC.EnterpriseC().UpdateDirectoryProviderUser(ctx, dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	res, err := s.toUserSCIM(dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.Write(resBytes)
}

func (s *server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()

	dpUsr, err := s.getDPUserFromPath(r)
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = s.octeliumC.CoreC().DeleteUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
			return
		}
	}

	_, err = s.octeliumC.EnterpriseC().DeleteDirectoryProviderUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Metadata.Uid,
	})
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) unmarshalUserSCIMFromReqBody(r *http.Request) (*resourceUser, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Errorf("Cannot read body")
	}
	scimUser := &resourceUser{}

	if err := json.Unmarshal(b, scimUser); err != nil {

		return nil, err
	}

	return scimUser, nil
}

func (s *server) getUserFromSCIM(i *resourceUser, dp *enterprisev1.DirectoryProvider) (*corev1.User, error) {
	ret := &corev1.User{
		Metadata: &metav1.Metadata{
			Name:        s.genUserName(i, dp),
			DisplayName: i.DisplayName,
			PicURL: func() string {
				if idx := slices.IndexFunc(i.Photos, func(itm *resourceUserPhoto) bool {
					return (itm.Value != "" && govalidator.IsURL(itm.Value)) && (itm.Primary || itm.Type == "photo")
				}); idx >= 0 {
					return i.Photos[idx].Value
				}

				return ""
			}(),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
			Info: &corev1.User_Spec_Info{
				Locale:    i.Locale,
				FirstName: i.Name.GivenName,
				LastName:  i.Name.FamilyName,
			},
			IsDisabled: !i.Active,
		},
		Status: &corev1.User_Status{},
	}

	for _, email := range i.Emails {
		if email.Primary && email.Value != "" {
			ret.Spec.Email = email.Value
			break
		}
	}

	return ret, nil

}

func (s *server) toUserSCIM(i *enterprisev1.DirectoryProviderUser) (*resourceUser, error) {

	rscMap, err := pbutils.ConvertToMap(i.Status.Attrs)
	if err != nil {
		return nil, err
	}

	ret := &resourceUser{}
	rscBytes, err := json.Marshal(rscMap)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rscBytes, ret); err != nil {
		return nil, err
	}

	ret.ID = i.Metadata.Uid
	ret.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}

	return ret, nil
}

func (s *server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	dpUsr, err := s.getDPUserFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	res, err := s.toUserSCIM(dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.Write(resBytes)

}

func (s *server) handleListUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	filterArgs := r.URL.Query().Get("filter")
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	zap.L().Debug("New listUser req", zap.String("filter", filterArgs))

	filters := []*rmetav1.ListOptions_Filter{
		urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", reqCtx.DirectoryProvider.Metadata.Uid),
	}

	if filterArgs != "" {
		switch {
		case strings.HasPrefix(filterArgs, "userName eq "):
			arg := strings.TrimPrefix(filterArgs, "userName eq ")
			arg = strings.TrimPrefix(arg, `"`)
			arg = strings.TrimSuffix(arg, `"`)

			filters = append(filters, urscsrv.FilterFieldEQValStr("status.attrs.userName", arg))
		default:
			s.setErrorBadRequestWithErr(w, errors.Errorf("Unsupported filter"))
			return
		}
	}

	dpUsrList, err := s.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{
		Filters:      filters,
		ItemsPerPage: getItemsPerPage(r),
		Paginate:     true,
		Page:         getPage(r),
	})
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	res := &responseUserList{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		ItemsPerPage: int(getItemsPerPage(r)),
		StartIndex:   int(getPage(r)) + 1,
		TotalResults: len(dpUsrList.Items),
		Resources:    []resourceUser{},
	}

	for _, itm := range dpUsrList.Items {
		scimUser, err := s.toUserSCIM(itm)
		if err != nil {
			s.setErrorInternal(w, err)
			return
		}
		res.Resources = append(res.Resources, *scimUser)
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.Write(resBytes)
}

func (s *server) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	dpUsr, err := s.getDPUserFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setError(w, 404, "User not found")
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	usr, err := s.getUserFromDPUser(ctx, dpUsr)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setError(w, 404, "User not found")
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	scimUsr, err := s.toUserSCIM(dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	req := &patchRequest{}
	if err := json.Unmarshal(b, req); err != nil {
		s.setErrorBadRequestWithErr(w, err)
		return
	}

	zap.L().Debug("New user patch req", zap.Any("req", req))

	if len(req.Operations) == 0 {
		s.setErrorBadRequestWithErr(w, errors.Errorf("No Operations found"))
		return
	}

	if len(req.Operations) > 1000 {
		s.setErrorBadRequestWithErr(w, errors.Errorf("Too many Operations found"))
		return
	}

	for _, op := range req.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			switch op.Path {
			case `emails[type eq "work"].value`:
				val, ok := op.Value.(string)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("Email must be a string"))
					return
				}

				for _, email := range scimUsr.Emails {
					if email.Type == "work" {
						email.Value = val
					}
				}

			case "userName":
				val, ok := op.Value.(string)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("userName must be a string"))
					return
				}
				scimUsr.UserName = val
			case "active":
				val, ok := op.Value.(bool)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("active must be a string"))
					return
				}

				scimUsr.Active = val
			}
		case "add":
		case "remove":
		default:
			s.setErrorBadRequestWithErr(w, errors.Errorf("Invalid patch operation type"))
			return
		}
	}

	if err := s.setUser(ctx, scimUsr, dpUsr, usr, reqCtx.DirectoryProvider); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  s.octeliumC,
		IsEmbedded: true,
	})
	if err := coreSrv.CheckAndSetUser(ctx, s.octeliumC, usr, false); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	usr, err = s.octeliumC.CoreC().UpdateUser(ctx, usr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	dpUsr, err = s.octeliumC.EnterpriseC().UpdateDirectoryProviderUser(ctx, dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	res, err := s.toUserSCIM(dpUsr)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.Write(resBytes)
}

func (s *server) getUserFromDPUser(ctx context.Context, dpu *enterprisev1.DirectoryProviderUser) (*corev1.User, error) {

	return s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: dpu.Status.UserRef.Uid,
	})
}

func getName(arg string) string {
	name := slug.Make(arg)

	return fmt.Sprintf("%s-%s", name, utilrand.GetRandomStringLowercase(4))
}

func (s *server) setUser(ctx context.Context,
	scimUser *resourceUser,
	dpUsr *enterprisev1.DirectoryProviderUser, usr *corev1.User,
	dp *enterprisev1.DirectoryProvider) error {
	var err error

	scimUsrMap := make(map[string]any)
	scimUsrJSON, err := json.Marshal(scimUser)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(scimUsrJSON, &scimUsrMap); err != nil {
		return err
	}

	scimUsrAttrs, err := pbutils.MapToStruct(scimUsrMap)
	if err != nil {
		return err
	}
	dpUsr.Status.Attrs = scimUsrAttrs

	nUsr, err := s.getUserFromSCIM(scimUser, dp)
	if err != nil {
		return err
	}

	usr.Metadata.DisplayName = nUsr.Metadata.DisplayName
	usr.Metadata.PicURL = nUsr.Metadata.PicURL
	usr.Spec.Authentication = nUsr.Spec.Authentication
	usr.Spec.Info = nUsr.Spec.Info
	usr.Spec.Email = nUsr.Spec.Email
	usr.Spec.IsDisabled = nUsr.Spec.IsDisabled

	return nil
}

func (s *server) getUID(r *http.Request) (string, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 1 {
		return "", errors.Errorf("Invalid URL path")
	}

	slices.Reverse(parts)
	uid := parts[0]
	if !govalidator.IsUUIDv4(uid) {
		return "", errors.Errorf("Invalid UID")
	}

	return uid, nil
}

func (s *server) getDPUserFromPath(r *http.Request) (*enterprisev1.DirectoryProviderUser, error) {

	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	uid, err := s.getUID(r)
	if err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
		Uid: uid,
	})
	if err != nil {
		return nil, err
	}

	if ret.Status.DirectoryProviderRef.Uid != reqCtx.DirectoryProvider.Metadata.Uid {
		return nil, grpcutils.NotFound("")
	}

	return ret, nil
}

func (s *server) genUserName(u *resourceUser, dp *enterprisev1.DirectoryProvider) string {
	return fmt.Sprintf("%s-%s", dp.Status.Id, slug.Make(u.UserName))
}

func (s *server) genGroupName(u *resourceGroup, dp *enterprisev1.DirectoryProvider) string {
	return fmt.Sprintf("%s-%s", dp.Status.Id, slug.Make(u.DisplayName))
}
