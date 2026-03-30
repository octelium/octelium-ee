// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watcher

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"go.uber.org/zap"
)

type Watcher struct {
	octeliumC octeliumc.ClientInterface
}

func InitWatcher(octeliumC octeliumc.ClientInterface) *Watcher {
	return &Watcher{
		octeliumC: octeliumC,
	}
}

func (w *Watcher) runCertificates(ctx context.Context) {
	zap.S().Debugf("starting Certificate watcher loop")
	time.Sleep(2 * time.Second)

	for {
		if err := w.doRunCertificates(ctx); err != nil {
			zap.S().Errorf("Could not run Certificate watcher: %+v", err)
		}
		time.Sleep(3 * time.Minute)
	}
}

func (w *Watcher) doRunCertificates(ctx context.Context) error {

	crtList, err := w.octeliumC.EnterpriseC().ListCertificate(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, crt := range crtList.Items {
		if err := w.doCheckAndIssueCrt(ctx, crt); err != nil {
			zap.L().Error("Could not issue Certificate", zap.Any("crt", crt), zap.Error(err))
		}
	}

	return nil
}

func (w *Watcher) doCheckAndIssueCrt(ctx context.Context, crt *enterprisev1.Certificate) error {
	issuance := crt.Status.Issuance

	if issuance == nil {
		return nil
	}

	doReissue := func() error {
		_, err := certutils.DoIssueCertificate(ctx, w.octeliumC, crt)
		return err
	}

	switch issuance.State {
	case enterprisev1.Certificate_Status_Issuance_FAILED:
		return doReissue()
	case enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED:
		return nil
	case enterprisev1.Certificate_Status_Issuance_ISSUING:
		if issuance.IssuanceStartedAt != nil && time.Now().After(issuance.IssuanceStartedAt.AsTime().Add(1*time.Hour)) {
			return doReissue()
		}

		return nil
	case enterprisev1.Certificate_Status_Issuance_SUCCESS:
		switch {
		case time.Now().After(crt.Status.Issuance.ExpiresAt.AsTime()),
			time.Now().Add(21 * 24 * time.Hour).After(crt.Status.Issuance.ExpiresAt.AsTime()):
			return doReissue()
		default:
			return nil
		}
	}

	return nil
}

func (w *Watcher) Run(ctx context.Context) {
	go w.runCertificates(ctx)
}
