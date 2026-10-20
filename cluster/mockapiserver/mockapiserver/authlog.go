// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const mockAuthLogStep = 10 * time.Minute

var mockAuthFailureReasons = []enterprisev1.AuthenticationLog_Entry_Failure_Reason{
	enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL,
	enterprisev1.AuthenticationLog_Entry_Failure_EXPIRED_CREDENTIAL,
	enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL,
	enterprisev1.AuthenticationLog_Entry_Failure_USER_NOT_FOUND,
	enterprisev1.AuthenticationLog_Entry_Failure_REVOKED_CREDENTIAL,
	enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL,
	enterprisev1.AuthenticationLog_Entry_Failure_LIMIT_EXCEEDED,
}

func getMockAuthLogWindow(from, to *timestamppb.Timestamp, defaultDuration time.Duration) (time.Time, time.Time) {
	end := time.Now()
	if to.IsValid() {
		end = to.AsTime()
	}

	start := end.Add(-defaultDuration)
	if from.IsValid() {
		start = from.AsTime()
	}

	return start, end
}

func (s *tstVisibilityLog) getAuthenticationLogs(ctx context.Context, from, to time.Time) ([]*enterprisev1.AuthenticationLog, error) {
	sessList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(sessList.Items) == 0 {
		return nil, nil
	}

	var ret []*enterprisev1.AuthenticationLog

	end := to.Truncate(mockAuthLogStep)
	for i := 0; i < 5000; i++ {
		createdAt := end.Add(-time.Duration(i) * mockAuthLogStep)
		if createdAt.Before(from) {
			break
		}

		slot := createdAt.Unix() / int64(mockAuthLogStep.Seconds())
		sess := sessList.Items[slot%int64(len(sessList.Items))]
		isFailure := slot%5 == 2 || slot%13 == 0

		lg := &enterprisev1.AuthenticationLog{
			ApiVersion: uenterprisev1.APIVersion,
			Kind:       uenterprisev1.KindAuthenticationLog,
			Metadata: &metav1.LogMetadata{
				Id:        fmt.Sprintf("mock-authlog-%d", slot),
				CreatedAt: pbutils.Timestamp(createdAt),
			},
			Entry: &enterprisev1.AuthenticationLog_Entry{
				SessionRef: umetav1.GetObjectReference(sess),
				UserRef:    sess.Status.UserRef,
				DeviceRef:  sess.Status.DeviceRef,
			},
		}

		info := &corev1.Session_Status_Authentication_Info{}
		if sess.Status.Authentication != nil && sess.Status.Authentication.Info != nil {
			info = pbutils.Clone(sess.Status.Authentication.Info).(*corev1.Session_Status_Authentication_Info)
		}

		if !isFailure {
			info.Type = []corev1.Session_Status_Authentication_Info_Type{
				corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER,
				corev1.Session_Status_Authentication_Info_REFRESH_TOKEN,
				corev1.Session_Status_Authentication_Info_CREDENTIAL,
			}[slot%3]
			info.Aal = []corev1.Session_Status_Authentication_Info_AAL{
				corev1.Session_Status_Authentication_Info_AAL1,
				corev1.Session_Status_Authentication_Info_AAL2,
				corev1.Session_Status_Authentication_Info_AAL2,
				corev1.Session_Status_Authentication_Info_AAL3,
			}[slot%4]

			lg.Entry.Status = enterprisev1.AuthenticationLog_Entry_SUCCESS
			lg.Entry.AuthenticationIndex = uint32(slot % 3)
			lg.Entry.Authentication = &corev1.Session_Status_Authentication{
				Info:  info,
				SetAt: pbutils.Timestamp(createdAt),
			}

			ret = append(ret, lg)
			continue
		}

		reason := mockAuthFailureReasons[slot%int64(len(mockAuthFailureReasons))]
		info.Aal = corev1.Session_Status_Authentication_Info_AAL_UNSET
		info.Geoip = nil

		switch reason {
		case enterprisev1.AuthenticationLog_Entry_Failure_USER_NOT_FOUND:
			lg.Entry.SessionRef = nil
			lg.Entry.UserRef = nil
			lg.Entry.DeviceRef = nil
			info.Type = corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER
			info.Details = &corev1.Session_Status_Authentication_Info_IdentityProvider_{
				IdentityProvider: &corev1.Session_Status_Authentication_Info_IdentityProvider{
					Identifier: fmt.Sprintf("unknown-%d@example.com", slot%7),
					Email:      fmt.Sprintf("unknown-%d@example.com", slot%7),
					Type:       corev1.IdentityProvider_Status_OIDC,
				},
			}
		case enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL:
			info.Type = corev1.Session_Status_Authentication_Info_AUTHENTICATOR
			info.Details = &corev1.Session_Status_Authentication_Info_Authenticator_{
				Authenticator: &corev1.Session_Status_Authentication_Info_Authenticator{
					Type: corev1.Authenticator_Status_TOTP,
					Mode: corev1.Session_Status_Authentication_Info_Authenticator_MFA,
				},
			}
		default:
			info.Type = corev1.Session_Status_Authentication_Info_REFRESH_TOKEN
			info.Details = nil
		}

		lg.Entry.Status = enterprisev1.AuthenticationLog_Entry_FAILURE
		lg.Entry.Failure = &enterprisev1.AuthenticationLog_Entry_Failure{
			Reason: reason,
		}
		lg.Entry.Authentication = &corev1.Session_Status_Authentication{
			Info:  info,
			SetAt: pbutils.Timestamp(createdAt),
		}

		ret = append(ret, lg)
	}

	return ret, nil
}

