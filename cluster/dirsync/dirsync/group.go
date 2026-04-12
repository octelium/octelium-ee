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
	"io"
	"net/http"
	"strings"

	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func (s *server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	scimGroup, err := s.unmarshalGroupSCIMFromReqBody(r)
	if err != nil {
		s.setErrorBadRequestWithErr(w, err)
		return
	}

	zap.L().Debug("got create req", zap.Any("scimGroup", scimGroup), zap.Any("idp", reqCtx.DirectoryProvider))

	{
		_, err := s.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{
			Name: s.genGroupName(scimGroup, reqCtx.DirectoryProvider),
		})
		if err == nil {
			s.setErrorAlreadyExists(w, err)
			return
		}
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
		}
	}

	dpGroup := &enterprisev1.DirectoryProviderGroup{
		Metadata: &metav1.Metadata{
			Name: s.genGroupName(scimGroup, reqCtx.DirectoryProvider),
		},
		Spec: &enterprisev1.DirectoryProviderGroup_Spec{},
		Status: &enterprisev1.DirectoryProviderGroup_Status{
			DirectoryProviderRef: umetav1.GetObjectReference(reqCtx.DirectoryProvider),
		},
	}

	group := &corev1.Group{
		Metadata: &metav1.Metadata{
			Name: s.genGroupName(scimGroup, reqCtx.DirectoryProvider),
		},
		Spec:   &corev1.Group_Spec{},
		Status: &corev1.Group_Status{},
	}

	{
		groupList, err := s.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.attrs.displayName", scimGroup.DisplayName),
				urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", reqCtx.DirectoryProvider.Metadata.Uid),
			},
		})
		if err != nil {
			s.setErrorInternal(w, err)
			return
		}

		if len(groupList.Items) > 0 {
			s.setError(w, 409, "Group is already created")
			return
		}
	}

	group, err = s.getGroupFromSCIM(scimGroup, reqCtx.DirectoryProvider)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	if err := s.setGroup(ctx, scimGroup, dpGroup, group, reqCtx.DirectoryProvider); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	group, err = s.octeliumC.CoreC().CreateGroup(ctx, group)
	if err != nil {
		zap.L().Debug("Could not create Group", zap.Error(err))
		s.setErrorInternal(w, err)
		return
	}

	dpGroup.Status.GroupRef = umetav1.GetObjectReference(group)

	dpGroup, err = s.octeliumC.EnterpriseC().CreateDirectoryProviderGroup(ctx, dpGroup)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	for _, member := range scimGroup.Members {
		if err := s.addGroupToUser(ctx, member.Value, group); err != nil {
			zap.L().Warn("Could not addGroupToUser", zap.Error(err))
		}
	}

	res, err := s.toGroupSCIM(dpGroup)
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

func (s *server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	dpGroup, err := s.getDPGroupFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}

		s.setErrorInternal(w, err)
		return
	}

	scimGroup, err := s.unmarshalGroupSCIMFromReqBody(r)
	if err != nil {
		s.setErrorBadRequestWithErr(w, err)
		return
	}

	group, err := s.getGroupFromDPGroup(ctx, dpGroup)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}

		s.setErrorInternal(w, err)
		return
	}

	if err := s.setGroup(ctx, scimGroup, dpGroup, group, reqCtx.DirectoryProvider); err != nil {
		s.setErrorInternal(w, err)
		return
	}

	group, err = s.octeliumC.CoreC().UpdateGroup(ctx, group)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	dpGroup, err = s.octeliumC.EnterpriseC().UpdateDirectoryProviderGroup(ctx, dpGroup)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	for _, member := range scimGroup.Members {
		if err := s.addGroupToUser(ctx, member.Value, group); err != nil {
			zap.L().Warn("Could not addGroupToUser", zap.Error(err))
		}
	}

	res, err := s.toGroupSCIM(dpGroup)
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

func (s *server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()

	dpGroup, err := s.getDPGroupFromPath(r)
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = s.octeliumC.CoreC().DeleteGroup(ctx, &rmetav1.DeleteOptions{
		Uid: dpGroup.Status.GroupRef.Uid,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			s.setErrorInternal(w, err)
			return
		}
	}

	_, err = s.octeliumC.EnterpriseC().DeleteDirectoryProviderGroup(ctx, &rmetav1.DeleteOptions{
		Uid: dpGroup.Metadata.Uid,
	})
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) unmarshalGroupSCIMFromReqBody(r *http.Request) (*resourceGroup, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Errorf("Cannot read body")
	}
	ret := &resourceGroup{}

	if err := json.Unmarshal(b, ret); err != nil {

		return nil, err
	}

	return ret, nil
}

