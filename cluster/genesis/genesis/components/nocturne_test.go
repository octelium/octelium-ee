package components

import (
	"testing"

	"github.com/octelium/octelium/apis/main/corev1"
	ometav1 "github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNocturneRuntimeRBAC(t *testing.T) {
	role := getNocturneRuntimeRole()
	assert.Equal(t, "octeliumee-nocturne-runtime", role.Name)
	assert.Equal(t, "octelium", role.Namespace)
	assert.Equal(t, []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"apps"},
			Resources:     []string{"deployments"},
			ResourceNames: []string{"octeliumee-collector", "octelium-octovigil", "octelium-ingress-dataplane"},
			Verbs:         []string{"get", "update"},
		},
	}, role.Rules)

	binding := getNocturneRuntimeRoleBinding()
	assert.Equal(t, "octelium", binding.Namespace)
	assert.Equal(t, rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: role.Name}, binding.RoleRef)
	assert.Equal(t, []rbacv1.Subject{{Kind: "ServiceAccount", Name: "octeliumee-nocturne-runtime", Namespace: "octelium"}}, binding.Subjects)
}

func TestCreateNocturne(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	opts := &CommonOpts{}
	opts.K8sC = client
	opts.ClusterConfig = &corev1.ClusterConfig{Metadata: &ometav1.Metadata{Name: "default"}, Spec: &corev1.ClusterConfig_Spec{}, Status: &corev1.ClusterConfig_Status{Domain: "example.com"}}
	opts.Region = &corev1.Region{Metadata: &ometav1.Metadata{Name: "default", Uid: "region-uid"}, Spec: &corev1.Region_Spec{}, Status: &corev1.Region_Status{}}

	for range 2 {
		require.NoError(t, CreateNocturne(ctx, opts))
	}

	deployment, err := client.AppsV1().Deployments(ns).Get(ctx, "octeliumee-nocturne", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "octeliumee-nocturne-runtime", deployment.Spec.Template.Spec.ServiceAccountName)

	_, err = client.CoreV1().ServiceAccounts(ns).Get(ctx, "octeliumee-nocturne-runtime", metav1.GetOptions{})
	require.NoError(t, err)

	role, err := client.RbacV1().Roles(ns).Get(ctx, "octeliumee-nocturne-runtime", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, getNocturneRuntimeRole().Rules, role.Rules)

	roleBinding, err := client.RbacV1().RoleBindings(ns).Get(ctx, "octeliumee-nocturne-runtime", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, getNocturneRuntimeRoleBinding().Subjects, roleBinding.Subjects)

	_, err = client.CoreV1().ServiceAccounts(ns).Get(ctx, "octeliumee-nocturne", metav1.GetOptions{})
	require.NoError(t, err)
	clusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, "octeliumee-nocturne", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []rbacv1.Subject{{Kind: "ServiceAccount", Name: "octeliumee-nocturne", Namespace: "octelium"}}, clusterRoleBinding.Subjects)

	assert.Equal(t, "octeliumee-nocturne", getClusterManDeployment(opts).Spec.Template.Spec.ServiceAccountName)
}
