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
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/cluster"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/visibility"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/policyportal/policyportal"
	"github.com/octelium/octelium-ee/cluster/rscserver/rscserver"
	"github.com/octelium/octelium-ee/cluster/rscstore/rscstore"
	"github.com/octelium/octelium/apis/main/authv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/clusterconfig"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/sessionc"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/improbable-eng/grpc-web/go/grpcweb"

	"github.com/octelium/octelium/cluster/authserver/authserver"
)

func Run(ctx context.Context) error {
	ovutils.SetMockMode()

	{
		zapCfg := zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}

		logger, err := zapCfg.Build()
		if err != nil {
			return err
		}

		zap.ReplaceGlobals(logger)
	}

	{
		dbName := fmt.Sprintf("octelium%s", utilrand.GetRandomStringLowercase(8))

		os.Setenv("OCTELIUM_POSTGRES_NOSSL", "true")

		dir, err := os.MkdirTemp("", "")
		defer os.RemoveAll(dir)

		os.Setenv("OCTELIUM_DUCKDB_PATH", path.Join(dir, "store.db"))

		os.Setenv("OCTELIUM_POSTGRES_HOST", "localhost")
		os.Setenv("OCTELIUM_POSTGRES_USERNAME", "postgres")
		os.Setenv("OCTELIUM_POSTGRES_PASSWORD", "postgres")
		os.Setenv("OCTELIUM_TEST_RSCSERVER_PORT", fmt.Sprintf("%d", 10001))

		ldflags.PrivateRegistry = "false"
		ldflags.Mode = "production"
		ldflags.TestMode = "true"

		{
			db, err := postgresutils.NewDBWithNODB()
			if err != nil {
				return err
			}
			if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
				return err
			}
			if err := db.Close(); err != nil {
				return err
			}
		}

		zap.S().Debugf("Starting new rsc server")

		os.Setenv("OCTELIUM_POSTGRES_DATABASE", dbName)

		rscSrv, err := rscserver.NewServer(ctx)
		if err != nil {
			return err
		}

		zap.S().Debugf("Running rsc server")
		err = rscSrv.Run(ctx)
		if err != nil {
			return err
		}

		time.Sleep(5 * time.Second)

		{
			clusterCfg := &corev1.ClusterConfig{
				ApiVersion: "cluster/v1",
				Kind:       "ClusterConfig",
				Metadata: &metav1.Metadata{
					Uid:             uuid.New().String(),
					ResourceVersion: uuid.New().String(),
					Name:            "default",
				},
				Spec: &corev1.ClusterConfig_Spec{},
				Status: &corev1.ClusterConfig_Status{
					Domain:  "example.com",
					Network: &corev1.ClusterConfig_Status_Network{},
				},
			}

			v6Prefix, err := utilrand.GetRandomBytes(4)
			if err != nil {
				return err
			}

			clusterCfg.Status.Network.V6RangePrefix = v6Prefix

			if err := clusterconfig.SetClusterSubnets(clusterCfg); err != nil {
				return err
			}

			_, err = rscSrv.CreateResource(ctx, clusterCfg, ucorev1.API, ucorev1.Version, ucorev1.KindClusterConfig)
			if err != nil {
				return err
			}
		}

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

			_, err = rscSrv.CreateResource(ctx, clusterCfg, "enterprise", "v1", "ClusterConfig")
			if err != nil {
				return err
			}
		}
	}

	zap.S().Debug("Starting octelium API server...")

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	srv := admin.NewServer(&admin.Opts{
		OcteliumC: octeliumC,
	})
	usrSrv := user.NewServer(octeliumC)
	eeSrv := enterprise.NewServer(octeliumC)

	accessRscSrv, err := visibility.NewServerResource(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := genResources(ctx, octeliumC); err != nil {
		return err
	}

	authSrv, err := authserver.GetAuthGRPCServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	zap.S().Debug("starting gRPC server...")

	mdlwr, err := newMiddleware(ctx, octeliumC)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(mdlwr.StreamServerInterceptor())),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(mdlwr.UnaryServerInterceptor())),
	)

	corev1.RegisterMainServiceServer(s, srv)
	userv1.RegisterMainServiceServer(s, usrSrv)
	authv1.RegisterMainServiceServer(s, authSrv)
	enterprisev1.RegisterMainServiceServer(s, eeSrv)

	vSrv := &tstVisibilityLog{
		octeliumC: octeliumC,
	}
	visibilityv1.RegisterAccessLogServiceServer(s, vSrv)
	visibilityv1.RegisterAuthenticationLogServiceServer(s, vSrv)

	vcorev1.RegisterResourceServiceServer(s, accessRscSrv)

	{
		pSrv, err := policyportal.NewServer(ctx, octeliumC)
		if err != nil {
			return err
		}

		enterprisev1.RegisterPolicyPortalServiceServer(s, pSrv)
	}
	{
		pSrv, err := cluster.NewServer(octeliumC)
		if err != nil {
			return err
		}

		enterprisev1.RegisterClusterServiceServer(s, pSrv)
	}

	go func() {
		zap.S().Debug("running gRPC server.")
		if err := s.Serve(lis); err != nil {
			zap.S().Infof("gRPC server closed: %+v", err)
		}
	}()

	go func() {

		zap.L().Debug("starting grpcWeb server")

		grpcWebSrv := &grpcWebSrv{
			srv: grpcweb.WrapServer(s),
		}

		srv := &http.Server{
			Handler: grpcWebSrv,
			Addr:    "127.0.0.1:10003",
		}

		if err := srv.ListenAndServe(); err != nil {
			zap.L().Fatal("Could not serve grpcWeb server", zap.Error(err))
		}
	}()

	if err := rscstore.DoRun(ctx, octeliumC); err != nil {
		return err
	}

	zap.L().Info("Mock API Server is now running")
	<-ctx.Done()
	zap.L().Debug("Shutting down gRPC server")
	s.Stop()

	return nil
}

