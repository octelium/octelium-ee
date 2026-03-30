// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"time"

	"github.com/google/uuid"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func now() k8smetav1.Time {
	return k8smetav1.Time{
		time.Now(),
	}
}

func getFakeK8sC() kubernetes.Interface {
	k8sC := fakek8s.NewSimpleClientset()
	k8sC.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		switch {
		case action.Matches("create", "secrets"):
			obj := action.(k8stesting.UpdateAction).GetObject().(*k8scorev1.Secret)
			obj.CreationTimestamp = now()
			obj.UID = types.UID(uuid.New().String())
		case action.Matches("create", "pods"):
			obj := action.(k8stesting.UpdateAction).GetObject().(*k8scorev1.Pod)
			obj.CreationTimestamp = now()
			obj.UID = types.UID(uuid.New().String())
		case action.Matches("create", "nodes"):
			obj := action.(k8stesting.UpdateAction).GetObject().(*k8scorev1.Node)
			obj.CreationTimestamp = now()
			obj.UID = types.UID(uuid.New().String())
		}
		return
	})

	return k8sC
}
