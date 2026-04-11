// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package policyportal

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/cluster/coctovigilv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/sessionc"
	"github.com/octelium/octelium/cluster/common/spiffec"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/cluster/octovigil/octovigil"
	policycontroller "github.com/octelium/octelium/cluster/octovigil/octovigil/controllers/policies"
	ptctl "github.com/octelium/octelium/cluster/octovigil/octovigil/controllers/policytemplates"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string
	genCache      *cache.Cache

	rootURL string

	oidcInfoMap map[string]*oidcInfo

	s *octovigil.Server
	enterprisev1.UnimplementedPolicyPortalServiceServer
}

type oidcInfo struct {
	regionRef *metav1.ObjectReference
	oidc      []byte
	jwks      []byte
}

func NewServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC:   octeliumC,
		oidcInfoMap: make(map[string]*oidcInfo),

		genCache: cache.New(cache.NoExpiration, 1*time.Minute),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	ret.rootURL = fmt.Sprintf("https://public.octelium.%s", cc.Status.Domain)

	ret.s, err = octovigil.New(ctx, octeliumC)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Server) run(ctx context.Context) error {
	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		return err
	}

	grpcSrv := grpc.NewServer(
		cred,
		grpc.ReadBufferSize(32*1024),
		grpc.MaxConcurrentStreams(1000000),
	)
	enterprisev1.RegisterPolicyPortalServiceServer(grpcSrv, s)
	go func() {

		lis, err := net.Listen("tcp", func() string {
			return ":8080"
		}())
		if err != nil {
			return
		}
		grpcSrv.Serve(lis)
	}()
	return nil
}

func Run(ctx context.Context) error {
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	srv, err := NewServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := srv.run(ctx); err != nil {
		return err
	}

	watcher := watchers.NewCoreV1(octeliumC)

	s := srv.s

	policyCtl := policycontroller.NewController(s.GetCache())
	ptCtl := ptctl.NewController(s.GetPolicyTriggerCtl())

	if err := watcher.Policy(ctx, nil, policyCtl.OnAdd, policyCtl.OnUpdate, policyCtl.OnDelete); err != nil {
		return err
	}

	if err := watcher.PolicyTrigger(ctx, nil, ptCtl.OnAdd, ptCtl.OnUpdate, ptCtl.OnDelete); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.S().Infof("Policy Portal server is running...")

	<-ctx.Done()

	return nil
}

func (s *Server) IsAuthorized(ctx context.Context,
	req *enterprisev1.IsAuthorizedRequest) (*enterprisev1.IsAuthorizedResponse, error) {

	if req.Downstream == nil {
		return nil, grpcutils.InvalidArg("")
	}
	if req.Upstream == nil {
		return nil, grpcutils.InvalidArg("")
	}
	var err error

	cc, err := s.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	reqCtx := &corev1.RequestContext{
		Request: req.Request,
	}

	switch req.Downstream.(type) {
	case *enterprisev1.IsAuthorizedRequest_SessionRef:
		sess, err := s.octeliumC.CoreC().GetSession(ctx, apivalidation.ObjectReferenceToRGetOptions(req.GetSessionRef()))
		if err != nil {
			return nil, err
		}
		reqCtx.Session = sess

		usr, err := s.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(sess.Status.UserRef))
		if err != nil {
			return nil, err
		}
		reqCtx.User = usr

		for _, g := range usr.Spec.Groups {
			grp, err := s.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
				Name: g,
			})
			if err == nil {
				reqCtx.Groups = append(reqCtx.Groups, grp)
			}
		}

		if sess.Status.DeviceRef != nil {
			dev, err := s.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{
				Uid: sess.Status.DeviceRef.Uid,
			})
			if err == nil {
				reqCtx.Device = dev
			}
		}

	case *enterprisev1.IsAuthorizedRequest_DeviceRef:
		dev, err := s.octeliumC.CoreC().GetDevice(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.GetDeviceRef()))
		if err != nil {
			return nil, err
		}

		usr, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dev.Status.UserRef.Uid,
		})
		if err != nil {
			return nil, err
		}

		reqCtx.Device = dev
		reqCtx.User = usr

		for _, g := range usr.Spec.Groups {
			grp, err := s.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
				Name: g,
			})
			if err == nil {
				reqCtx.Groups = append(reqCtx.Groups, grp)
			}
		}

		sess, err := sessionc.NewSession(ctx, &sessionc.CreateSessionOpts{
			Usr:           usr,
			ClusterConfig: cc,
			SessType:      corev1.Session_Status_CLIENT,
		})
		if err != nil {
			return nil, err
		}
		reqCtx.Session = sess
	case *enterprisev1.IsAuthorizedRequest_UserRef:
		usr, err := s.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(req.GetUserRef()))
		if err != nil {
			return nil, err
		}
		reqCtx.User = usr

		for _, g := range usr.Spec.Groups {
			grp, err := s.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
				Name: g,
			})
			if err == nil {
				reqCtx.Groups = append(reqCtx.Groups, grp)
			}
		}

		sess, err := sessionc.NewSession(ctx, &sessionc.CreateSessionOpts{
			Usr:           usr,
			ClusterConfig: cc,
		})
		if err != nil {
			return nil, err
		}
		reqCtx.Session = sess
	default:
		return nil, grpcutils.InvalidArg("")
	}

	switch req.Upstream.(type) {
	case *enterprisev1.IsAuthorizedRequest_ServiceRef:
		svc, err := s.octeliumC.CoreC().GetService(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.GetServiceRef()))
		if err != nil {
			return nil, err
		}

		ns, err := s.octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
			Uid: svc.Status.NamespaceRef.Uid,
		})
		if err != nil {
			return nil, err
		}
		reqCtx.Service = svc
		reqCtx.Namespace = ns
	case *enterprisev1.IsAuthorizedRequest_NamespaceRef:
		ns, err := s.octeliumC.CoreC().GetNamespace(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.GetNamespaceRef()))
		if err != nil {
			return nil, err
		}
		reqCtx.Namespace = ns

		rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: "default",
		})
		if err != nil {
			return nil, err
		}

		svc := &corev1.Service{
			Metadata: &metav1.Metadata{
				Uid:  vutils.UUIDv4(),
				Name: fmt.Sprintf("fake-service.%s", ns.Metadata.Name),
			},
			Spec: &corev1.Service_Spec{
				Port: 8080,
				Mode: corev1.Service_Spec_HTTP,
				Config: &corev1.Service_Spec_Config{
					Upstream: &corev1.Service_Spec_Config_Upstream{
						Type: &corev1.Service_Spec_Config_Upstream_Url{
							Url: "https://www.example.com",
						},
					},
				},
			},
			Status: &corev1.Service_Status{
				Port:            8080,
				NamespaceRef:    umetav1.GetObjectReference(ns),
				RegionRef:       umetav1.GetObjectReference(rgn),
				PrimaryHostname: "fake-service",
			},
		}

		reqCtx.Service = svc
	default:
		return nil, grpcutils.InvalidArg("")
	}

	var additional *coctovigilv1.Authorization
	if req.Additional != nil {
		additional = &coctovigilv1.Authorization{
			Policies:       req.Additional.Policies,
			InlinePolicies: req.Additional.InlinePolicies,
		}
	}

	res, err := s.s.DoAuthorize(ctx, reqCtx, additional)
	if err != nil {
		return nil, err
	}

	return &enterprisev1.IsAuthorizedResponse{
		IsAuthorized: res.IsAuthorized,
		Reason:       res.Reason,
	}, nil
}