type grpcWebSrv struct {
	srv *grpcweb.WrappedGrpcServer
}

func (s *grpcWebSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	/*
		zap.L().Debug("New req",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("scheme", r.URL.Scheme),
			zap.String("host", r.Host),
			zap.Any("hdrs", r.Header),
			zap.Bool("isGRPC", s.srv.IsGrpcWebRequest(r)),
		)
	*/
	s.srv.ServeHTTP(w, r)
}

type Middleware struct {
	usr  *corev1.User
	sess *corev1.Session
	dev  *corev1.Device

	accessToken  string
	refreshToken string
}

func newMiddleware(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Middleware, error) {

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC: octeliumC,
	})

	usr, err := adminSrv.CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: "george@octelium.com",
		},
	})
	if err != nil {
		return nil, err
	}

	dev, err := octeliumC.CoreC().CreateDevice(ctx, &corev1.Device{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(6),
		},
		Spec: &corev1.Device_Spec{
			State: corev1.Device_Spec_ACTIVE,
		},
		Status: &corev1.Device_Status{
			UserRef: umetav1.GetObjectReference(usr),
			OsType:  corev1.Device_Status_LINUX,
		},
	})
	if err != nil {
		return nil, err
	}

	sess, err := sessionc.CreateSession(ctx, &sessionc.CreateSessionOpts{
		Usr:       usr,
		Device:    dev,
		OcteliumC: octeliumC,
		SessType:  corev1.Session_Status_CLIENTLESS,
		IsBrowser: true,
	})
	if err != nil {
		return nil, err
	}

	jwkCtl, err := jwkctl.NewJWKController(ctx, octeliumC)
	if err != nil {
		return nil, err
	}

	accessToken, err := jwkCtl.CreateAccessToken(sess)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwkCtl.CreateRefreshToken(sess)
	if err != nil {
		return nil, err
	}

	return &Middleware{
		usr:  usr,
		dev:  dev,
		sess: sess,

		accessToken:  accessToken,
		refreshToken: refreshToken,
	}, nil

}

func (m *Middleware) getDownstream(ctx context.Context) (*userctx.UserCtx, error) {

	return &userctx.UserCtx{
		User:    m.usr,
		Session: m.sess,
		Groups:  nil,
		Device:  m.dev,
	}, nil
}

func (m *Middleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		zap.L().Debug("New req", zap.String("fullMethod", info.FullMethod))
		i, err := m.getDownstream(ctx)
		if err != nil {
			zap.S().Debugf("Could not authenticate User: %+v", err)
			return nil, status.Errorf(codes.Unauthenticated, "Could not authenticate User")
		}

		newCtx := context.WithValue(ctx, "octelium-user-ctx", i)
		newCtx = context.WithValue(newCtx, "x-octelium-auth", m.accessToken)
		newCtx = context.WithValue(newCtx, "x-octelium-refresh-token", m.refreshToken)

		md, _ := metadata.FromIncomingContext(newCtx)

		md["x-octelium-refresh-token"] = []string{
			m.refreshToken,
		}
		md["x-octelium-auth"] = []string{
			m.accessToken,
		}

		md["x-octelium-req-path"] = []string{
			info.FullMethod,
		}

		newCtx = metadata.NewIncomingContext(newCtx, md)

		return handler(newCtx, req)
	}
}

