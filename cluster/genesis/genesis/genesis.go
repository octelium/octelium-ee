// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"database/sql"

	nadclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	_ "github.com/lib/pq"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	iocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Genesis struct {
	pod           *k8scorev1.Pod
	podNamespace  string
	k8sC          kubernetes.Interface
	nadC          nadclientset.Interface
	octeliumC     octeliumc.ClientInterface
	octeliumCInit iocteliumc.ClientInterface
	db            *sql.DB
}

func NewGenesis() (*Genesis, error) {
	ret := &Genesis{}

	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	k8sClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	nadC, err := nadclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	ret.k8sC = k8sClientSet
	ret.nadC = nadC

	if err := ret.setPodNamespace(); err != nil {
		return nil, err
	}
	return ret, nil
}
