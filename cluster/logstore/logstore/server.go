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
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/spiffec"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
)

const tstAddr = "localhost:32123"

const (
	maxDBConns            = 8
	maxConcurrentQueries  = 4
	queryTimeout          = 60 * time.Second
	otlpLogsServicePrefix = "/opentelemetry.proto.collector.logs.v1.LogsService/"
)

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain   string
	genCache        *cache.Cache
	db              *sql.DB
	cleanupDuration time.Duration
	querySem        chan struct{}
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC:       octeliumC,
		genCache:        cache.New(cache.NoExpiration, 1*time.Minute),
		cleanupDuration: 30 * 24 * time.Hour,
		querySem:        make(chan struct{}, maxConcurrentQueries),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	ret.db, err = sql.Open("duckdb", ovutils.GetDuckDBDSNWithOpts(&ovutils.DuckDBOpts{}))
	if err != nil {
		return nil, err
	}

	ret.db.SetMaxOpenConns(maxDBConns)
	ret.db.SetMaxIdleConns(maxDBConns)
	ret.db.SetConnMaxLifetime(0)

	return ret, nil
}

func (s *Server) limitQueries(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

	if strings.HasPrefix(info.FullMethod, otlpLogsServicePrefix) {
		return handler(ctx, req)
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	select {
	case s.querySem <- struct{}{}:
		defer func() { <-s.querySem }()
	case <-ctx.Done():
		return nil, status.Error(codes.ResourceExhausted, "LogStore is busy")
	}

	return handler(ctx, req)
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
	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		return err
	}

	grpcSrv := grpc.NewServer(
		cred,
		grpc.ReadBufferSize(32*1024),
		grpc.MaxConcurrentStreams(1000000),
		grpc.ChainUnaryInterceptor(s.limitQueries),
	)

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

	vllmv1.RegisterLLMServiceServer(grpcSrv, &srvLLM{
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
var maxDBComponentLogsDebug = 10_000

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
		type maxCleanup struct {
			table string
			where string
			limit int
		}

		for _, c := range []maxCleanup{
			{table: "access_logs", limit: maxDBAccessLogs},
			{table: "component_logs", limit: maxDBComponentLogs},
			{
				table: "component_logs",
				where: fmt.Sprintf(`(rsc->'entry'->>'level') = '%s'`,
					corev1.ComponentLog_Entry_DEBUG.String()),
				limit: maxDBComponentLogsDebug,
			},
			{table: "audit_logs", limit: maxDBAuditLogs},
			{table: "authentication_logs", limit: maxDBAuthenticationLogs},
		} {
			if err := s.cleanupByMaxCount(ctx, c.table, c.where, c.limit); err != nil {
				zap.L().Warn("Could not cleanup by max",
					zap.String("table", c.table), zap.String("where", c.where), zap.Error(err))
			}
		}
	}

	if _, err := s.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		zap.L().Error("Could not checkpoint", zap.Error(err))
	}

	return nil
}

func (s *Server) cleanupByMaxCount(ctx context.Context, table, where string, limit int) error {
	whereClause := ""
	if where != "" {
		whereClause = fmt.Sprintf(" WHERE %s", where)
	}

	var count int
	if err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM %s%s`, table, whereClause)).Scan(&count); err != nil {
		return err
	}

	if count <= limit {
		return nil
	}

	var cutoff string
	if err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT rsc->'metadata'->>'createdAt' FROM %s%s ORDER BY (rsc->'metadata'->>'createdAt') DESC LIMIT 1 OFFSET %d`,
			table, whereClause, limit-1)).Scan(&cutoff); err != nil {
		return err
	}

	deleteWhere := `(rsc->'metadata'->>'createdAt') < $1`
	if where != "" {
		deleteWhere = fmt.Sprintf(`%s AND %s`, where, deleteWhere)
	}

	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE %s`, table, deleteWhere), cutoff)

	return err
}