func (m *Middleware) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		ctx := stream.Context()

		i, err := m.getDownstream(ctx)
		if err != nil {
			zap.S().Debugf("Could not authenticate User: %+v", err)
			return status.Errorf(codes.Unauthenticated, "Could not authenticate User")
		}

		newCtx := context.WithValue(ctx, "octelium-user-ctx", i)
		newCtx = context.WithValue(newCtx, "x-octelium-auth", m.accessToken)
		newCtx = context.WithValue(newCtx, "x-octelium-refresh-token", m.refreshToken)

		_, _ = metadata.FromIncomingContext(newCtx)

		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx

		return handler(srv, wrapped)
	}
}

type tstVisibilityLog struct {
	octeliumC octeliumc.ClientInterface
	visibilityv1.UnimplementedAccessLogServiceServer
	visibilityv1.UnimplementedAuthenticationLogServiceServer
	visibilityv1.UnimplementedComponentLogServiceServer
	visibilityv1.UnimplementedAuditLogServiceServer
}

func (s *tstVisibilityLog) ListAccessLog(ctx context.Context, req *visibilityv1.ListAccessLogRequest) (*visibilityv1.ListAccessLogResponse, error) {

	zap.L().Debug("ListAccessLog req", zap.Any("req", req))
	sess, err := s.getSessionRandom(ctx)
	if err != nil {
		return nil, err
	}
	svc, err := s.getServiceRandom(ctx)
	if err != nil {
		return nil, err
	}

	return &visibilityv1.ListAccessLogResponse{
		Items: func() []*corev1.AccessLog {
			var ret []*corev1.AccessLog
			for range 100 {
				lg := vutils.GenerateLog()

				lg.Entry = &corev1.AccessLog_Entry{
					Common: &corev1.AccessLog_Entry_Common{
						SessionRef:   umetav1.GetObjectReference(sess),
						UserRef:      sess.Status.UserRef,
						DeviceRef:    sess.Status.DeviceRef,
						ServiceRef:   umetav1.GetObjectReference(svc),
						NamespaceRef: svc.Status.NamespaceRef,
					},
				}
				ret = append(ret, lg)
			}

			return ret
		}(),
		ListResponseMeta: &metav1.ListResponseMeta{
			Page: func() uint32 {
				if req.Common != nil {
					return req.Common.Page
				}
				return 0
			}(),
			ItemsPerPage: 100,
			TotalCount:   1000,
			HasMore: func() bool {
				if req.Common != nil && req.Common.Page > 10 {
					return false
				}
				return true
			}(),
		},
	}, nil
}

func (s *tstVisibilityLog) ListAccessLogTopUser(ctx context.Context, req *visibilityv1.ListAccessLogTopUserRequest) (*visibilityv1.ListAccessLogTopUserResponse, error) {
	itmList, err := s.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &visibilityv1.ListAccessLogTopUserResponse{}
	for i, itm := range itmList.Items[:min(len(itmList.Items), 10)] {
		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopUserResponse_Item{
			User:  itm,
			Count: int32(100 - i),
		})
	}
	return ret, nil
}

func (s *tstVisibilityLog) ListAccessLogTopService(ctx context.Context, req *visibilityv1.ListAccessLogTopServiceRequest) (*visibilityv1.ListAccessLogTopServiceResponse, error) {
	itmList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &visibilityv1.ListAccessLogTopServiceResponse{}
	for i, itm := range itmList.Items[:min(len(itmList.Items), 10)] {
		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopServiceResponse_Item{
			Service: itm,
			Count:   int32(100 - i),
		})
	}
	return ret, nil
}

func (s *tstVisibilityLog) ListAccessLogTopPolicy(ctx context.Context, req *visibilityv1.ListAccessLogTopPolicyRequest) (*visibilityv1.ListAccessLogTopPolicyResponse, error) {
	itmList, err := s.octeliumC.CoreC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &visibilityv1.ListAccessLogTopPolicyResponse{}
	for i, itm := range itmList.Items[:min(len(itmList.Items), 10)] {
		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopPolicyResponse_Item{
			Policy: itm,
			Count:  int32(100 - i),
		})
	}
	return ret, nil
}

func (s *tstVisibilityLog) ListAccessLogTopSession(ctx context.Context, req *visibilityv1.ListAccessLogTopSessionRequest) (*visibilityv1.ListAccessLogTopSessionResponse, error) {

	itmList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &visibilityv1.ListAccessLogTopSessionResponse{}
	for i, itm := range itmList.Items[:min(len(itmList.Items), 10)] {
		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopSessionResponse_Item{
			Session: itm,
			Count:   int32(100 - i),
		})
	}
	return ret, nil
}

