// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"
	"fmt"
	"os"

	oc "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	iocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/cluster/genesis/genesis/genesisutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InitOpts struct {
	EnableSPIFFECSI         bool
	SPIFFECSIDriver         string
	SPIFFETrustDomain       string
	EnableIngressFrontProxy bool
}

func (g *Genesis) RunInit(ctx context.Context, o *InitOpts) error {
	zap.L().Info("Starting initializing the Cluster")

	octeliumCInit, err := iocteliumc.NewClient(ctx)
	if err != nil {
		return err
	}

	g.octeliumCInit = octeliumCInit

	/*
		if err := g.installClusterResources(ctx); err != nil {
			return err
		}
	*/

	clusterCfg, err := octeliumCInit.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	regionName := func() string {
		if os.Getenv("OCTELIUM_REGION_NAME") != "" {
			return os.Getenv("OCTELIUM_REGION_NAME")
		}
		return "default"
	}()

	rgn, err := octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: regionName,
	})
	if err != nil {
		return err
	}

	if err := g.setVolumes(ctx); err != nil {
		zap.L().Warn("Could not setVolumes", zap.Error(err))
		return err
	}

	if err := g.installComponents(ctx, &components.CommonOpts{
		CommonOpts: gc.CommonOpts{
			Region:                  rgn,
			EnableSPIFFECSI:         o.EnableSPIFFECSI,
			SPIFFECSIDriver:         o.SPIFFECSIDriver,
			SPIFFETrustDomain:       o.SPIFFETrustDomain,
			EnableIngressFrontProxy: o.EnableIngressFrontProxy,
		},
	}, true); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}
	g.octeliumC = octeliumC

	if err := g.installOcteliumResources(ctx, clusterCfg, rgn); err != nil {
		return err
	}

	zap.L().Info("Successfully initialized the Cluster")

	return nil
}

