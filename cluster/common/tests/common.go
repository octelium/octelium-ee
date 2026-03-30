// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package tests

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	ot "github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/cluster/rscserver/rscserver"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

type Opts struct {
}

type T struct {
	C     *FakeClient
	inner *ot.T
}

type FakeClient struct {
	K8sC      kubernetes.Interface
	OcteliumC octeliumc.ClientInterface
}

func (t *T) Destroy() error {
	return t.inner.Destroy()
}

func Initialize(o *Opts) (*T, error) {

	var rscs []umetav1.ResourceObjectI
	{
		clusterCfg := &enterprisev1.ClusterConfig{
			ApiVersion: "enterprise/v1",
			Kind:       "ClusterConfig",
			Metadata: &metav1.Metadata{
				Uid:             uuid.New().String(),
				ResourceVersion: uuid.New().String(),
				Name:            "default",
			},
			Spec:   &enterprisev1.ClusterConfig_Spec{},
			Status: &enterprisev1.ClusterConfig_Status{},
		}

		rscs = append(rscs, clusterCfg)

		rscs = append(rscs, &enterprisev1.CertificateIssuer{
			Kind:       "CertificateIssuer",
			ApiVersion: "enterprise/v1",
			Metadata: &metav1.Metadata{
				Name: "default",
				// IsSystem: true,
			},
			Spec: &enterprisev1.CertificateIssuer_Spec{
				Type: &enterprisev1.CertificateIssuer_Spec_Acme{
					Acme: &enterprisev1.CertificateIssuer_Spec_ACME{
						Email: fmt.Sprintf("contact@%s", "example.com"),
						Solver: &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
							Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{
								Dns: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_DNS{},
							},
						},
					},
				},
			},
			Status: &enterprisev1.CertificateIssuer_Status{},
		})
	}

	{
		rscs = append(rscs, &enterprisev1.SecretStore{
			Kind:       "SecretStore",
			ApiVersion: "enterprise/v1",
			Metadata: &metav1.Metadata{
				Name: "default",
				// IsSystem:    true,
				DisplayName: "The root SecretStore",
				Description: `
The initial SecretStore powered by the Kubernetes Cluster of the default Region
	`,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
					Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_KUBERNETES,
			},
		})

	}

	inner, err := ot.Initialize(&ot.Opts{
		PreCreatedResources: rscs,
		RscServerOpts: &rscserver.Opts{
			RegisterResourceFn: func(s grpc.ServiceRegistrar) error {
				rcorev1.RegisterResourceServiceServer(s, &struct {
					rcorev1.UnimplementedResourceServiceServer
				}{})
				renterprisev1.RegisterResourceServiceServer(s, &struct {
					renterprisev1.UnimplementedResourceServiceServer
				}{})

				return nil
			},

			NewResourceObject:     ovutils.NewResourceObject,
			NewResourceObjectList: ovutils.NewResourceObjectList,
		},
	})
	if err != nil {
		return nil, err
	}

	octeliumC, err := octeliumc.NewClient(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &T{
		inner: inner,
		C: &FakeClient{
			K8sC:      inner.C.K8sC,
			OcteliumC: octeliumC,
		},
	}, nil
}
