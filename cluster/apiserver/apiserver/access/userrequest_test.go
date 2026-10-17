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
	"sort"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestUserRequest(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)

	mainSrv := NewServerMain(tst.C.OcteliumC)
	srv, err := NewServerUser(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

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

	cat2, err := mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
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

	req, err := srv.CreateRequest(usr.Ctx(), &accessv1.Request{
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
			Justification: utilrand.GetRandomString(20),
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.NotNil(t, req.Status.State)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, req.Status.State.Status)
	assert.Equal(t, usr.Usr.Metadata.Uid, req.Status.UserRef.Uid)
	assert.NotNil(t, req.Spec.Subject)
	assert.Equal(t, usr.Usr.Metadata.Uid, req.Spec.Subject.GetUserRef().Uid)

	reqG, err := srv.GetRequest(usr.Ctx(), &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, req.Metadata.Uid, reqG.Metadata.Uid)

	reqList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(reqList.Items))
	assert.Equal(t, req.Metadata.Uid, reqList.Items[0].Metadata.Uid)

	reqG.Spec.Urgency = accessv1.Request_Spec_HIGH
	reqG.Spec.Justification = utilrand.GetRandomString(24)
	reqU, err := srv.UpdateRequest(usr.Ctx(), reqG)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessv1.Request_Spec_HIGH, reqU.Spec.Urgency)

	reqChangeResource := pbutils.Clone(reqU).(*accessv1.Request)
	reqChangeResource.Spec.Resource = &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_Catalog_{
			Catalog: &accessv1.Request_Spec_Resource_Catalog{
				CatalogRef: umetav1.GetObjectReference(cat2),
			},
		},
	}
	_, err = srv.UpdateRequest(usr.Ctx(), reqChangeResource)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	usr2, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	_, err = srv.GetRequest(usr2.Ctx(), &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.NotNil(t, err)

	_, err = srv.CancelRequest(usr.Ctx(), &accessv1.CancelRequestRequest{
		RequestRef: umetav1.GetObjectReference(reqU),
	})
	assert.Nil(t, err, "%+v", err)

	reqC, err := srv.GetRequest(usr.Ctx(), &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err)
	assert.Equal(t, accessv1.Request_Status_State_CANCELLED, reqC.Status.State.Status)

	_, err = srv.UpdateRequest(usr.Ctx(), reqC)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))
}

func TestUserRequestListFilters(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)

	mainSrv := NewServerMain(tst.C.OcteliumC)
	srv, err := NewServerUser(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	newCatalog := func() *accessv1.Catalog {
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
		return cat
	}

	newRequest := func(name string, cat *accessv1.Catalog, urgency accessv1.Request_Spec_Urgency) *accessv1.Request {
		req, err := srv.CreateRequest(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: name,
			},
			Spec: &accessv1.Request_Spec{
				Urgency: urgency,
				Resource: &accessv1.Request_Spec_Resource{
					Type: &accessv1.Request_Spec_Resource_Catalog_{
						Catalog: &accessv1.Request_Spec_Resource_Catalog{
							CatalogRef: umetav1.GetObjectReference(cat),
						},
					},
				},
				Justification: utilrand.GetRandomString(20),
			},
		})
		assert.Nil(t, err, "%+v", err)
		return req
	}

	catA := newCatalog()
	catB := newCatalog()

	reqA := newRequest("a-"+utilrand.GetRandomStringCanonical(6), catA, accessv1.Request_Spec_HIGH)
	reqB := newRequest("b-"+utilrand.GetRandomStringCanonical(6), catB, accessv1.Request_Spec_LOW)
	reqC := newRequest("c-"+utilrand.GetRandomStringCanonical(6), catA, accessv1.Request_Spec_LOW)

	_, err = srv.CancelRequest(usr.Ctx(), &accessv1.CancelRequestRequest{
		RequestRef: umetav1.GetObjectReference(reqC),
	})
	assert.Nil(t, err, "%+v", err)

	uids := func(itemList *accessv1.RequestList) []string {
		var ret []string
		for _, itm := range itemList.Items {
			ret = append(ret, itm.Metadata.Uid)
		}
		return ret
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.ElementsMatch(t, []string{reqA.Metadata.Uid, reqB.Metadata.Uid, reqC.Metadata.Uid}, uids(itemList))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			Common: &metav1.CommonListOptions{
				OrderBy: &metav1.CommonListOptions_OrderBy{
					Type: metav1.CommonListOptions_OrderBy_NAME,
					Mode: metav1.CommonListOptions_OrderBy_ASC,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 3, len(itemList.Items))
		assert.True(t, sort.SliceIsSorted(itemList.Items, func(i, j int) bool {
			return itemList.Items[i].Metadata.Name < itemList.Items[j].Metadata.Name
		}))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			Common: &metav1.CommonListOptions{
				OrderBy: &metav1.CommonListOptions_OrderBy{
					Type: metav1.CommonListOptions_OrderBy_NAME,
					Mode: metav1.CommonListOptions_OrderBy_DESC,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 3, len(itemList.Items))
		assert.True(t, sort.SliceIsSorted(itemList.Items, func(i, j int) bool {
			return itemList.Items[i].Metadata.Name > itemList.Items[j].Metadata.Name
		}))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			State: accessv1.Request_Status_State_PENDING,
		})
		assert.Nil(t, err, "%+v", err)
		assert.ElementsMatch(t, []string{reqA.Metadata.Uid, reqB.Metadata.Uid}, uids(itemList))
		assert.Equal(t, uint32(2), itemList.ListResponseMeta.TotalCount)
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			State: accessv1.Request_Status_State_CANCELLED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{reqC.Metadata.Uid}, uids(itemList))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			Urgency: accessv1.Request_Spec_LOW,
		})
		assert.Nil(t, err, "%+v", err)
		assert.ElementsMatch(t, []string{reqB.Metadata.Uid, reqC.Metadata.Uid}, uids(itemList))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			CatalogRef: umetav1.GetObjectReference(catA),
		})
		assert.Nil(t, err, "%+v", err)
		assert.ElementsMatch(t, []string{reqA.Metadata.Uid, reqC.Metadata.Uid}, uids(itemList))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			CatalogRef: &metav1.ObjectReference{
				Name: catB.Metadata.Name,
			},
			State: accessv1.Request_Status_State_PENDING,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{reqB.Metadata.Uid}, uids(itemList))
	}

	{
		itemList, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			ServiceRef: &metav1.ObjectReference{
				Name: utilrand.GetRandomStringCanonical(8),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 0, len(itemList.Items))
	}

	{
		_, err := srv.ListRequest(usr.Ctx(), &accessv1.ListUserRequestOptions{
			CatalogRef: &metav1.ObjectReference{
				Name: "INVALID name",
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		usr2, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
		assert.Nil(t, err)

		itemList, err := srv.ListRequest(usr2.Ctx(), &accessv1.ListUserRequestOptions{
			State: accessv1.Request_Status_State_PENDING,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 0, len(itemList.Items))
	}
}