func filterMockAuthenticationLogs(items []*enterprisev1.AuthenticationLog,
	status enterprisev1.AuthenticationLog_Entry_Status,
	reason enterprisev1.AuthenticationLog_Entry_Failure_Reason,
	userRef *metav1.ObjectReference) []*enterprisev1.AuthenticationLog {
	var ret []*enterprisev1.AuthenticationLog

	for _, itm := range items {
		if status != enterprisev1.AuthenticationLog_Entry_STATUS_UNSET && itm.Entry.Status != status {
			continue
		}
		if reason != enterprisev1.AuthenticationLog_Entry_Failure_REASON_UNSET && itm.Entry.Failure.GetReason() != reason {
			continue
		}
		if userRef != nil && (userRef.Uid != "" && itm.Entry.UserRef.GetUid() != userRef.Uid ||
			userRef.Name != "" && itm.Entry.UserRef.GetName() != userRef.Name) {
			continue
		}

		ret = append(ret, itm)
	}

	return ret
}

func (s *tstVisibilityLog) ListAuthenticationLog(ctx context.Context, req *visibilityv1.ListAuthenticationLogRequest) (*visibilityv1.ListAuthenticationLogResponse, error) {
	from, to := getMockAuthLogWindow(req.From, req.To, 6*time.Hour)

	items, err := s.getAuthenticationLogs(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items = filterMockAuthenticationLogs(items, req.Status, req.Reason, req.UserRef)

	itemsPerPage := uint32(25)
	var page uint32
	if req.Common != nil {
		page = req.Common.Page
		if req.Common.ItemsPerPage > 0 {
			itemsPerPage = req.Common.ItemsPerPage
		}
	}

	ret := &visibilityv1.ListAuthenticationLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{
			Page:         page,
			ItemsPerPage: itemsPerPage,
			TotalCount:   uint32(len(items)),
			HasMore:      uint32(len(items)) > (page+1)*itemsPerPage,
		},
	}

	start := int(page * itemsPerPage)
	for i := start; i < len(items) && i < start+int(itemsPerPage); i++ {
		ret.Items = append(ret.Items, items[i])
	}

	return ret, nil
}

