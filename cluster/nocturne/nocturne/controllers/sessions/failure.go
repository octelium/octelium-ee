// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package sessioncontroller

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rpubsubv1"
	"github.com/octelium/octelium/cluster/common/authevents"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

const (
	failureGroup             = "octeliumee-nocturne"
	failureAckTimeoutSeconds = 60
	failureExportTimeout     = 10 * time.Second
	failureRetryBackoff      = 5 * time.Second
	failureUnimplementedWait = 1 * time.Minute
)

func (c *Controller) RunFailures(ctx context.Context) {
	go c.runFailures(ctx)
}

func (c *Controller) runFailures(ctx context.Context) {
	for {
		err := c.doRunFailures(ctx)
		if ctx.Err() != nil {
			return
		}

		wait := failureRetryBackoff
		if grpcerr.IsUnimplemented(err) {
			wait = failureUnimplementedWait
		}

		zap.L().Debug("The authentication failures subscription exited",
			zap.Error(err), zap.Duration("wait", wait))

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (c *Controller) doRunFailures(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := c.octeliumC.PubSubC().Subscribe(ctx, &rpubsubv1.SubscribeRequest{
		Topic:    []byte(authevents.TopicFailure),
		Group:    []byte(failureGroup),
		Position: rpubsubv1.SubscribeRequest_EARLIEST,
		AckTimeout: &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: failureAckTimeoutSeconds,
			},
		},
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		if len(resp.Messages) == 0 {
			continue
		}

		if err := c.handleFailures(ctx, resp.Messages); err != nil {
			zap.L().Warn("Could not handle the authentication failures", zap.Error(err))
		}
	}
}

func (c *Controller) handleFailures(ctx context.Context, msgs []*rpubsubv1.Message) error {
	var items []*enterprisev1.AuthenticationLog
	var ids [][]byte

	for _, msg := range msgs {
		ids = append(ids, msg.Id)

		failure, err := authevents.ParseFailure(msg)
		if err != nil {
			zap.L().Debug("Skipping an invalid authentication failure message",
				zap.ByteString("id", msg.Id), zap.Error(err))
			continue
		}

		items = append(items, getFailureAuthenticationLog(failure))
	}

	if len(items) > 0 {
		exportCtx, cancel := context.WithTimeout(ctx, failureExportTimeout)
		defer cancel()

		if err := c.exportLogs(exportCtx, items); err != nil {
			return err
		}
	}

	_, err := c.octeliumC.PubSubC().Acknowledge(ctx, &rpubsubv1.AcknowledgeRequest{
		Topic: []byte(authevents.TopicFailure),
		Group: []byte(failureGroup),
		Ids:   ids,
	})

	return err
}

func getFailureAuthenticationLog(failure *authevents.Failure) *enterprisev1.AuthenticationLog {
	createdAt := pbutils.Timestamp(failure.CreatedAt)

	return &enterprisev1.AuthenticationLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuthenticationLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: createdAt,
		},
		Entry: &enterprisev1.AuthenticationLog_Entry{
			SessionRef: failure.SessionRef,
			UserRef:    failure.UserRef,
			DeviceRef:  failure.DeviceRef,
			Authentication: &corev1.Session_Status_Authentication{
				Info:  failure.Info,
				SetAt: createdAt,
			},
			Status: enterprisev1.AuthenticationLog_Entry_FAILURE,
			Failure: &enterprisev1.AuthenticationLog_Entry_Failure{
				Reason: enterprisev1.AuthenticationLog_Entry_Failure_Reason(
					enterprisev1.AuthenticationLog_Entry_Failure_Reason_value[string(failure.Reason)]),
			},
		},
	}
}
