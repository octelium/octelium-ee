// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package nocturne

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/common/watchers"
	cccontroller "github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/cluster_config"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/devicemanagers"
	devcontroller "github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/devices"
	dpcontroller "github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/directoryproviders"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/requests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/reviews"
	sesscontroller "github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/sessions"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	ewatcher "github.com/octelium/octelium-ee/cluster/nocturne/nocturne/watcher"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/watcher/devwatcher"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	cwatchers "github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

func Run(ctx context.Context) error {

	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	k8sC, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	if err := doInit(ctx, octeliumC, k8sC); err != nil {
		return err
	}

	celEngine, err := celengine.New(ctx, nil)
	if err != nil {
		return err
	}

	watcher := watchers.NewEnterpriseV1(octeliumC)
	cWatcher := cwatchers.NewCoreV1(octeliumC)
	aWatcher := watchers.NewAccessV1(octeliumC)

	registry := devicemgrcommon.NewRegistry()

	devCtl := devcontroller.NewController(octeliumC, registry, &dmConditionEvaluator{
		octeliumC: octeliumC,
		celEngine: celEngine,
	})

	devWatcher, err := devwatcher.NewWatcher(&devwatcher.Opts{
		OcteliumC:  octeliumC,
		Resolver:   registry,
		Reconciler: devCtl,
		Locker:     devCtl,
	})
	if err != nil {
		return err
	}

	dmCtl, err := devicemanagers.NewController(ctx, octeliumC, registry, devWatcher, devCtl)
	if err != nil {
		return err
	}

	{
		ccCtl := cccontroller.NewController(octeliumC, k8sC)
		if err := watcher.ClusterConfig(ctx, nil, ccCtl.OnUpdate); err != nil {
			return err
		}
	}

	if err := cWatcher.Device(ctx, nil, devCtl.OnAdd, devCtl.OnUpdate, devCtl.OnDelete); err != nil {
		return err
	}

	if err := watcher.DeviceManager(ctx, nil, dmCtl.OnAdd, dmCtl.OnUpdate, dmCtl.OnDelete); err != nil {
		return err
	}

	devWatcher.Run(ctx)

	{
		dpCtl, err := dpcontroller.NewController(ctx, octeliumC)
		if err != nil {
			return err
		}
		if err := watcher.DirectoryProvider(ctx, nil,
			dpCtl.OnAdd, dpCtl.OnUpdate, dpCtl.OnDelete); err != nil {
			return err
		}
	}

	{
		ctl, err := requests.NewController(ctx, octeliumC)
		if err != nil {
			return err
		}
		if err := aWatcher.Request(ctx, nil,
			ctl.OnAdd, ctl.OnUpdate, ctl.OnDelete); err != nil {
			return err
		}
	}

	{
		ctl, err := reviews.NewController(ctx, octeliumC)
		if err != nil {
			return err
		}
		if err := aWatcher.Review(ctx, nil,
			ctl.OnAdd, ctl.OnUpdate, ctl.OnDelete); err != nil {
			return err
		}
	}

	{
		sessCtl, err := sesscontroller.NewController(ctx, octeliumC)
		if err != nil {
			return err
		}

		if err := cWatcher.Session(ctx, nil, sessCtl.OnAdd, sessCtl.OnUpdate, sessCtl.OnDelete); err != nil {
			return err
		}
	}

	ewatcher.InitWatcher(octeliumC).Run(ctx)

	zap.L().Info("Enterprise Nocturne is now running...")
	healthcheck.Run(vutils.HealthCheckPortMain)
	<-ctx.Done()

	return nil
}

type dmConditionEvaluator struct {
	octeliumC octeliumc.ClientInterface
	celEngine *celengine.CELEngine
}

var _ devcontroller.ConditionEvaluator = (*dmConditionEvaluator)(nil)

func (e *dmConditionEvaluator) MatchesDevice(
	ctx context.Context,
	condition *corev1.Condition,
	dev *corev1.Device,
) (bool, error) {
	if condition == nil {
		return true, nil
	}

	ctxMap := map[string]any{
		"device": pbutils.MustConvertToMap(dev),
	}

	if ref := dev.Status.UserRef; ref != nil && ref.GetUid() != "" {
		usr, err := e.octeliumC.CoreC().GetUser(ctx,
			apivalidation.ObjectReferenceToRGetOptions(ref))
		switch {
		case err == nil:
			ctxMap["user"] = pbutils.MustConvertToMap(usr)
		case !grpcerr.IsNotFound(err):
			return false, errors.Wrap(err, "Could not resolve Device User for Condition evaluation")
		}
	}

	return e.celEngine.EvalCondition(ctx, condition, map[string]any{
		"ctx": ctxMap,
	})
}

func doInit(ctx context.Context, octeliumC octeliumc.ClientInterface, k8sC kubernetes.Interface) error {
	if err := doSetOIDCSecret(ctx, octeliumC, k8sC); err != nil {
		zap.L().Warn("Could not set OIDC config and jwks secrets", zap.Error(err))
	}
	return nil
}

func doSetOIDCSecret(ctx context.Context, octeliumC octeliumc.ClientInterface, k8sC kubernetes.Interface) error {
	cs, ok := k8sC.(*kubernetes.Clientset)
	if !ok {
		return errors.Errorf("Could not get k8s client set from interface")
	}

	_, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: ovutils.GetOIDCConfigSecretName(vutils.GetMyRegionName()),
	})
	if err == nil {
		zap.L().Debug("OIDC config secret is already set. Nothing to be done...")
		return nil
	}
	if !grpcerr.IsNotFound(err) {
		return err
	}

	{

		oidcCfg, err := cs.RESTClient().Get().AbsPath("/.well-known/openid-configuration").DoRaw(ctx)
		if err != nil {
			return err
		}

		_, err = octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           ovutils.GetOIDCConfigSecretName(vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: string(oidcCfg),
				},
			},
		})
		if err != nil {
			return err
		}
	}

	{
		oidcJWKS, err := cs.RESTClient().Get().AbsPath("/openid/v1/jwks").DoRaw(ctx)
		if err != nil {
			return err
		}

		_, err = octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           ovutils.GetOIDCConfigSecretName(vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: string(oidcJWKS),
				},
			},
		})
		if err != nil {
			return err
		}
	}

	zap.L().Debug("OIDC config and jwks secrets successfully set...")

	return nil
}