func (s *server) getGroupFromSCIM(i *resourceGroup, dp *enterprisev1.DirectoryProvider) (*corev1.Group, error) {
	ret := &corev1.Group{
		Metadata: &metav1.Metadata{
			Name:        s.genGroupName(i, dp),
			DisplayName: i.DisplayName,
		},
		Spec:   &corev1.Group_Spec{},
		Status: &corev1.Group_Status{},
	}

	return ret, nil

}

func (s *server) toGroupSCIM(i *enterprisev1.DirectoryProviderGroup) (*resourceGroup, error) {

	rscMap, err := pbutils.ConvertToMap(i.Status.Attrs)
	if err != nil {
		return nil, err
	}

	ret := &resourceGroup{}
	rscBytes, err := json.Marshal(rscMap)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rscBytes, ret); err != nil {
		return nil, err
	}

	ret.ID = i.Metadata.Uid
	ret.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}

	return ret, nil
}

func (s *server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	dpGroup, err := s.getDPGroupFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setErrorBadRequestWithErr(w, err)
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	res, err := s.toGroupSCIM(dpGroup)
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

func (s *server) handleListGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()
	filterArgs := r.URL.Query().Get("filter")
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	filters := []*rmetav1.ListOptions_Filter{
		urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", reqCtx.DirectoryProvider.Metadata.Uid),
	}

	zap.L().Debug("New listGroup req", zap.String("filter", filterArgs))

	if filterArgs != "" {
		switch {
		case strings.HasPrefix(filterArgs, "displayName eq "):
			arg := strings.TrimPrefix(filterArgs, "displayName eq ")
			arg = strings.TrimPrefix(arg, `"`)
			arg = strings.TrimSuffix(arg, `"`)

			filters = append(filters, urscsrv.FilterFieldEQValStr("status.attrs.displayName", arg))
		default:
			s.setErrorBadRequestWithErr(w, errors.Errorf("Unsupported filter"))
			return
		}
	}

	grpList, err := s.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{
		Filters:      filters,
		ItemsPerPage: getItemsPerPage(r),
		Paginate:     true,
		Page:         getPage(r),
	})
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	res := &responseGroupList{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		ItemsPerPage: int(getItemsPerPage(r)),
		StartIndex:   int(getPage(r)) + 1,
		TotalResults: len(grpList.Items),
		Resources:    []resourceGroup{},
	}

	for _, itm := range grpList.Items {
		scimGroup, err := s.toGroupSCIM(itm)
		if err != nil {
			s.setErrorInternal(w, err)
			return
		}
		res.Resources = append(res.Resources, *scimGroup)
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}

	w.Write(resBytes)
}

