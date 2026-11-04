package genesis

import (
	"testing"

	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestJoinRejectsPasswordReferencesBeforeMutation(t *testing.T) {
	for _, storage := range []string{"postgres", "redis"} {
		t.Run(storage, func(t *testing.T) {
			resetPasswordEnvironment(t)
			ldflags.TestMode = "true"
			t.Setenv("OCTELIUM_DOMAIN", "example.com")
			t.Setenv("OCTELIUM_REGION_NAME", "")
			bs := getTestJoinBootstrap()
			setPasswordReference(bs, storage, "credentials", "default", "missing")
			c := fake.NewSimpleClientset(
				getTestJoinSecret(bs, getTestJoinRegion(t, "west", 1), "example.com"),
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"password": []byte("wrong-fallback")}},
			)
			g := &Genesis{k8sC: c}
			require.ErrorContains(t, g.RunJoin(t.Context(), &JoinOpts{CoreVersion: "9.9.9"}), "missing or empty")
			assertPasswordResolutionReadOnly(t, c)
		})
	}
}
