// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"fmt"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/require"
)

func newTestServiceName(t *testing.T, octeliumC octeliumc.ClientInterface) string {
	return newTestService(t, octeliumC, nil).Metadata.Name
}

func newTestService(t *testing.T, octeliumC octeliumc.ClientInterface, md *metav1.Metadata) *corev1.Service {
	if md == nil {
		md = &metav1.Metadata{}
	}
	md.Name = fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8))

	svc, err := octeliumC.CoreC().CreateService(context.Background(), &corev1.Service{
		Metadata: md,
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{},
	})
	require.NoError(t, err)
	return svc
}
