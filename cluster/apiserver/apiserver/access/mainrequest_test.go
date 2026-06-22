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
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestMainRequest(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	mainSrv := NewServerMain(tst.C.OcteliumC)

	cat, err := mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services: []string{utilrand.GetRandomStringCanonical(6)},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	req, err := tst.C.OcteliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef: umetav1.GetObjectReference(cat),
					},
				},
			},
		},
		Status: &accessv1.Request_Status{
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	reqG, err := mainSrv.GetRequest(ctx, &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, req.Metadata.Uid, reqG.Metadata.Uid)

	reqList, err := mainSrv.ListRequest(ctx, &accessv1.ListRequestOptions{})
	assert.Nil(t, err)
	assert.True(t, len(reqList.Items) > 0)

	found := false
	for _, item := range reqList.Items {
		if item.Metadata.Uid == req.Metadata.Uid {
			found = true
		}
	}
	assert.True(t, found)

	_, err = mainSrv.DeleteRequest(ctx, &metav1.DeleteOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.GetRequest(ctx, &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.NotNil(t, err)
}
