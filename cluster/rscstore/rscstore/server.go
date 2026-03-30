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
	"database/sql"
	"net"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	watchercore "github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

const tstAddr = "localhost:32123"

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string
	genCache      *cache.Cache
	db            *sql.DB
	auditLogItem  chan umetav1.ResourceObjectI
	client        plogotlp.GRPCClient
	idxDebouncer  idxDebouncer
}

type idxDebouncer struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
	s     *Server
}

func (d *idxDebouncer) debounce() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.after, d.s.runRecreateFTSIndex)
}

func getAddr() string {
	if ldflags.IsTest() {
		return "localhost:54321"
	}

	return "octeliumee-logstore.octelium.svc:8080"
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC:    octeliumC,
		genCache:     cache.New(cache.NoExpiration, 1*time.Minute),
		auditLogItem: make(chan umetav1.ResourceObjectI, 10000),
	}

	ret.idxDebouncer = idxDebouncer{
		after: 3 * time.Second,
		s:     ret,
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	ret.db, err = sql.Open("duckdb", ovutils.GetDuckDBDSN())
	if err != nil {
		return nil, err
	}

	{
		grpcOpts := []grpc.DialOption{
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff: backoff.DefaultConfig,
			}),
		}

		{
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		conn, err := grpc.NewClient(getAddr(), grpcOpts...)
		if err != nil {
			return nil, err
		}

		ret.client = plogotlp.NewGRPCClient(conn)
	}

	return ret, nil
}

func (s *Server) Run(ctx context.Context) error {

	if err := s.initDB(ctx); err != nil {
		return err
	}

	if err := s.initGRPC(ctx); err != nil {
		return err
	}

	if err := s.setResources(ctx); err != nil {
		return err
	}

	go s.startProcessAuditLogLoop(ctx)

	return nil
}

func Run(ctx context.Context) error {
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := s.Run(ctx); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.S().Infof("RscStore is running...")

	<-ctx.Done()

	return nil
}

func (s *Server) initGRPC(ctx context.Context) error {

	grpcSrv := grpc.NewServer()

	vcorev1.RegisterResourceServiceServer(grpcSrv, &srvCore{
		s: s,
	})

	go func() {

		lis, err := net.Listen("tcp", func() string {

			if ovutils.IsMockMode() {
				return "localhost:40001"
			}

			if ldflags.IsTest() {
				return tstAddr
			}

			return ":8080"
		}())
		if err != nil {
			return
		}
		grpcSrv.Serve(lis)
	}()

	return nil
}

func (s *Server) initDB(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS resources (api VARCHAR, version VARCHAR, kind VARCHAR, uid VARCHAR, resource_version VARCHAR, rsc JSON, rsc_str VARCHAR, PRIMARY KEY (uid))`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS versioned_resources (api VARCHAR, version VARCHAR, kind VARCHAR, uid VARCHAR, deleted BOOLEAN, resource_version VARCHAR, rsc JSON, PRIMARY KEY (uid, resource_version))`); err != nil {
		return err
	}

	/*
		s.db.Exec(`INSTALL json`)
		s.db.Exec(`LOAD json`)
	*/

	/*
		if _, err := s.db.ExecContext(ctx, `INSTALL fts`); err != nil {
			return err
		}

		if _, err := s.db.ExecContext(ctx, `LOAD fts`); err != nil {
			return err
		}
	*/

	if err := s.recreateFTSIndex(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) recreateFTSIndex(ctx context.Context) error {
	// zap.L().Debug("Recreating FTS index")
	if _, err := s.db.ExecContext(ctx,
		`PRAGMA drop_fts_index('resources')`); err != nil {
		zap.L().Warn("Could not drop_ftx_index", zap.Error(err))
	}

	if _, err := s.db.ExecContext(ctx,
		`PRAGMA create_fts_index('resources', 'uid', 'rsc_str')`); err != nil {
		zap.L().Warn("Could not create_fts_index", zap.Error(err))
		return err
	}

	return nil
}

func (s *Server) runRecreateFTSIndex() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.recreateFTSIndex(ctx); err != nil {
		zap.L().Warn("Could not recreateFTSIndex", zap.Error(err))
	}
}

func (s *Server) setResources(ctx context.Context) error {
	w := watchercore.NewCoreV1(s.octeliumC)

	if err := w.User(ctx, nil,
		func(ctx context.Context, item *corev1.User) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.User) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.User) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Session(ctx, nil,
		func(ctx context.Context, item *corev1.Session) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Session) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Session) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Device(ctx, nil,
		func(ctx context.Context, item *corev1.Device) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Device) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Device) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Group(ctx, nil,
		func(ctx context.Context, item *corev1.Group) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Group) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Group) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Service(ctx, nil,
		func(ctx context.Context, item *corev1.Service) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Service) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Service) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Namespace(ctx, nil,
		func(ctx context.Context, item *corev1.Namespace) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Namespace) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Namespace) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Secret(ctx, nil,
		func(ctx context.Context, item *corev1.Secret) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Secret) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Secret) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Policy(ctx, nil,
		func(ctx context.Context, item *corev1.Policy) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Policy) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Policy) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.IdentityProvider(ctx, nil,
		func(ctx context.Context, item *corev1.IdentityProvider) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.IdentityProvider) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.IdentityProvider) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Authenticator(ctx, nil,
		func(ctx context.Context, item *corev1.Authenticator) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Authenticator) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Authenticator) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Region(ctx, nil,
		func(ctx context.Context, item *corev1.Region) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Region) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Region) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Gateway(ctx, nil,
		func(ctx context.Context, item *corev1.Gateway) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Gateway) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Gateway) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	if err := w.Credential(ctx, nil,
		func(ctx context.Context, item *corev1.Credential) error {
			return s.insertResource(ctx, item)
		}, func(ctx context.Context, new, old *corev1.Credential) error {
			return s.insertResource(ctx, new)
		}, func(ctx context.Context, item *corev1.Credential) error {
			return s.removeResource(ctx, item)
		}); err != nil {
		return err
	}

	return nil
}

func DoRun(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := s.Run(ctx); err != nil {
		return err
	}

	return nil
}
