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
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestMainCatalog(t *testing.T) {
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

	catG, err := mainSrv.GetCatalog(ctx, &metav1.GetOptions{
		Uid: cat.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, cat.Metadata.Uid, catG.Metadata.Uid)

	catList, err := mainSrv.ListCatalog(ctx, &accessv1.ListCatalogOptions{})
	assert.Nil(t, err)
	assert.True(t, len(catList.Items) > 0)

	found := false
	for _, item := range catList.Items {
		if item.Metadata.Uid == cat.Metadata.Uid {
			found = true
		}
	}
	assert.True(t, found)

	newServices := []string{utilrand.GetRandomStringCanonical(6)}
	catG.Spec.ResourceCollection.Service.Services = newServices
	catU, err := mainSrv.UpdateCatalog(ctx, catG)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, newServices, catU.Spec.ResourceCollection.Service.Services)

	_, err = mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: cat.Metadata.Name,
		},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services: []string{utilrand.GetRandomStringCanonical(6)},
				},
			},
		},
	})
	assert.NotNil(t, err)

	_, err = mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Catalog_Spec{},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.DeleteCatalog(ctx, &metav1.DeleteOptions{
		Uid: cat.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.GetCatalog(ctx, &metav1.GetOptions{
		Uid: cat.Metadata.Uid,
	})
	assert.NotNil(t, err)
}