func (s *tstVisibilityLog) ListSSHSession(ctx context.Context, req *visibilityv1.ListSSHSessionRequest) (*visibilityv1.ListSSHSessionResponse, error) {
	zap.L().Debug("ListSSHSession req", zap.Any("req", req))
	sess, err := s.getSessionRandom(ctx)
	if err != nil {
		return nil, err
	}
	svc, err := s.getServiceRandom(ctx)
	if err != nil {
		return nil, err
	}

	return &visibilityv1.ListSSHSessionResponse{
		Items: func() []*visibilityv1.SSHSession {
			var ret []*visibilityv1.SSHSession
			for range 100 {

				ret = append(ret, &visibilityv1.SSHSession{
					Id:           vutils.GenerateLogID(),
					State:        visibilityv1.SSHSession_COMPLETED,
					StartedAt:    pbutils.Timestamp(time.Now().Add(-5 * time.Minute)),
					EndedAt:      pbutils.Now(),
					SessionRef:   umetav1.GetObjectReference(sess),
					UserRef:      sess.Status.UserRef,
					ServiceRef:   umetav1.GetObjectReference(svc),
					NamespaceRef: svc.Status.NamespaceRef,
				})
			}

			return ret
		}(),
		ListResponseMeta: &metav1.ListResponseMeta{
			Page: func() uint32 {
				if req.Common != nil {
					return req.Common.Page
				}
				return 0
			}(),
			ItemsPerPage: 100,
			TotalCount:   1000,
			HasMore: func() bool {
				if req.Common != nil && req.Common.Page > 10 {
					return false
				}
				return true
			}(),
		},
	}, nil
}

func (s *tstVisibilityLog) GetAccessLogDataPoint(ctx context.Context, req *visibilityv1.GetAccessLogDataPointRequest) (*visibilityv1.GetAccessLogDataPointResponse, error) {

	zap.L().Debug("REQ GetAccessLogDataPoint", zap.Any("req", req))
	ret := &visibilityv1.GetAccessLogDataPointResponse{}

	start := utilrand.GetRandomRangeMath(1, 100)
	end := utilrand.GetRandomRangeMath(start, 10*start)

	max := 300
	for i := range max {
		n := utilrand.GetRandomRangeMath(start, end)
		ts := time.Now().Add(time.Duration(i-max) * time.Hour)
		start = n - n/2
		end = n * 2
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAccessLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Count:     int64(utilrand.GetRandomRangeMath(10, 100)),
		})
	}

	return ret, nil
}

func (s *tstVisibilityLog) GetAuthenticationLogDataPoint(ctx context.Context, req *visibilityv1.GetAuthenticationLogDataPointRequest) (*visibilityv1.GetAuthenticationLogDataPointResponse, error) {
	ret := &visibilityv1.GetAuthenticationLogDataPointResponse{}

	start := utilrand.GetRandomRangeMath(1, 100)
	end := utilrand.GetRandomRangeMath(start, 10*start)

	max := 300
	for i := range max {
		n := utilrand.GetRandomRangeMath(start, end)
		ts := time.Now().Add(time.Duration(i-max) * time.Hour)
		start = n - n/2
		end = n * 2
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAuthenticationLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Count:     int64(utilrand.GetRandomRangeMath(10, 100)),
		})
	}

	return ret, nil
}

func (s *tstVisibilityLog) ListSSHSessionRecording(ctx context.Context, req *visibilityv1.ListSSHSessionRecordingRequest) (*visibilityv1.ListSSHSessionRecordingResponse, error) {

	return &visibilityv1.ListSSHSessionRecordingResponse{}, nil
}

func (s *tstVisibilityLog) GetAccessLogSummary(ctx context.Context, req *visibilityv1.GetAccessLogSummaryRequest) (*visibilityv1.GetAccessLogSummaryResponse, error) {
	return &visibilityv1.GetAccessLogSummaryResponse{}, nil
}

func (s *tstVisibilityLog) GetSSHSession(ctx context.Context, req *visibilityv1.GetSSHSessionRequest) (*visibilityv1.SSHSession, error) {

	return &visibilityv1.SSHSession{}, nil
}

func (s *tstVisibilityLog) getSessionRandom(ctx context.Context) (*corev1.Session, error) {
	sessList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return sessList.Items[0], nil
}

func (s *tstVisibilityLog) getServiceRandom(ctx context.Context) (*corev1.Service, error) {
	itmList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return itmList.Items[0], nil
}