func (s *server) handlePatchGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	ctx := r.Context()

	dpGroup, err := s.getDPGroupFromPath(r)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setError(w, 404, "User not found")
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	group, err := s.getGroupFromDPGroup(ctx, dpGroup)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			s.setError(w, 404, "Group not found")
			return
		}
		s.setErrorInternal(w, err)
		return
	}

	scimGroup, err := s.toGroupSCIM(dpGroup)
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

	zap.L().Debug("New group patch request", zap.Any("req", req))

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
			case "displayName":
				val, ok := op.Value.(string)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("displayName must be a string"))
					return
				}
				scimGroup.DisplayName = val
			}
		case "add":
			if strings.ToLower(op.Path) == "members" {
				members, ok := op.Value.([]any)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("members must be a list of objects"))
					return
				}

				for _, mem := range members {
					v, ok := mem.(map[string]any)
					if !ok {
						continue
					}

					valueAny, ok := v["value"]
					if !ok {
						continue
					}

					value, ok := valueAny.(string)
					if !ok {
						continue
					}

					if err := s.addGroupToUser(ctx, value, group); err != nil {
						zap.L().Warn("Could not add Group to User",
							zap.Any("member", mem), zap.String("groupUID", group.Metadata.Uid), zap.Error(err))
					}
				}
			}
		case "remove":
			lowerPath := strings.ToLower(op.Path)
			if strings.HasPrefix(lowerPath, "members[value eq") {
				parts := strings.Split(op.Path, `"`)
				if len(parts) >= 3 {
					uid := parts[1]
					if err := s.removeGroupFromUser(ctx, uid, group); err != nil {
						zap.L().Warn("Could not remove Group from User via path filter",
							zap.String("memberUID", uid), zap.String("groupUID", group.Metadata.Uid), zap.Error(err))
					}
				} else {
					partsSingle := strings.Split(op.Path, "'")
					if len(partsSingle) >= 3 {
						uid := partsSingle[1]
						if err := s.removeGroupFromUser(ctx, uid, group); err != nil {
							zap.L().Warn("Could not remove Group from User via path filter",
								zap.String("memberUID", uid), zap.String("groupUID", group.Metadata.Uid), zap.Error(err))
						}
					}
				}
			} else if lowerPath == "members" {
				if op.Value == nil {
					continue
				}

				members, ok := op.Value.([]any)
				if !ok {
					s.setErrorBadRequestWithErr(w, errors.Errorf("members must be a list of objects"))
					return
				}

				for _, mem := range members {
					v, ok := mem.(map[string]any)
					if !ok {
						continue
					}

					valueAny, ok := v["value"]
					if !ok {
						continue
					}

					value, ok := valueAny.(string)
					if !ok {
						continue
					}

					if err := s.removeGroupFromUser(ctx, value, group); err != nil {
						zap.L().Warn("Could not remove Group from User via value payload",
							zap.Any("member", mem), zap.String("groupUID", group.Metadata.Uid), zap.Error(err))
					}
				}
			}
		default:
			s.setErrorBadRequestWithErr(w, errors.Errorf("Invalid patch operation type"))
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) addGroupToUser(ctx context.Context, userUID string, grp *corev1.Group) error {
	dpUsr, err := s.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{Uid: userUID})
	if err != nil {
		return err
	}

	usr, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	})
	if err != nil {
		return err
	}

	if ucorev1.ToUser(usr).HasGroupName(grp.Metadata.Name) {
		return nil
	}

	zap.L().Debug("Adding Group to User", zap.Any("group", grp), zap.Any("user", usr))

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  s.octeliumC,
		IsEmbedded: true,
	})

	usr.Spec.Groups = append(usr.Spec.Groups, grp.Metadata.Name)
	if err := coreSrv.CheckAndSetUser(ctx, s.octeliumC, usr, false); err != nil {
		return err
	}

	_, err = s.octeliumC.CoreC().UpdateUser(ctx, usr)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) removeGroupFromUser(ctx context.Context, userUID string, grp *corev1.Group) error {
	dpUsr, err := s.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{Uid: userUID})
	if err != nil {
		return err
	}

	usr, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	})
	if err != nil {
		return err
	}

	if !ucorev1.ToUser(usr).HasGroupName(grp.Metadata.Name) {
		return nil
	}

	usr.Spec.Groups = slices.DeleteFunc(usr.Spec.Groups, func(itm string) bool {
		return itm == grp.Metadata.Name
	})

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  s.octeliumC,
		IsEmbedded: true,
	})

	if err := coreSrv.CheckAndSetUser(ctx, s.octeliumC, usr, false); err != nil {
		return err
	}

	_, err = s.octeliumC.CoreC().UpdateUser(ctx, usr)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) getGroupFromDPGroup(ctx context.Context, dpu *enterprisev1.DirectoryProviderGroup) (*corev1.Group, error) {

	return s.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
		Uid: dpu.Status.GroupRef.Uid,
	})
}

func (s *server) setGroup(ctx context.Context,
	scimGroup *resourceGroup, dpGroup *enterprisev1.DirectoryProviderGroup,
	group *corev1.Group, dp *enterprisev1.DirectoryProvider) error {
	var err error

	scimGroupMap := make(map[string]any)
	scimGroupJSON, err := json.Marshal(scimGroup)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(scimGroupJSON, &scimGroupMap); err != nil {
		return err
	}

	scimUsrAttrs, err := pbutils.MapToStruct(scimGroupMap)
	if err != nil {
		return err
	}
	dpGroup.Status.Attrs = scimUsrAttrs

	nGroup, err := s.getGroupFromSCIM(scimGroup, dp)
	if err != nil {
		return err
	}

	group.Metadata.DisplayName = nGroup.Metadata.DisplayName

	return nil
}

func (s *server) getDPGroupFromPath(r *http.Request) (*enterprisev1.DirectoryProviderGroup, error) {

	ctx := r.Context()
	reqCtx := middlewares.GetCtxRequestContext(ctx)

	uid, err := s.getUID(r)
	if err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{
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
