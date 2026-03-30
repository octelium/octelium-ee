// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

const tstAddr = "localhost:32123"

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain   string
	genCache        *cache.Cache
	db              *sql.DB
	cleanupDuration time.Duration
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC:       octeliumC,
		genCache:        cache.New(cache.NoExpiration, 1*time.Minute),
		cleanupDuration: 30 * 24 * time.Hour,
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

	return ret, nil
}

func (s *Server) Run(ctx context.Context) error {

	if err := s.initDB(ctx); err != nil {
		return err
	}

	if err := s.initGRPC(ctx); err != nil {
		return err
	}

	go s.startCleanupLoop(ctx)

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

func Run(ctx context.Context) error {
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	if err := DoRun(ctx, octeliumC); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.S().Infof("LogStore is running...")

	<-ctx.Done()

	return nil
}

func (s *Server) initGRPC(ctx context.Context) error {

	grpcSrv := grpc.NewServer()

	srvLog := s.newSrvLog()

	go srvLog.startProcessLoop(ctx)

	plogotlp.RegisterGRPCServer(grpcSrv, srvLog)

	visibilityv1.RegisterAccessLogServiceServer(grpcSrv, &srvAccessLog{
		s: s,
	})

	visibilityv1.RegisterAuthenticationLogServiceServer(grpcSrv, &srvAuthenticationLog{
		s: s,
	})

	visibilityv1.RegisterAuditLogServiceServer(grpcSrv, &srvAuditLog{
		s: s,
	})

	visibilityv1.RegisterComponentLogServiceServer(grpcSrv, &srvComponentLog{
		s: s,
	})

	zap.L().Debug("Starting gRPC sever", zap.Bool("mockMode", ovutils.IsMockMode()))

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
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS access_logs (rsc JSON)`)
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS component_logs (rsc JSON)`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS authentication_logs (rsc JSON)`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS audit_logs (rsc JSON)`); err != nil {
		return err
	}

	return nil
}

func (s *Server) startCleanupLoop(ctx context.Context) {
	tickerCh := time.NewTicker(15 * time.Minute)
	defer tickerCh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh.C:
			if err := s.doCleanup(ctx); err != nil {
				zap.L().Error("Could not do cleanup", zap.Error(err))
			}
		}
	}
}

var maxDBAccessLogs = 2_000_000
var maxDBAuthenticationLogs = 100_000
var maxDBAuditLogs = 100_000
var maxDBComponentLogs = 100_000

func (s *Server) doCleanup(ctx context.Context) error {
	monthAgo := pbutils.Now().AsTime().Add(-1 * s.cleanupDuration).UTC().Format(time.RFC3339Nano)

	{
		if _, err := s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM access_logs WHERE rsc->'metadata'->>'createdAt' < '%s'`, monthAgo)); err != nil {
			zap.L().Warn("Could not cleanup access_logs", zap.Error(err))
		}

		if _, err := s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM component_logs WHERE rsc->'metadata'->>'createdAt' < '%s'`, monthAgo)); err != nil {
			zap.L().Warn("Could not cleanup component_logs", zap.Error(err))
		}

		if _, err := s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM audit_logs WHERE rsc->'metadata'->>'createdAt' < '%s'`, monthAgo)); err != nil {
			zap.L().Warn("Could not cleanup audit_logs", zap.Error(err))
		}

		if _, err := s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM authentication_logs WHERE rsc->'metadata'->>'createdAt' < '%s'`, monthAgo)); err != nil {
			zap.L().Warn("Could not cleanup authentication_logs", zap.Error(err))

		}
	}

	{

		getQry := func(table string, limit int) string {
			return fmt.Sprintf(`DELETE FROM %s WHERE rsc->'metadata'->>'uid' NOT IN (SELECT rsc->'metadata'->>'uid' FROM %s ORDER BY rsc->'metadata'->>'createdAt' DESC LIMIT %d)`,
				table, table, limit)
		}

		if _, err := s.db.ExecContext(ctx, getQry("access_logs", maxDBAccessLogs)); err != nil {
			zap.L().Warn("Could not cleanup access_logs by max", zap.Error(err))
		}
		if _, err := s.db.ExecContext(ctx, getQry("component_logs", maxDBComponentLogs)); err != nil {
			zap.L().Warn("Could not cleanup component_logs by max", zap.Error(err))
		}
		if _, err := s.db.ExecContext(ctx, getQry("audit_logs", maxDBAuditLogs)); err != nil {
			zap.L().Warn("Could not cleanup audit_logs by max", zap.Error(err))
		}
		if _, err := s.db.ExecContext(ctx, getQry("authentication_logs", maxDBAuthenticationLogs)); err != nil {
			zap.L().Warn("Could not cleanup authentication_logs by max", zap.Error(err))
		}
	}

	if _, err := s.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		zap.L().Error("Could not checkpoint", zap.Error(err))
	}

	return nil
}
