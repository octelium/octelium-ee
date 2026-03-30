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

	_ "github.com/lib/pq"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	iocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Genesis struct {
	k8sC          kubernetes.Interface
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

	ret.k8sC = k8sClientSet

	return ret, nil
}
