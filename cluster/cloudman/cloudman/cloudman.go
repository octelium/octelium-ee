// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cloudman

import (
	"context"

	isscontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/certificateissuers"
	crtcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/certificates"
	cccontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/cluster_config"
	dnspcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/dnsproviders"
	gwcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/gateways"
	nscontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/namespaces"
	regioncontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/regions"
	svccontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/services"
	eewatchers "github.com/octelium/octelium-ee/cluster/common/watchers"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/common/watchers"
	"go.uber.org/zap"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/watcher"
)

func Run(ctx context.Context) error {
	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	/*
		if err := doRegisterACMEAccount(ctx, octeliumC); err != nil {
			zap.L().Error("Could not register initial ACME account", zap.Error(err))
		}
	*/

	/*
		if err := doInit(ctx, octeliumC); err != nil {
			return err
		}
	*/

	watcher.InitWatcher(octeliumC).Run(ctx)

	svcCtl := svccontroller.NewController(octeliumC)
	// secretCtl := secretcontroller.NewController(octeliumC)
	ccCtl := cccontroller.NewController(octeliumC)
	regionCtl := regioncontroller.NewController(octeliumC)
	gwCtl := gwcontroller.NewController(octeliumC)
	crtCtl := crtcontroller.NewController(octeliumC)
	issCtl := isscontroller.NewController(octeliumC)
	nsCtl := nscontroller.NewController(octeliumC)
	dnsPCtl := dnspcontroller.NewController(octeliumC)

	{
		watcher := watchers.NewCoreV1(octeliumC)
		if err := watcher.Service(ctx, nil, svcCtl.OnAdd, svcCtl.OnUpdate, svcCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Region(ctx, nil, regionCtl.OnAdd, regionCtl.OnUpdate, regionCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Gateway(ctx, nil, gwCtl.OnAdd, gwCtl.OnUpdate, gwCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Namespace(ctx, nil, nsCtl.OnAdd, nsCtl.OnUpdate, nsCtl.OnDelete); err != nil {
			return err
		}
	}

	{
		watcher := eewatchers.NewEnterpriseV1(octeliumC)
		if err := watcher.ClusterConfig(ctx, nil, ccCtl.OnUpdate); err != nil {
			return err
		}

		if err := watcher.DNSProvider(ctx, nil, dnsPCtl.OnAdd, dnsPCtl.OnUpdate, dnsPCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.CertificateIssuer(ctx, nil, issCtl.OnAdd, issCtl.OnUpdate, issCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Certificate(ctx, nil, crtCtl.OnAdd, crtCtl.OnUpdate, crtCtl.OnDelete); err != nil {
			return err
		}

	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.S().Debugf("Cloud Manager is running...")

	<-ctx.Done()

	return nil
}

/*
func doInit(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	_, err := octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	} else {
		return nil
	}

	zap.L().Debug("Creating default CertificateIssuer")

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	_, err = octeliumC.EnterpriseC().CreateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
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
		return err
	}

	zap.L().Debug("Successfully created default CertificateIssuer")

	return nil
}
*/

/*
func doInitCertificates(ctx context.Context, octeliumC octeliumc.ClientInterface, iss *enterprisev1.CertificateIssuer) error {

	nsList, err := octeliumC.CoreC().ListNamespace(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range nsList.Items {
		_, err := octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Name: fmt.Sprintf("ns-%s", ns.Metadata.Name),
		})
		if err == nil {
			continue
		}
		if !grpcerr.IsNotFound(err) {
			return err
		}

		if _, err := octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("ns-%s", ns.Metadata.Name),
			},
			Spec: &enterprisev1.Certificate_Spec{},
			Status: &enterprisev1.Certificate_Status{
				NamespaceRef:         umetav1.GetObjectReference(ns),
				CertificateIssuerRef: umetav1.GetObjectReference(iss),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}
*/
/*
func doRegisterACMEAccount(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	hasAccount, err := acmec.HasAccount(ctx, octeliumC)
	if err != nil {
		return err
	}

	if hasAccount {
		zap.S().Debugf("Already has an ACME account. No need to register a new one.")
		return nil
	}

	zap.S().Debugf("Registering initial ACME account")
	if err := cloudmanutils.RegisterACMEAccount(ctx, octeliumC); err != nil {
		return err
	}

	zap.L().Debug("Successfully registered ACME account")

	return nil
}
*/
