// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) startProcessAuditLogLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting process loop")

	for {
		select {
		case <-ctx.Done():
			return
		case lr := <-s.auditLogItem:
			s.processAuditLog(ctx, lr)
		}
	}
}

func (s *Server) processAuditLog(ctx context.Context, rsc umetav1.ResourceObjectI) error {

	if rsc == nil || rsc.GetMetadata() == nil || rsc.GetMetadata().ActorRef == nil {
		return nil
	}

	sess, err := s.getSessionFromActorRef(ctx, rsc.GetMetadata().ActorRef)
	if err != nil {
		zap.L().Warn("Could not getSessionFromActorRef")
	}

	if sess == nil || sess.Status == nil {
		sess = &corev1.Session{
			Status: &corev1.Session_Status{},
		}
	}

	/*
		if strings.HasPrefix(rsc.GetMetadata().ActorOperation, "octelium.api.main.user") {
			return nil
		}
	*/

	operation := rsc.GetMetadata().ActorOperation

	var pkg string
	var svc string
	var method string
	opArgs := strings.Split(operation, "/")
	if len(opArgs) > 0 {
		method = opArgs[1]
		fullSvc := opArgs[0]

		idx := strings.LastIndex(fullSvc, ".")
		if idx < len(fullSvc) {
			pkg = fullSvc[:idx]
			svc = fullSvc[idx+1:]
		}
	}

	lg := &enterprisev1.AuditLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuditLog,
		Metadata: &metav1.LogMetadata{
			Id: vutils.GenerateLogID(),
			CreatedAt: func() *timestamppb.Timestamp {
				if rsc.GetMetadata().UpdatedAt != nil {
					return rsc.GetMetadata().UpdatedAt
				}

				return rsc.GetMetadata().CreatedAt
			}(),
			ActorRef:  rsc.GetMetadata().ActorRef,
			TargetRef: umetav1.GetObjectReference(rsc),
		},

		Entry: &enterprisev1.AuditLog_Entry{
			ResourceRef: umetav1.GetObjectReference(rsc),
			SessionRef:  rsc.GetMetadata().ActorRef,
			Operation:   operation,
			UserRef:     sess.Status.UserRef,
			DeviceRef:   sess.Status.DeviceRef,
			Package:     pkg,
			Service:     svc,
			Method:      method,
		},
	}

	lgs := plog.NewLogs()
	lgs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	logRecords := lgs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	lr := logRecords.AppendEmpty()

	convertLogRecord(lg, lr)

	_, err = s.client.Export(ctx, plogotlp.NewExportRequestFromLogs(lgs))
	return err
}

func (s *Server) getSessionFromActorRef(ctx context.Context, actorRef *metav1.ObjectReference) (*corev1.Session, error) {

	ds := goqu.From("resources").Where(goqu.L(`uid`).Eq(actorRef.Uid)).
		Select(
			goqu.L(`rsc`),
		)
	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	ret := &corev1.Session{}

	rsc := make(map[string]any)

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&rsc); err != nil {
		return nil, err
	}

	if err := pbutils.UnmarshalFromMap(rsc, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func convertLogRecord(in *enterprisev1.AuditLog, ret plog.LogRecord) {
	inMap := pbutils.MustConvertToMap(in)

	ret.SetTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetObservedTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetSeverityNumber(plog.SeverityNumberInfo)
	ret.SetSeverityText(plog.SeverityNumberInfo.String())
	ret.Body().SetEmptyMap().FromRaw(inMap)
}