func (g *Genesis) installOcteliumResources(ctx context.Context, clusterCfg *corev1.ClusterConfig, rgn *corev1.Region) error {
	{
		if err := g.setInitKEK(ctx); err != nil {
			return err
		}

		if err := g.createDNSProvider(ctx); err != nil {
			return err
		}

		if err := g.createDefaultCertificateIssuer(ctx); err != nil {
			return err
		}
	}

	if rgn.Metadata.Name == "default" {
		{
			svc := &corev1.Service{
				Metadata: &metav1.Metadata{
					Name:         "enterprise.octelium-api",
					IsSystem:     true,
					IsUserHidden: true,

					SystemLabels: map[string]string{
						"octelium-apiserver": "true",
						"apiserver-path":     "/octelium.api.main.enterprise,/octelium.api.main.identity,/octelium.api.main.clusterman,/octelium.api.main.visibility",
					},
				},
				Spec: &corev1.Service_Spec{
					Port:     8080,
					IsPublic: true,
					Mode:     corev1.Service_Spec_GRPC,
				},
				Status: &corev1.Service_Status{
					ManagedService: &corev1.Service_Status_ManagedService{
						Type:  "apiserver",
						Image: oc.GetImage(oc.APIServer, ""),
						HealthCheck: &corev1.Service_Status_ManagedService_HealthCheck{
							Type: &corev1.Service_Status_ManagedService_HealthCheck_Grpc{
								Grpc: &corev1.Service_Status_ManagedService_HealthCheck_GRPC{
									Port: vutils.HealthCheckPortManagedService,
								},
							},
						},
					},
				},
			}

			if err := genesisutils.CreateOrUpdateService(ctx, g.octeliumC, svc); err != nil {
				return err
			}
		}

		{
			svc := &corev1.Service{
				Metadata: &metav1.Metadata{
					Name:         "public.octelium",
					IsSystem:     true,
					IsUserHidden: true,
				},
				Spec: &corev1.Service_Spec{
					Port:        8080,
					IsPublic:    true,
					Mode:        corev1.Service_Spec_HTTP,
					IsAnonymous: true,
				},
				Status: &corev1.Service_Status{
					ManagedService: &corev1.Service_Status_ManagedService{
						Image:              oc.GetImage(oc.PublicServer, ""),
						ReadOnlyFileSystem: true,
						HealthCheck: &corev1.Service_Status_ManagedService_HealthCheck{
							Type: &corev1.Service_Status_ManagedService_HealthCheck_Grpc{
								Grpc: &corev1.Service_Status_ManagedService_HealthCheck_GRPC{
									Port: vutils.HealthCheckPortManagedService,
								},
							},
						},
					},
				},
			}

			if err := genesisutils.CreateOrUpdateService(ctx, g.octeliumC, svc); err != nil {
				return err
			}
		}
	}

	if rgn.Metadata.Name == "default" {
		{
			svc := &corev1.Service{
				Metadata: &metav1.Metadata{
					Name:         "dirsync.octelium",
					IsSystem:     true,
					DisplayName:  "Octelium Directory Sync",
					Description:  "This Service implements a SCIM 2.0 server to sync Users and Groups",
					IsUserHidden: true,
				},
				Spec: &corev1.Service_Spec{
					Port:     8080,
					IsPublic: true,
					Mode:     corev1.Service_Spec_HTTP,
				},
				Status: &corev1.Service_Status{
					ManagedService: &corev1.Service_Status_ManagedService{
						Image:              oc.GetImage(oc.DirSync, ""),
						ReadOnlyFileSystem: true,
						HealthCheck: &corev1.Service_Status_ManagedService_HealthCheck{
							Type: &corev1.Service_Status_ManagedService_HealthCheck_Grpc{
								Grpc: &corev1.Service_Status_ManagedService_HealthCheck_GRPC{
									Port: vutils.HealthCheckPortManagedService,
								},
							},
						},
					},
				},
			}

			if err := genesisutils.CreateOrUpdateService(ctx, g.octeliumC, svc); err != nil {
				return err
			}
		}
		{
			svc := &corev1.Service{
				Metadata: &metav1.Metadata{
					Name:        "console.octelium",
					IsSystem:    true,
					DisplayName: "Octelium Console",
					Description: "Octelium Cluster's control dashboard",
				},
				Spec: &corev1.Service_Spec{
					Port:     8080,
					IsPublic: true,
					Mode:     corev1.Service_Spec_WEB,
				},
				Status: &corev1.Service_Status{
					ManagedService: &corev1.Service_Status_ManagedService{
						Image:              oc.GetImage(oc.Console, ""),
						ReadOnlyFileSystem: true,
						HealthCheck: &corev1.Service_Status_ManagedService_HealthCheck{
							Type: &corev1.Service_Status_ManagedService_HealthCheck_Grpc{
								Grpc: &corev1.Service_Status_ManagedService_HealthCheck_GRPC{
									Port: vutils.HealthCheckPortManagedService,
								},
							},
						},
					},
				},
			}

			if err := genesisutils.CreateOrUpdateService(ctx, g.octeliumC, svc); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Genesis) createDNSProvider(ctx context.Context) error {
	_, err := g.octeliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err == nil {
		zap.L().Debug("default DNSProvider is already created...")
		return nil
	}
	if !grpcerr.IsNotFound(err) {
		return err
	}

	if _, err := g.octeliumC.EnterpriseC().CreateDNSProvider(ctx, &enterprisev1.DNSProvider{
		Metadata: &metav1.Metadata{
			Name: "default",
			// IsSystem: true,
		},
		Spec:   &enterprisev1.DNSProvider_Spec{},
		Status: &enterprisev1.DNSProvider_Status{},
	}); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) createDefaultCertificateIssuer(ctx context.Context) error {

	zap.L().Debug("Creating default CertificateIssuer")

	cc, err := g.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	_, err = g.octeliumC.EnterpriseC().CreateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{
			Name: "default",
			// IsSystem: true,
		},
		Spec: &enterprisev1.CertificateIssuer_Spec{
			Type: &enterprisev1.CertificateIssuer_Spec_Acme{
				Acme: &enterprisev1.CertificateIssuer_Spec_ACME{
					Email: fmt.Sprintf("contact@%s", cc.Status.Domain),
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
	if err != nil {
		if grpcerr.AlreadyExists(err) {
			zap.L().Debug("default certificateIssuer is already created...")
			return nil
		}

		return err
	}

	zap.L().Debug("Successfully created default CertificateIssuer")

	return nil

}

func (g *Genesis) setVolumes(ctx context.Context) error {

	if err := g.createOrUpdatePVC(ctx, g.getPVC("octelium-logstore", 20000, "")); err != nil {
		return err
	}

	if err := g.createOrUpdatePVC(ctx, g.getPVC("octelium-rscstore", 10000, "")); err != nil {
		return err
	}

	if err := g.createOrUpdatePVC(ctx, g.getPVC("octelium-metricstore", 5000, "")); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) getPVC(name string, size int64, storageClass string) *k8scorev1.PersistentVolumeClaim {
	storageReq := getResourceQuantity(fmt.Sprintf("%dMi", int64((5000))))

	return &k8scorev1.PersistentVolumeClaim{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: vutils.K8sNS,
		},
		Spec: k8scorev1.PersistentVolumeClaimSpec{

			Resources: k8scorev1.VolumeResourceRequirements{
				Requests: k8scorev1.ResourceList{
					"storage": *storageReq,
				},
			},
			AccessModes: []k8scorev1.PersistentVolumeAccessMode{
				k8scorev1.ReadWriteOnce,
			},
			StorageClassName: func() *string {
				if storageClass == "" {
					return nil
				}
				return utils_types.StrToPtr(storageClass)
			}(),
		},
	}
}

func (g *Genesis) createOrUpdateVolume(ctx context.Context, itm *k8scorev1.PersistentVolume) error {

	if _, err := g.k8sC.CoreV1().PersistentVolumes().Create(ctx, itm, k8smetav1.CreateOptions{}); err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func (g *Genesis) createOrUpdatePVC(ctx context.Context, itm *k8scorev1.PersistentVolumeClaim) error {

	if _, err := g.k8sC.CoreV1().PersistentVolumeClaims(itm.Namespace).
		Create(ctx, itm, k8smetav1.CreateOptions{}); err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}
func getResourceQuantity(arg string) *resource.Quantity {
	ret := resource.MustParse(arg)
	return &ret
}
