package genesis

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestJoinTargetRecordingPreservesInstallation(t *testing.T) {
	for _, failure := range []string{"none", "conflict", "before-commit", "after-commit"} {
		t.Run(failure, func(t *testing.T) {
			g, cc, region, c := newPackageTarget(t)
			updates := 0
			c.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				if updates > 1 || failure == "none" {
					return false, nil, nil
				}
				if failure == "conflict" {
					return true, nil, k8serr.NewConflict(action.GetResource().GroupResource(), "octelium-installation", io.EOF)
				}
				if failure == "after-commit" {
					require.NoError(t, c.Tracker().Update(action.GetResource(), action.(k8stesting.UpdateAction).GetObject(), action.GetNamespace()))
				}
				return true, nil, io.ErrUnexpectedEOF
			})
			require.NoError(t, g.recordRegionTarget(t.Context(), cc, region))
			cm, err := c.CoreV1().ConfigMaps(vutils.K8sNS).Get(t.Context(), "octelium-installation", metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"custom": "preserved"}, cm.Data)
			assert.Equal(t, packageTargetAnnotations(cc, region), cm.Annotations)
			if failure == "after-commit" {
				assert.Equal(t, 1, updates)
			}
			c.ClearActions()
			require.NoError(t, g.recordRegionTarget(t.Context(), cc, region))
			assertPackageTargetReadOnly(t, c)
		})
	}
	g, cc, region, c := newPackageTarget(t)
	cc.Metadata.Uid = ""
	require.Error(t, g.recordRegionTarget(t.Context(), cc, region))
	assertPackageTargetReadOnly(t, c)
}

func TestJoinStorageNeverOverwritesExistingCredentials(t *testing.T) {
	for _, name := range []string{"octelium-postgres", "octelium-redis"} {
		for _, failure := range []string{"matching", "different", "lost-acknowledgement"} {
			t.Run(name+"/"+failure, func(t *testing.T) {
				c := fake.NewSimpleClientset()
				g := &Genesis{k8sC: c}
				if failure != "lost-acknowledgement" {
					require.NoError(t, g.initJoinStorage(t.Context(), getTestJoinBootstrap()))
					secret, err := c.CoreV1().Secrets(vutils.K8sNS).Get(t.Context(), name, metav1.GetOptions{})
					require.NoError(t, err)
					secret.Data["custom"] = []byte("preserved")
					if failure == "different" {
						secret.Data["host"] = []byte("existing-storage")
					}
					require.NoError(t, c.Tracker().Update(corev1.SchemeGroupVersion.WithResource("secrets"), secret, vutils.K8sNS))
					c.ClearActions()
				} else {
					failed := false
					c.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
						secret := action.(k8stesting.CreateAction).GetObject().(*corev1.Secret)
						if failed || secret.Name != name {
							return false, nil, nil
						}
						failed = true
						require.NoError(t, c.Tracker().Create(action.GetResource(), secret, action.GetNamespace()))
						return true, nil, io.ErrUnexpectedEOF
					})
				}
				err := g.initJoinStorage(t.Context(), getTestJoinBootstrap())
				if failure == "different" {
					require.ErrorContains(t, err, "Refusing to replace")
				} else {
					require.NoError(t, err)
				}
				secret, err := c.CoreV1().Secrets(vutils.K8sNS).Get(t.Context(), name, metav1.GetOptions{})
				require.NoError(t, err)
				if failure == "different" {
					assert.Equal(t, "existing-storage", string(secret.Data["host"]))
				}
				if failure != "lost-acknowledgement" {
					assert.Equal(t, "preserved", string(secret.Data["custom"]))
				}
				for _, action := range c.Actions() {
					assert.NotContains(t, []string{"update", "delete", "patch"}, action.GetVerb())
				}
			})
		}
	}
}

func TestJoinPreservesRegisteredNodes(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, state := range []string{"registered", "new", "already-tainted", "registration-conflict", "lost-acknowledgement"} {
		t.Run(state, func(t *testing.T) {
			node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "gateway", Labels: map[string]string{vutils.NodeLabelDataPlane: "", "custom": "preserved"}}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: "custom", Effect: corev1.TaintEffectNoSchedule}}}}
			if state == "registered" {
				node.Labels[vutils.NodeLabelGatewayRegistered] = "true"
			}
			if state == "already-tainted" {
				node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{Key: vutils.NodeTaintGatewayInit, Value: "true", Effect: corev1.TaintEffectNoSchedule})
			}
			c := fake.NewSimpleClientset(node)
			g := &Genesis{k8sC: c}
			updates := 0
			c.PrependReactor("update", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				if updates > 1 {
					return false, nil, nil
				}
				if state == "registration-conflict" {
					registered := node.DeepCopy()
					registered.Labels[vutils.NodeLabelGatewayRegistered] = "true"
					require.NoError(t, c.Tracker().Update(action.GetResource(), registered, ""))
					return true, nil, k8serr.NewConflict(action.GetResource().GroupResource(), node.Name, io.EOF)
				}
				if state == "lost-acknowledgement" {
					require.NoError(t, c.Tracker().Update(action.GetResource(), action.(k8stesting.UpdateAction).GetObject(), ""))
					return true, nil, io.ErrUnexpectedEOF
				}
				return false, nil, nil
			})
			require.NoError(t, g.setDataPlaneNodeTaints(t.Context()))
			updated, err := c.CoreV1().Nodes().Get(t.Context(), node.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, "preserved", updated.Labels["custom"])
			assert.Equal(t, node.Spec.Taints[0], updated.Spec.Taints[0])
			if state == "registered" || state == "registration-conflict" {
				assert.Equal(t, "true", updated.Labels[vutils.NodeLabelGatewayRegistered])
				assert.Equal(t, node.Spec.Taints, updated.Spec.Taints)
			} else {
				assert.Len(t, updated.Spec.Taints, 2)
			}
			if state == "registered" || state == "already-tainted" {
				assert.Zero(t, updates)
			}
			if state == "registration-conflict" || state == "lost-acknowledgement" {
				assert.Equal(t, 1, updates)
			}
		})
	}
}

func TestJoinCancelsBeforeMutation(t *testing.T) {
	g := &Genesis{k8sC: fake.NewSimpleClientset()}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, g.RunInit(ctx, nil), context.Canceled)
	require.ErrorIs(t, g.RunUpgrade(ctx, nil), context.Canceled)
	require.ErrorIs(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "2.0.0"}), context.Canceled)
	assert.Empty(t, g.k8sC.(*fake.Clientset).Actions())
	ctx, cancel = context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	g.k8sC.(*fake.Clientset).PrependReactor("create", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, io.EOF })
	g.k8sC.(*fake.Clientset).PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, io.EOF })
	require.ErrorIs(t, g.initJoinStorage(ctx, getTestJoinBootstrap()), context.DeadlineExceeded)
}
