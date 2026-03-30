// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/cluster/common/sessionc"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) CreateDirectoryProvider(ctx context.Context, req *enterprisev1.DirectoryProvider) (*enterprisev1.DirectoryProvider, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err == nil {
		return nil, serr.InvalidArg("The DirectoryProvider %s already exists", req.Metadata.Name)
	}

	if !grpcerr.IsNotFound(err) {
		return nil, serr.K8sInternal(err)
	}

	if err := s.validateDirectoryProvider(ctx, req); err != nil {
		return nil, err
	}

	item := &enterprisev1.DirectoryProvider{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status: &enterprisev1.DirectoryProvider_Status{
			Id: utilrand.GetRandomStringCanonical(6),
		},
	}

	item, err = s.octeliumC.EnterpriseC().CreateDirectoryProvider(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	usr, err := s.octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name:           fmt.Sprintf("sys:dp-%s", item.Status.Id),
			IsSystem:       true,
			IsUserHidden:   true,
			IsSystemHidden: true,
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_WORKLOAD,
			Session: &corev1.User_Spec_Session{
				MaxPerUser:   1,
				DefaultState: corev1.Session_Spec_ACTIVE,
				ClientlessDuration: &metav1.Duration{
					Type: &metav1.Duration_Months{
						Months: 6,
					},
				},
				AccessTokenDuration: &metav1.Duration{
					Type: &metav1.Duration_Months{
						Months: 6,
					},
				},
				RefreshTokenDuration: &metav1.Duration{
					Type: &metav1.Duration_Months{
						Months: 6,
					},
				},
			},
			Authorization: &corev1.User_Spec_Authorization{
				InlinePolicies: []*corev1.InlinePolicy{
					{
						Name: "dirsync-client",
						Spec: &corev1.Policy_Spec{
							Rules: []*corev1.Policy_Spec_Rule{
								{
									Effect: corev1.Policy_Spec_Rule_ALLOW,
									Condition: &corev1.Condition{
										Type: &corev1.Condition_Match{
											Match: `ctx.service.metadata.name == "dirsync.octelium"`,
										},
									},
									Priority: -1,
								},
								{
									Effect: corev1.Policy_Spec_Rule_DENY,
									Condition: &corev1.Condition{
										Type: &corev1.Condition_MatchAny{
											MatchAny: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Status: &corev1.User_Status{
			Ext: map[string]*structpb.Struct{
				ovutils.ExtInfoKeyEnterprise: pbutils.MessageToStructMust(&enterprisev1.UserExtInfo{
					DirectoryProviderRef: umetav1.GetObjectReference(item),
				}),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	item, err = s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	item.Status.UserRef = umetav1.GetObjectReference(usr)

	item, err = s.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, item)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) GetDirectoryProvider(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.DirectoryProvider, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListDirectoryProvider(ctx context.Context, req *enterprisev1.ListDirectoryProviderOptions) (*enterprisev1.DirectoryProviderList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListDirectoryProvider(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteDirectoryProvider(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckIsSystem(g); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteDirectoryProvider(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) UpdateDirectoryProvider(ctx context.Context, req *enterprisev1.DirectoryProvider) (*enterprisev1.DirectoryProvider, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if err := s.validateDirectoryProvider(ctx, req); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) GenerateDirectoryProviderCredential(ctx context.Context,
	req *enterprisev1.GenerateDirectoryProviderCredentialRequest) (*enterprisev1.GenerateDirectoryProviderCredentialResponse, error) {

	if err := apivalidation.CheckObjectRef(req.DirectoryProviderRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.DirectoryProviderRef))
	if err != nil {
		return nil, err
	}

	usr, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: item.Status.UserRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	var sess *corev1.Session

	if item.Status.SessionRef == nil {
		sess, err = sessionc.CreateSession(ctx, &sessionc.CreateSessionOpts{
			Usr:       usr,
			SessType:  corev1.Session_Status_CLIENTLESS,
			OcteliumC: s.octeliumC,
		})
		if err != nil {
			return nil, err
		}
	} else {
		sess, err = s.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
			Uid: item.Status.SessionRef.Uid,
		})
		if err != nil {
			if !grpcerr.IsNotFound(err) {
				return nil, err
			}

			sess, err = sessionc.CreateSession(ctx, &sessionc.CreateSessionOpts{
				Usr:       usr,
				SessType:  corev1.Session_Status_CLIENTLESS,
				OcteliumC: s.octeliumC,
				AuthenticationInfo: &corev1.Session_Status_Authentication_Info{
					Type: corev1.Session_Status_Authentication_Info_INTERNAL,
				},
			})
			if err != nil {
				return nil, err
			}
		} else {
			cc, err := s.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
			if err != nil {
				return nil, err
			}
			sessionc.SetCurrAuthentication(&sessionc.SetCurrAuthenticationOpts{
				Session:       sess,
				ClusterConfig: cc,
				AuthInfo: &corev1.Session_Status_Authentication_Info{
					Type: corev1.Session_Status_Authentication_Info_INTERNAL,
				},
			})

			sess, err = s.octeliumC.CoreC().UpdateSession(ctx, sess)
			if err != nil {
				return nil, grpcutils.InternalWithErr(err)
			}
		}
	}

	jwkCtl, err := jwkctl.NewJWKController(ctx, s.octeliumC)
	if err != nil {
		return nil, err
	}

	accessTkn, err := jwkCtl.CreateAccessToken(sess)
	if err != nil {
		return nil, err
	}

	if item.Status.SessionRef == nil || (item.Status.SessionRef.Uid != sess.Metadata.Uid) {
		item.Status.SessionRef = umetav1.GetObjectReference(sess)

		_, err = s.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, item)
		if err != nil {
			return nil, serr.K8sInternal(err)
		}

	}

	return &enterprisev1.GenerateDirectoryProviderCredentialResponse{
		Type: &enterprisev1.GenerateDirectoryProviderCredentialResponse_Bearer_{
			Bearer: &enterprisev1.GenerateDirectoryProviderCredentialResponse_Bearer{
				AccessToken: accessTkn,
			},
		},
	}, nil
}

func (s *Server) validateDirectoryProvider(ctx context.Context, req *enterprisev1.DirectoryProvider) error {

	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	switch req.Spec.Type.(type) {
	case *enterprisev1.DirectoryProvider_Spec_Scim:
	case *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_:
	default:
		return grpcutils.InvalidArg("Invalid DirectoryProvider type")
	}

	return nil
}

func (s *Server) ListDirectoryProviderUser(ctx context.Context, req *enterprisev1.ListDirectoryProviderUserOptions) (*enterprisev1.DirectoryProviderUserList, error) {

	var listOpts []*rmetav1.ListOptions_Filter
	if req.DirectoryProviderRef != nil {
		if err := apivalidation.CheckObjectRef(req.DirectoryProviderRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		dp, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
			Uid:  req.DirectoryProviderRef.Uid,
			Name: req.DirectoryProviderRef.Name,
		})
		if err != nil {
			return nil, err
		}
		listOpts = append(listOpts, urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", dp.Metadata.Uid))
	}
	itemList, err := s.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, urscsrv.GetPublicListOptions(req, listOpts...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) ListDirectoryProviderGroup(ctx context.Context, req *enterprisev1.ListDirectoryProviderGroupOptions) (*enterprisev1.DirectoryProviderGroupList, error) {

	var listOpts []*rmetav1.ListOptions_Filter
	if req.DirectoryProviderRef != nil {
		if err := apivalidation.CheckObjectRef(req.DirectoryProviderRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		dp, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.DirectoryProviderRef))
		if err != nil {
			return nil, err
		}
		listOpts = append(listOpts, urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", dp.Metadata.Uid))
	}

	itemList, err := s.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, urscsrv.GetPublicListOptions(req, listOpts...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) SynchronizeDirectoryProvider(ctx context.Context,
	req *enterprisev1.SynchronizeDirectoryProviderRequest) (*enterprisev1.SynchronizeDirectoryProviderResponse, error) {

	if err := apivalidation.CheckObjectRef(req.DirectoryProviderRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.DirectoryProviderRef))
	if err != nil {
		return nil, err
	}

	switch item.Spec.Type.(type) {
	case *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_:
	default:
		return nil, grpcutils.InvalidArg("This type does not supprot synchronizations")
	}

	item.Status.Synchronization = &enterprisev1.DirectoryProvider_Status_Synchronization{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
	}

	if _, err := s.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, item); err != nil {
		return nil, err
	}

	return &enterprisev1.SynchronizeDirectoryProviderResponse{}, nil
}
