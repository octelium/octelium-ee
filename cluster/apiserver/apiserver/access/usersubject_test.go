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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestUserSubjectUser(t *testing.T) {
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

	srv, err := NewServerUser(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	subject, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	subject.Usr.Metadata.DisplayName = utilrand.GetRandomString(12)
	subject.Usr.Metadata.PicURL = "https://example.com/pic.png"
	subject.Usr.Spec.Type = corev1.User_Spec_HUMAN
	subject.Usr.Spec.Email = utilrand.GetRandomStringCanonical(8) + "@example.com"
	subject.Usr, err = tst.C.OcteliumC.CoreC().UpdateUser(ctx, subject.Usr)
	assert.Nil(t, err, "%+v", err)

	{
		item, err := srv.GetSubjectUser(usr.Ctx(), &accessv1.GetSubjectUserRequest{
			UserRef: umetav1.GetObjectReference(subject.Usr),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, subject.Usr.Metadata.Uid, item.UserRef.Uid)
		assert.Equal(t, subject.Usr.Metadata.Name, item.UserRef.Name)
		assert.Equal(t, subject.Usr.Metadata.DisplayName, item.DisplayName)
		assert.Equal(t, subject.Usr.Metadata.PicURL, item.PicURL)
		assert.Equal(t, subject.Usr.Spec.Email, item.Email)
		assert.Equal(t, corev1.User_Spec_HUMAN, item.Type)
	}

	{
		_, err := srv.GetSubjectUser(usr.Ctx(), &accessv1.GetSubjectUserRequest{})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetSubjectUser(usr.Ctx(), &accessv1.GetSubjectUserRequest{
			UserRef: &metav1.ObjectReference{
				Name: utilrand.GetRandomStringCanonical(8),
			},
		})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		hidden, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
		assert.Nil(t, err)

		hidden.Usr.Metadata.IsSystemHidden = true
		hidden.Usr, err = tst.C.OcteliumC.CoreC().UpdateUser(ctx, hidden.Usr)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetSubjectUser(usr.Ctx(), &accessv1.GetSubjectUserRequest{
			UserRef: umetav1.GetObjectReference(hidden.Usr),
		})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	for _, query := range []string{"", " ", "a", " a "} {
		_, err := srv.ListSubjectUser(usr.Ctx(), &accessv1.ListSubjectUserOptions{
			Query: query,
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "query %q: %+v", query, err)
	}
}

func TestUserCreateRequestForSubject(t *testing.T) {
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

	srv, err := NewServerUser(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	subject, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

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

	newSpec := func(subjectRef *metav1.ObjectReference) *accessv1.Request_Spec {
		ret := &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef: umetav1.GetObjectReference(cat),
					},
				},
			},
			Justification: utilrand.GetRandomString(20),
		}

		if subjectRef != nil {
			ret.Subject = &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{
					UserRef: subjectRef,
				},
			}
		}

		return ret
	}

	{
		req, err := srv.CreateRequest(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: newSpec(umetav1.GetObjectReference(subject.Usr)),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, usr.Usr.Metadata.Uid, req.Spec.Subject.GetUserRef().Uid)
		assert.Equal(t, usr.Usr.Metadata.Uid, req.Status.UserRef.Uid)
	}

	{
		req, err := srv.CreateRequest(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: newSpec(nil),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, usr.Usr.Metadata.Uid, req.Spec.Subject.GetUserRef().Uid)
	}

	{
		req, err := srv.CreateRequestForSubject(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: newSpec(umetav1.GetObjectReference(subject.Usr)),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, subject.Usr.Metadata.Uid, req.Spec.Subject.GetUserRef().Uid)
		assert.Equal(t, usr.Usr.Metadata.Uid, req.Status.UserRef.Uid)
	}

	{
		_, err := srv.CreateRequestForSubject(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: newSpec(nil),
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateRequestForSubject(usr.Ctx(), &accessv1.Request{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: newSpec(&metav1.ObjectReference{
				Name: utilrand.GetRandomStringCanonical(8),
			}),
		})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}