func (s *tstVisibilityLog) doGetAuthenticationLogSummary(ctx context.Context,
	req *visibilityv1.GetAuthenticationLogSummaryRequest,
	from, to time.Time) (*visibilityv1.GetAuthenticationLogSummaryResponse, error) {
	items, err := s.getAuthenticationLogs(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items = filterMockAuthenticationLogs(items, req.Status, req.Reason, req.UserRef)

	ret := &visibilityv1.GetAuthenticationLogSummaryResponse{
		TotalByFailureReason: make(map[string]uint64),
	}

	users := make(map[string]struct{})
	sessions := make(map[string]struct{})

	for _, itm := range items {
		entry := itm.Entry
		info := entry.Authentication.GetInfo()

		ret.TotalNumber++
		if entry.Status == enterprisev1.AuthenticationLog_Entry_FAILURE {
			ret.TotalFailure++
			ret.TotalByFailureReason[entry.Failure.GetReason().String()]++
		} else {
			ret.TotalSuccess++
		}

		if entry.UserRef.GetUid() != "" {
			users[entry.UserRef.Uid] = struct{}{}
		}
		if entry.SessionRef.GetUid() != "" {
			sessions[entry.SessionRef.Uid] = struct{}{}
		}
		if entry.AuthenticationIndex > 0 {
			ret.TotalReauthentication++
		}

		switch info.GetType() {
		case corev1.Session_Status_Authentication_Info_REFRESH_TOKEN:
			ret.TotalRefreshToken++
		case corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER:
			ret.TotalIdentityProvider++
		case corev1.Session_Status_Authentication_Info_AUTHENTICATOR:
			ret.TotalAuthenticator++
		case corev1.Session_Status_Authentication_Info_CREDENTIAL:
			ret.TotalCredential++
		}

		switch info.GetAal() {
		case corev1.Session_Status_Authentication_Info_AAL1:
			ret.TotalAAL1++
		case corev1.Session_Status_Authentication_Info_AAL2:
			ret.TotalAAL2++
		case corev1.Session_Status_Authentication_Info_AAL3:
			ret.TotalAAL3++
		}

		switch info.GetAuthenticator().GetType() {
		case corev1.Authenticator_Status_FIDO:
			ret.TotalAuthenticatorFIDO++
		case corev1.Authenticator_Status_TOTP:
			ret.TotalAuthenticatorTOTP++
		case corev1.Authenticator_Status_TPM:
			ret.TotalAuthenticatorTPM++
		}

		switch info.GetAuthenticator().GetMode() {
		case corev1.Session_Status_Authentication_Info_Authenticator_MFA:
			ret.TotalAuthenticatorMFA++
		case corev1.Session_Status_Authentication_Info_Authenticator_PASSKEY:
			ret.TotalAuthenticatorPasskey++
		}
	}

	ret.TotalUser = uint64(len(users))
	ret.TotalSession = uint64(len(sessions))

	return ret, nil
}

func (s *tstVisibilityLog) GetAuthenticationLogSummary(ctx context.Context, req *visibilityv1.GetAuthenticationLogSummaryRequest) (*visibilityv1.GetAuthenticationLogSummaryResponse, error) {
	from, to := getMockAuthLogWindow(req.From, req.To, time.Hour)

	ret, err := s.doGetAuthenticationLogSummary(ctx, req, from, to)
	if err != nil {
		return nil, err
	}

	if req.CompareFrom.IsValid() && req.CompareTo.IsValid() {
		ret.Previous, err = s.doGetAuthenticationLogSummary(ctx, req,
			req.CompareFrom.AsTime(), req.CompareTo.AsTime())
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func getMockAuthLogSeriesKey(itm *enterprisev1.AuthenticationLog,
	groupBy visibilityv1.GetAuthenticationLogDataPointRequest_GroupBy) string {
	switch groupBy {
	case visibilityv1.GetAuthenticationLogDataPointRequest_STATUS:
		if itm.Entry.Status == enterprisev1.AuthenticationLog_Entry_FAILURE {
			return enterprisev1.AuthenticationLog_Entry_FAILURE.String()
		}
		return enterprisev1.AuthenticationLog_Entry_SUCCESS.String()
	case visibilityv1.GetAuthenticationLogDataPointRequest_FAILURE_REASON:
		if itm.Entry.Failure == nil {
			return ""
		}
		return itm.Entry.Failure.Reason.String()
	case visibilityv1.GetAuthenticationLogDataPointRequest_ASSURANCE_LEVEL:
		if aal := itm.Entry.Authentication.GetInfo().GetAal(); aal != corev1.Session_Status_Authentication_Info_AAL_UNSET {
			return aal.String()
		}
		return ""
	case visibilityv1.GetAuthenticationLogDataPointRequest_TYPE:
		return itm.Entry.Authentication.GetInfo().GetType().String()
	default:
		return ""
	}
}

func (s *tstVisibilityLog) GetAuthenticationLogDataPoint(ctx context.Context, req *visibilityv1.GetAuthenticationLogDataPointRequest) (*visibilityv1.GetAuthenticationLogDataPointResponse, error) {
	from, to := getMockAuthLogWindow(req.From, req.To, time.Hour)

	items, err := s.getAuthenticationLogs(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items = filterMockAuthenticationLogs(items, req.Status, req.Reason, req.UserRef)

	interval := umetav1.ToDuration(req.Interval).ToGo()
	if interval < mockAuthLogStep {
		interval = mockAuthLogStep
	}

	bucket := func(t time.Time) time.Time {
		return t.Truncate(interval)
	}

	totals := make(map[time.Time]int64)
	series := make(map[string]map[time.Time]int64)

	for _, itm := range items {
		ts := bucket(itm.Metadata.CreatedAt.AsTime())
		totals[ts]++

		if key := getMockAuthLogSeriesKey(itm, req.GroupBy); key != "" {
			if _, ok := series[key]; !ok {
				series[key] = make(map[time.Time]int64)
			}
			series[key][ts]++
		}
	}

	ret := &visibilityv1.GetAuthenticationLogDataPointResponse{}

	var buckets []time.Time
	for ts := bucket(from); !ts.After(to); ts = ts.Add(interval) {
		buckets = append(buckets, ts)
	}

	for _, ts := range buckets {
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAuthenticationLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Count:     totals[ts],
		})
	}

	for key, counts := range series {
		itm := &visibilityv1.DataPointSeries{
			Key:         key,
			DisplayName: key,
		}

		for _, ts := range buckets {
			itm.Total += counts[ts]
			itm.Datapoints = append(itm.Datapoints, &visibilityv1.DataPointSeries_DataPoint{
				Timestamp: pbutils.Timestamp(ts),
				Count:     counts[ts],
			})
		}

		ret.Series = append(ret.Series, itm)
	}

	sort.Slice(ret.Series, func(i, j int) bool {
		return ret.Series[i].Total > ret.Series[j].Total
	})

	return ret, nil
}
