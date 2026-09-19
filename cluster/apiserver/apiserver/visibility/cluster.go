// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package visibility

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	pb "github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	oc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const clusterHealthDefaultWindow = time.Hour

// maxUnhealthyComponents caps the number of the components that are reported
// by GetClusterHealth.
const maxUnhealthyComponents = 20

type ServerCluster struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedClusterServiceServer

	coreC         vcorev1.ResourceServiceClient
	accessC       vaccessv1.ResourceServiceClient
	enterpriseC   venterprisev1.ResourceServiceClient
	componentLogC pb.ComponentLogServiceClient
	accessLogC    pb.AccessLogServiceClient
}

func NewServerCluster(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerCluster, error) {

	var rscStoreHost string
	var logStoreHost string

	if ovutils.IsMockMode() {
		rscStoreHost = "localhost:40001"
		logStoreHost = "localhost:40001"
	} else if ldflags.IsTest() {
		rscStoreHost = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
		logStoreHost = rscStoreHost
	} else {
		rscStoreHost = "octeliumee-rscstore.octelium.svc:8080"
		logStoreHost = "octeliumee-logstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}

	rscStoreConn, err := grpc.NewClient(rscStoreHost, grpcOpts...)
	if err != nil {
		return nil, err
	}

	logStoreConn, err := grpc.NewClient(logStoreHost, grpcOpts...)
	if err != nil {
		return nil, err
	}

	return &ServerCluster{
		octeliumC:     octeliumC,
		coreC:         vcorev1.NewResourceServiceClient(rscStoreConn),
		accessC:       vaccessv1.NewResourceServiceClient(rscStoreConn),
		enterpriseC:   venterprisev1.NewResourceServiceClient(rscStoreConn),
		componentLogC: pb.NewComponentLogServiceClient(logStoreConn),
		accessLogC:    pb.NewAccessLogServiceClient(logStoreConn),
	}, nil
}

// summaryTask fetches the summary of a single kind into the response.
type summaryTask struct {
	kind pb.GetClusterSummaryRequest_Kind
	fn   func(context.Context, *vmetav1.CommonSummaryOptions, *pb.GetClusterSummaryResponse) error
}

func (s *ServerCluster) getSummaryTasks() []*summaryTask {
	return []*summaryTask{
		{
			kind: pb.GetClusterSummaryRequest_CORE_USER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetUserSummary(ctx, &vcorev1.GetUserSummaryRequest{Common: common})
				ret.Core.User = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_SESSION,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetSessionSummary(ctx, &vcorev1.GetSessionSummaryRequest{Common: common})
				ret.Core.Session = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_DEVICE,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetDeviceSummary(ctx, &vcorev1.GetDeviceSummaryRequest{Common: common})
				ret.Core.Device = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_SERVICE,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetServiceSummary(ctx, &vcorev1.GetServiceSummaryRequest{Common: common})
				ret.Core.Service = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_NAMESPACE,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetNamespaceSummary(ctx, &vcorev1.GetNamespaceSummaryRequest{Common: common})
				ret.Core.Namespace = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_POLICY,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetPolicySummary(ctx, &vcorev1.GetPolicySummaryRequest{Common: common})
				ret.Core.Policy = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_GROUP,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetGroupSummary(ctx, &vcorev1.GetGroupSummaryRequest{Common: common})
				ret.Core.Group = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_CREDENTIAL,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetCredentialSummary(ctx, &vcorev1.GetCredentialSummaryRequest{Common: common})
				ret.Core.Credential = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_IDENTITY_PROVIDER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetIdentityProviderSummary(ctx,
					&vcorev1.GetIdentityProviderSummaryRequest{Common: common})
				ret.Core.IdentityProvider = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_AUTHENTICATOR,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetAuthenticatorSummary(ctx,
					&vcorev1.GetAuthenticatorSummaryRequest{Common: common})
				ret.Core.Authenticator = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_SECRET,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetSecretSummary(ctx, &vcorev1.GetSecretSummaryRequest{Common: common})
				ret.Core.Secret = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_GATEWAY,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetGatewaySummary(ctx, &vcorev1.GetGatewaySummaryRequest{Common: common})
				ret.Core.Gateway = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_CORE_REGION,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.coreC.GetRegionSummary(ctx, &vcorev1.GetRegionSummaryRequest{Common: common})
				ret.Core.Region = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ACCESS_POLICY,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.accessC.GetPolicySummary(ctx, &vaccessv1.GetPolicySummaryRequest{Common: common})
				ret.Access.Policy = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ACCESS_CATALOG,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.accessC.GetCatalogSummary(ctx, &vaccessv1.GetCatalogSummaryRequest{Common: common})
				ret.Access.Catalog = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ACCESS_REQUEST,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.accessC.GetRequestSummary(ctx, &vaccessv1.GetRequestSummaryRequest{Common: common})
				ret.Access.Request = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ACCESS_REVIEW,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.accessC.GetReviewSummary(ctx, &vaccessv1.GetReviewSummaryRequest{Common: common})
				ret.Access.Review = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_COLLECTOR_EXPORTER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetCollectorExporterSummary(ctx,
					&venterprisev1.GetCollectorExporterSummaryRequest{Common: common})
				ret.Enterprise.CollectorExporter = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_SECRET,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetSecretSummary(ctx,
					&venterprisev1.GetSecretSummaryRequest{Common: common})
				ret.Enterprise.Secret = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_SECRET_STORE,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetSecretStoreSummary(ctx,
					&venterprisev1.GetSecretStoreSummaryRequest{Common: common})
				ret.Enterprise.SecretStore = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_CERTIFICATE,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetCertificateSummary(ctx,
					&venterprisev1.GetCertificateSummaryRequest{Common: common})
				ret.Enterprise.Certificate = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_CERTIFICATE_ISSUER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetCertificateIssuerSummary(ctx,
					&venterprisev1.GetCertificateIssuerSummaryRequest{Common: common})
				ret.Enterprise.CertificateIssuer = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_DNSPROVIDER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetDNSProviderSummary(ctx,
					&venterprisev1.GetDNSProviderSummaryRequest{Common: common})
				ret.Enterprise.DnsProvider = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_DIRECTORY_PROVIDER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetDirectoryProviderSummary(ctx,
					&venterprisev1.GetDirectoryProviderSummaryRequest{Common: common})
				ret.Enterprise.DirectoryProvider = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_DIRECTORY_PROVIDER_USER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetDirectoryProviderUserSummary(ctx,
					&venterprisev1.GetDirectoryProviderUserSummaryRequest{Common: common})
				ret.Enterprise.DirectoryProviderUser = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_DIRECTORY_PROVIDER_GROUP,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetDirectoryProviderGroupSummary(ctx,
					&venterprisev1.GetDirectoryProviderGroupSummaryRequest{Common: common})
				ret.Enterprise.DirectoryProviderGroup = res
				return err
			},
		},
		{
			kind: pb.GetClusterSummaryRequest_ENTERPRISE_DEVICE_MANAGER,
			fn: func(ctx context.Context, common *vmetav1.CommonSummaryOptions, ret *pb.GetClusterSummaryResponse) error {
				res, err := s.enterpriseC.GetDeviceManagerSummary(ctx,
					&venterprisev1.GetDeviceManagerSummaryRequest{Common: common})
				ret.Enterprise.DeviceManager = res
				return err
			},
		},
	}
}

