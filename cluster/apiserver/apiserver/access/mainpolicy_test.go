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

func TestMainPolicy(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	mainSrv := NewServerMain(tst.C.OcteliumC)

	pol, err := mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	polG, err := mainSrv.GetPolicy(ctx, &metav1.GetOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, pol.Metadata.Uid, polG.Metadata.Uid)

	polList, err := mainSrv.ListPolicy(ctx, &accessv1.ListPolicyOptions{})
	assert.Nil(t, err)
	assert.True(t, len(polList.Items) > 0)

	found := false
	for _, item := range polList.Items {
		if item.Metadata.Uid == pol.Metadata.Uid {
			found = true
		}
	}
	assert.True(t, found)

	newName := utilrand.GetRandomStringCanonical(6)
	polG.Spec.Rules[0].Name = newName
	polU, err := mainSrv.UpdatePolicy(ctx, polG)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, newName, polU.Spec.Rules[0].Name)

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: pol.Metadata.Name,
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	dupName := utilrand.GetRandomStringCanonical(6)
	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   dupName,
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
				{
					Name:   dupName,
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_REVIEW,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_DENY,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.DeletePolicy(ctx, &metav1.DeleteOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.GetPolicy(ctx, &metav1.GetOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.NotNil(t, err)
}
