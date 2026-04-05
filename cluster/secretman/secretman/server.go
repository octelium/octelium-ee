// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"context"
	"database/sql"
	"net"
	"sync"

	_ "github.com/lib/pq"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/watchers"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type server struct {
	sync.Mutex

	csecretmanv1.MainServiceServer
	octeliumC octeliumc.ClientInterface
	// k8sC      kubernetes.Interface

	db *sql.DB

	deks struct {
		sync.RWMutex
		dekMap map[string]*dek
	}

	keks struct {
		sync.RWMutex
		kekMap map[string]stores.Store
	}
}

func newServer(ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	// k8sC kubernetes.Interface,
	db *sql.DB) (*server, error) {

	ret := &server{
		octeliumC: octeliumC,
		// k8sC:      k8sC,
		db: db,
	}

	ret.deks.dekMap = make(map[string]*dek)
	ret.keks.kekMap = make(map[string]stores.Store)

	return ret, nil
}

func (s *server) run(ctx context.Context) error {

	srv := grpc.NewServer()
	csecretmanv1.RegisterMainServiceServer(srv, s)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	go func() {
		zap.S().Debug("running gRPC server.")
		if err := srv.Serve(lis); err != nil {
			zap.L().Info("gRPC server closed", zap.Error(err))
		}
	}()

	return nil
}

func Run(ctx context.Context) error {
	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	db, err := postgresutils.NewDB()
	if err != nil {
		return err
	}

	if err := migrations.Migrate(ctx, db); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	s, err := newServer(ctx, octeliumC, db)
	if err != nil {
		return err
	}

	if err := s.initRootDEK(ctx); err != nil {
		return err
	}

	if err := s.setDEKMap(ctx); err != nil {
		return err
	}

	if err := s.setStaleSecretResources(ctx); err != nil {
		return err
	}

	if err := watchers.NewEnterpriseV1(s.octeliumC).SecretStore(ctx, nil, nil, s.onSecretManUpdate, nil); err != nil {
		return err
	}

	if err := s.run(ctx); err != nil {
		return err
	}
	healthcheck.Run(vutils.HealthCheckPortMain)

	zap.L().Info("SecretManager is now running")

	<-ctx.Done()

	return nil
}

func (s *server) initRootDEK(ctx context.Context) error {

	deks, err := s.doListDEK(ctx)
	if err != nil {
		return err
	}
	if len(deks) > 0 {
		zap.L().Debug("Found DEKs already. Nothing to be done", zap.Int("dekLen", len(deks)))
		return nil
	}

	zap.L().Debug("Could not find any DEK. Creating one...")

	if err := s.doCreateDEK(ctx); err != nil {
		return err
	}

	zap.L().Debug("Successfully created DEK")

	return nil
}

func (s *server) setDEKMap(ctx context.Context) error {
	deks, err := s.doListDEK(ctx)
	if err != nil {
		return err
	}

	for _, dek := range deks {
		zap.L().Debug("Setting dek", zap.String("uid", dek.uid))
		s.deks.dekMap[dek.uid] = dek
	}

	return nil
}
