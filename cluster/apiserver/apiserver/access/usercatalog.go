// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"sort"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerUser) ListCatalog(ctx context.Context, req *accessv1.ListUserCatalogOptions) (*accessv1.CatalogList, error) {
	if _, err := userctx.GetUserCtx(ctx); err != nil {
		return nil, err
	}

	itemList, err := s.octeliumC.AccessC().ListCatalog(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerUser) ListCatalogService(ctx context.Context,
	req *accessv1.ListUserCatalogServiceOptions) (*userv1.ServiceList, error) {
	if _, err := userctx.GetUserCtx(ctx); err != nil {
		return nil, err
	}

	catalogList, err := s.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	svcMap := map[string]*corev1.Service{}

	for _, catalog := range catalogList.Items {
		if catalog.Spec.ResourceCollection == nil ||
			catalog.Spec.ResourceCollection.Service == nil {
			continue
		}

		if err := s.addCatalogServicesByName(ctx, svcMap, catalog.Spec.ResourceCollection.Service.Services); err != nil {
			return nil, err
		}

		if err := s.addCatalogServicesByNamespace(ctx, svcMap, catalog.Spec.ResourceCollection.Service.Namespaces); err != nil {
			return nil, err
		}
	}

	items := make([]*corev1.Service, 0, len(svcMap))
	for _, svc := range svcMap {
		items = append(items, svc)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Metadata.Name < items[j].Metadata.Name
	})

	ret := &userv1.ServiceList{
		ApiVersion: "user/v1",
		Kind:       "ServiceList",
		Items:      make([]*userv1.Service, 0, len(items)),
		ListResponseMeta: &metav1.ListResponseMeta{
			TotalCount: uint32(len(items)),
		},
	}

	for _, svc := range items {
		ret.Items = append(ret.Items, toUserCatalogService(svc))
	}

	return ret, nil
}

func (s *ServerUser) addCatalogServicesByName(ctx context.Context,
	svcMap map[string]*corev1.Service, services []string) error {
	for _, name := range services {
		if name == "" {
			continue
		}

		svc, err := s.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Name: vutils.GetServiceFullNameFromName(name),
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return serr.InternalWithErr(err)
		}

		if err := apivalidation.CheckIsUserHidden(svc); err != nil {
			continue
		}

		svcMap[svc.Metadata.Uid] = svc
	}

	return nil
}

func (s *ServerUser) addCatalogServicesByNamespace(ctx context.Context,
	svcMap map[string]*corev1.Service, namespaces []string) error {
	for _, nsName := range namespaces {
		if nsName == "" {
			continue
		}

		ns, err := s.octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
			Name: nsName,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return serr.InternalWithErr(err)
		}

		svcList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.namespaceRef.uid", ns.Metadata.Uid),
			},
		})
		if err != nil {
			return serr.InternalWithErr(err)
		}

		for _, svc := range svcList.Items {
			if err := apivalidation.CheckIsUserHidden(svc); err != nil {
				continue
			}

			svcMap[svc.Metadata.Uid] = svc
		}
	}

	return nil
}

func toUserCatalogService(svc *corev1.Service) *userv1.Service {
	return &userv1.Service{
		ApiVersion: "user/v1",
		Kind:       "Service",
		Metadata:   pbutils.Clone(svc.Metadata).(*metav1.Metadata),
		Spec: &userv1.Service_Spec{
			Type: userv1.Service_Spec_Type(svc.Spec.Mode),
			Port: uint32(ucorev1.ToService(svc).RealPort()),
		},
		Status: &userv1.Service_Status{
			Namespace: func() string {
				if svc.Status.NamespaceRef != nil {
					return svc.Status.NamespaceRef.Name
				}
				return ""
			}(),
		},
	}
}
