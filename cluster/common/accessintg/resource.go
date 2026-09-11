// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func ResolveResourceQuery(ctx context.Context, octeliumC octeliumc.ClientInterface,
	query string) (*accessv1.Request_Spec_Resource, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, grpcutils.InvalidArg("No resource was provided")
	}

	prefix, name, hasPrefix := strings.Cut(query, ":")
	if !hasPrefix {
		prefix = ""
		name = query
	}

	switch strings.ToLower(prefix) {
	case "svc", "service":
		return resolveServiceResource(ctx, octeliumC, name)

	case "cat", "catalog":
		return resolveCatalogResource(ctx, octeliumC, name)

	case "":
		ret, err := resolveServiceResource(ctx, octeliumC, name)
		if err == nil {
			return ret, nil
		}
		if !grpcerr.IsInvalidArg(err) {
			return nil, err
		}

		return resolveCatalogResource(ctx, octeliumC, name)

	default:
		return nil, grpcutils.InvalidArg(
			"The resource must be either `svc:NAME` or `cat:NAME`")
	}
}

func resolveServiceResource(ctx context.Context, octeliumC octeliumc.ClientInterface,
	name string) (*accessv1.Request_Spec_Resource, error) {
	svc, err := octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
		Name: vutils.GetServiceFullNameFromName(name),
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, grpcutils.InvalidArg("The Service %s does not exist", name)
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_ServiceRef{
			ServiceRef: umetav1.GetObjectReference(svc),
		},
	}, nil
}

func resolveCatalogResource(ctx context.Context, octeliumC octeliumc.ClientInterface,
	name string) (*accessv1.Request_Spec_Resource, error) {
	catalog, err := octeliumC.AccessC().GetCatalog(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, grpcutils.InvalidArg("No Service or Catalog is named %s", name)
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_Catalog_{
			Catalog: &accessv1.Request_Spec_Resource_Catalog{
				CatalogRef: umetav1.GetObjectReference(catalog),
			},
		},
	}, nil
}
