// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumc

import (
	"context"
	"fmt"
	"os"

	"github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/raccessv1"
	"github.com/octelium/octelium/apis/rsc/rcachev1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/apis/rsc/rratelimitv1"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"google.golang.org/grpc"
)

type Client struct {
	coreC       rcorev1.ResourceServiceClient
	cacheC      rcachev1.MainServiceClient
	rateLimitC  rratelimitv1.MainServiceClient
	enterpriseC renterprisev1.ResourceServiceClient

	coreV1UtilsC       *coreV1UtilsC
	enterpriseV1UtilsC *enterpriseV1UtilsC

	accessC raccessv1.ResourceServiceClient
}

type CoreV1Utils interface {
	GetClusterConfig(ctx context.Context) (*corev1.ClusterConfig, error)
}

type EnterpriseV1Utils interface {
	GetClusterConfig(ctx context.Context) (*enterprisev1.ClusterConfig, error)
}

type Opts struct {
	Addr string
}

func DefaultAddr() string {
	return fmt.Sprintf("%s.%s.svc:8080",
		components.OcteliumEnterpriseComponent(components.RscServer),
		vutils.K8sNS,
	)
}

func NewClient(ctx context.Context, opts *Opts) (*Client, error) {

	if opts == nil {
		opts = &Opts{}
	}

	var host string

	if opts.Addr != "" {
		host = opts.Addr
	} else if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSERVER_PORT"))

	} else {
		host = DefaultAddr()
	}

	tOpts, err := octeliumc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}

	grpcConn, err := grpc.NewClient(host, tOpts...)
	if err != nil {
		return nil, err
	}

	ret := &Client{
		coreC:       rcorev1.NewResourceServiceClient(grpcConn),
		cacheC:      rcachev1.NewMainServiceClient(grpcConn),
		enterpriseC: renterprisev1.NewResourceServiceClient(grpcConn),
		accessC:     raccessv1.NewResourceServiceClient(grpcConn),

		coreV1UtilsC: &coreV1UtilsC{},

		enterpriseV1UtilsC: &enterpriseV1UtilsC{},
	}

	ret.coreV1UtilsC.c = ret.coreC
	ret.enterpriseV1UtilsC.c = ret.enterpriseC

	return ret, nil
}

func (c *Client) CoreC() rcorev1.ResourceServiceClient {
	return c.coreC
}

func (c *Client) CacheC() rcachev1.MainServiceClient {
	return c.cacheC
}

func (c *Client) RateLimitC() rratelimitv1.MainServiceClient {
	return c.rateLimitC
}

func (c *Client) EnterpriseC() renterprisev1.ResourceServiceClient {
	return c.enterpriseC
}

func (c *Client) AccessC() raccessv1.ResourceServiceClient {
	return c.accessC
}

func (c *Client) CoreV1Utils() octeliumc.CoreV1Utils {
	return c.coreV1UtilsC
}

func (c *Client) EnterpriseV1Utils() EnterpriseV1Utils {
	return c.enterpriseV1UtilsC
}

type ClientInterface interface {
	octeliumc.ClientInterface

	EnterpriseC() renterprisev1.ResourceServiceClient
	EnterpriseV1Utils() EnterpriseV1Utils
	AccessC() raccessv1.ResourceServiceClient
}

type coreV1UtilsC struct {
	c rcorev1.ResourceServiceClient
}

func (c *coreV1UtilsC) GetClusterConfig(ctx context.Context) (*corev1.ClusterConfig, error) {
	return c.c.GetClusterConfig(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
}

type enterpriseV1UtilsC struct {
	c renterprisev1.ResourceServiceClient
}

func (c *enterpriseV1UtilsC) GetClusterConfig(ctx context.Context) (*enterprisev1.ClusterConfig, error) {
	return c.c.GetClusterConfig(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
}