func (s *ServerCluster) GetClusterSummary(ctx context.Context,
	req *pb.GetClusterSummaryRequest) (*pb.GetClusterSummaryResponse, error) {

	ret := &pb.GetClusterSummaryResponse{
		Core:       &pb.GetClusterSummaryResponse_Core{},
		Access:     &pb.GetClusterSummaryResponse_Access{},
		Enterprise: &pb.GetClusterSummaryResponse_Enterprise{},
	}

	wanted := make(map[pb.GetClusterSummaryRequest_Kind]bool, len(req.GetKinds()))
	for _, kind := range req.GetKinds() {
		wanted[kind] = true
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, task := range s.getSummaryTasks() {
		if len(wanted) > 0 && !wanted[task.kind] {
			continue
		}

		wg.Add(1)
		go func(task *summaryTask) {
			defer wg.Done()

			// Every task writes its own, distinct response field, so only the
			// unavailables slice needs to be serialized.
			err := task.fn(ctx, req.GetCommon(), ret)
			if err == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			ret.Unavailables = append(ret.Unavailables, &pb.GetClusterSummaryResponse_Unavailable{
				Kind:    task.kind,
				Message: err.Error(),
			})
		}(task)
	}

	wg.Wait()

	sort.Slice(ret.Unavailables, func(i, j int) bool {
		return ret.Unavailables[i].Kind < ret.Unavailables[j].Kind
	})

	return ret, nil
}

func worstStatus(statuses ...pb.GetClusterHealthResponse_Status) pb.GetClusterHealthResponse_Status {
	ret := pb.GetClusterHealthResponse_OK
	rank := func(status pb.GetClusterHealthResponse_Status) int {
		switch status {
		case pb.GetClusterHealthResponse_CRITICAL:
			return 3
		case pb.GetClusterHealthResponse_DEGRADED:
			return 2
		case pb.GetClusterHealthResponse_UNKNOWN:
			return 1
		default:
			return 0
		}
	}

	for _, status := range statuses {
		if rank(status) > rank(ret) {
			ret = status
		}
	}

	return ret
}

func healthStatus(critical, degraded uint64) pb.GetClusterHealthResponse_Status {
	switch {
	case critical > 0:
		return pb.GetClusterHealthResponse_CRITICAL
	case degraded > 0:
		return pb.GetClusterHealthResponse_DEGRADED
	default:
		return pb.GetClusterHealthResponse_OK
	}
}

func (s *ServerCluster) GetClusterHealth(ctx context.Context,
	req *pb.GetClusterHealthRequest) (*pb.GetClusterHealthResponse, error) {

	to := time.Now()
	if req.GetTo() != nil {
		to = req.GetTo().AsTime()
	}
	from := to.Add(-clusterHealthDefaultWindow)
	if req.GetFrom() != nil {
		from = req.GetFrom().AsTime()
	}

	ret := &pb.GetClusterHealthResponse{
		From: timestamppb.New(from),
		To:   timestamppb.New(to),
	}

	fromPB := timestamppb.New(from)
	toPB := timestamppb.New(to)

	var mu sync.Mutex
	var wg sync.WaitGroup

	addSubsystem := func(subsystem *pb.GetClusterHealthResponse_Subsystem) {
		mu.Lock()
		defer mu.Unlock()
		ret.Subsystems = append(ret.Subsystems, subsystem)
	}

	unknownSubsystem := func(sType pb.GetClusterHealthResponse_Subsystem_Type, err error) {
		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:    sType,
			Status:  pb.GetClusterHealthResponse_UNKNOWN,
			Message: fmt.Sprintf("Could not be evaluated: %s", err.Error()),
		})
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		res, err := s.componentLogC.ListComponentLogTopComponent(ctx,
			&pb.ListComponentLogTopComponentRequest{
				From:  fromPB,
				To:    toPB,
				Limit: maxUnhealthyComponents,
			})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_COMPONENTS, err)
			return
		}

		var critical, degraded uint64
		var components []*pb.GetClusterHealthResponse_Component

		for _, item := range res.Items {
			fatal := item.CountError + item.CountPanic + item.CountFatal
			if fatal == 0 && item.CountWarn == 0 {
				continue
			}

			status := healthStatus(fatal, item.CountWarn)
			switch status {
			case pb.GetClusterHealthResponse_CRITICAL:
				critical++
			case pb.GetClusterHealthResponse_DEGRADED:
				degraded++
			}

			components = append(components, &pb.GetClusterHealthResponse_Component{
				Component:   item.Component,
				Status:      status,
				TotalNumber: item.Count,
				TotalWarn:   item.CountWarn,
				TotalError:  item.CountError,
				TotalPanic:  item.CountPanic,
				TotalFatal:  item.CountFatal,
			})
		}

		sort.SliceStable(components, func(i, j int) bool {
			return components[i].TotalError+components[i].TotalPanic+components[i].TotalFatal >
				components[j].TotalError+components[j].TotalPanic+components[j].TotalFatal
		})

		mu.Lock()
		ret.Components = components
		mu.Unlock()

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_COMPONENTS,
			Status: healthStatus(critical, degraded),
			Message: fmt.Sprintf("%d of %d components emitted an error or a warning",
				critical+degraded, res.TotalCount),
			Total:     res.TotalCount,
			Unhealthy: critical + degraded,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		res, err := s.accessLogC.GetAccessLogSummary(ctx, &pb.GetAccessLogSummaryRequest{
			From: fromPB,
			To:   toPB,
		})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_AUTHORIZATION, err)
			return
		}

		status := pb.GetClusterHealthResponse_OK
		if res.TotalNumber > 0 && res.TotalDenied*2 > res.TotalNumber {
			status = pb.GetClusterHealthResponse_DEGRADED
		}

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_AUTHORIZATION,
			Status: status,
			Message: fmt.Sprintf("%d of %d authorization decisions were denials",
				res.TotalDenied, res.TotalNumber),
			Total:     res.TotalNumber,
			Unhealthy: res.TotalDenied,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		certificates, err := s.enterpriseC.GetCertificateSummary(ctx,
			&venterprisev1.GetCertificateSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_CERTIFICATES, err)
			return
		}

		issuers, err := s.enterpriseC.GetCertificateIssuerSummary(ctx,
			&venterprisev1.GetCertificateIssuerSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_CERTIFICATES, err)
			return
		}

		critical := certificates.TotalExpired + certificates.TotalIssuanceFailed + issuers.TotalNotReady
		degraded := certificates.TotalExpiringSoon

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_CERTIFICATES,
			Status: healthStatus(critical, degraded),
			Message: fmt.Sprintf("%d expired, %d expiring soon, %d failed issuance, %d issuers not ready",
				certificates.TotalExpired, certificates.TotalExpiringSoon,
				certificates.TotalIssuanceFailed, issuers.TotalNotReady),
			Total:     certificates.TotalNumber + issuers.TotalNumber,
			Unhealthy: critical + degraded,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		res, err := s.enterpriseC.GetDirectoryProviderSummary(ctx,
			&venterprisev1.GetDirectoryProviderSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_DIRECTORY_PROVIDERS, err)
			return
		}

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_DIRECTORY_PROVIDERS,
			Status: healthStatus(res.TotalSynchronizationFailed, 0),
			Message: fmt.Sprintf("%d of %d directory providers are failing to synchronize",
				res.TotalSynchronizationFailed, res.TotalNumber),
			Total:     res.TotalNumber,
			Unhealthy: res.TotalSynchronizationFailed,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		res, err := s.enterpriseC.GetSecretStoreSummary(ctx,
			&venterprisev1.GetSecretStoreSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_SECRET_STORES, err)
			return
		}

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_SECRET_STORES,
			Status: healthStatus(res.TotalSynchronizationFailed, 0),
			Message: fmt.Sprintf("%d of %d Secret stores are failing to synchronize",
				res.TotalSynchronizationFailed, res.TotalNumber),
			Total:     res.TotalNumber,
			Unhealthy: res.TotalSynchronizationFailed,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		res, err := s.enterpriseC.GetDeviceManagerSummary(ctx,
			&venterprisev1.GetDeviceManagerSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_DEVICE_MANAGERS, err)
			return
		}

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_DEVICE_MANAGERS,
			Status: healthStatus(res.TotalError, res.TotalDegraded+res.TotalFailedUpdates),
			Message: fmt.Sprintf("%d device managers are erroring, %d are degraded, %d have failed updates",
				res.TotalError, res.TotalDegraded, res.TotalFailedUpdates),
			Total:     res.TotalNumber,
			Unhealthy: res.TotalError + res.TotalDegraded,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sessions, err := s.coreC.GetSessionSummary(ctx, &vcorev1.GetSessionSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_ENROLLMENT, err)
			return
		}

		devices, err := s.coreC.GetDeviceSummary(ctx, &vcorev1.GetDeviceSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_ENROLLMENT, err)
			return
		}

		authenticators, err := s.coreC.GetAuthenticatorSummary(ctx,
			&vcorev1.GetAuthenticatorSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_ENROLLMENT, err)
			return
		}

		pending := sessions.TotalPending + devices.TotalPending + authenticators.TotalPending

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_ENROLLMENT,
			Status: healthStatus(0, pending),
			Message: fmt.Sprintf("%d Sessions, %d Devices and %d Authenticators are waiting for an approval",
				sessions.TotalPending, devices.TotalPending, authenticators.TotalPending),
			Total:     sessions.TotalNumber + devices.TotalNumber + authenticators.TotalNumber,
			Unhealthy: pending,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		requests, err := s.accessC.GetRequestSummary(ctx, &vaccessv1.GetRequestSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS, err)
			return
		}

		reviews, err := s.accessC.GetReviewSummary(ctx, &vaccessv1.GetReviewSummaryRequest{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS, err)
			return
		}

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS,
			Status: healthStatus(requests.TotalDeadlinePassed, requests.TotalPending+reviews.TotalPending),
			Message: fmt.Sprintf("%d access requests are pending, %d are past their deadline, %d reviews are open",
				requests.TotalPending, requests.TotalDeadlinePassed, reviews.TotalPending),
			Total:     requests.TotalNumber,
			Unhealthy: requests.TotalPending + requests.TotalDeadlinePassed,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		regions, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_DATA_PLANE, err)
			return
		}

		gateways, err := s.octeliumC.CoreC().ListGateway(ctx, &rmetav1.ListOptions{})
		if err != nil {
			unknownSubsystem(pb.GetClusterHealthResponse_Subsystem_DATA_PLANE, err)
			return
		}

		gatewaysByRegion := make(map[string]uint64)
		for _, gateway := range gateways.Items {
			gatewaysByRegion[gateway.Status.GetRegionRef().GetUid()]++
		}

		var withoutGateway uint64
		var items []*pb.GetClusterHealthResponse_Region

		for _, region := range regions.Items {
			total := gatewaysByRegion[region.Metadata.Uid]
			status := pb.GetClusterHealthResponse_OK
			if total == 0 {
				status = pb.GetClusterHealthResponse_CRITICAL
				withoutGateway++
			}

			items = append(items, &pb.GetClusterHealthResponse_Region{
				RegionRef:    umetav1.GetObjectReference(region),
				Status:       status,
				Version:      region.Status.GetVersion(),
				TotalGateway: total,
			})
		}

		mu.Lock()
		ret.Regions = items
		mu.Unlock()

		addSubsystem(&pb.GetClusterHealthResponse_Subsystem{
			Type:   pb.GetClusterHealthResponse_Subsystem_DATA_PLANE,
			Status: healthStatus(withoutGateway, 0),
			Message: fmt.Sprintf("%d Gateways across %d Regions",
				len(gateways.Items), len(regions.Items)),
			Total:     uint64(len(regions.Items)),
			Unhealthy: withoutGateway,
		})
	}()

	wg.Wait()

	sort.Slice(ret.Subsystems, func(i, j int) bool {
		return ret.Subsystems[i].Type < ret.Subsystems[j].Type
	})

	statuses := make([]pb.GetClusterHealthResponse_Status, 0, len(ret.Subsystems))
	for _, subsystem := range ret.Subsystems {
		statuses = append(statuses, subsystem.Status)
	}
	ret.Status = worstStatus(statuses...)

	return ret, nil
}
