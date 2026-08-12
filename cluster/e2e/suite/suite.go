// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"os"
	"strconv"
	"strings"

	"github.com/octelium/octelium/cluster/e2e/suite"
)

func Phases() []suite.Phase {
	return []suite.Phase{
		{Name: "EnterpriseSDK", Run: testEnterpriseSDK},
		{Name: "EnterpriseTopology", Run: testEnterpriseTopology},

		{Name: "SCIM", Run: testSCIM},
		{Name: "SCIMProtocol", Run: testSCIMProtocol},
		{Name: "SCIMDisabledProvider", Run: testSCIMDisabledProvider},
		{Name: "SCIMToDataPlane", Run: testSCIMToDataPlane},
		{Name: "KeycloakDirectoryProvider", Run: testKeycloakDirectoryProvider},
		{Name: "KeycloakBadCredentials", Run: testKeycloakBadCredentials},
		{Name: "DirectoryProviderIsolation", Run: testDirectoryProviderIsolation},

		{Name: "AccessAutoApprove", Run: testAccessAutoApprove},
		{Name: "AccessSingleStepReview", Run: testAccessSingleStepReview},
		{Name: "AccessMultiStepQuorum", Run: testAccessMultiStepQuorum},
		{Name: "AccessRejectCancelRevoke", Run: testAccessRejectCancelRevoke},
		{Name: "AccessStepTimeout", Run: testAccessStepTimeout},
		{Name: "AccessStepTimeoutRejects", Run: testAccessStepTimeoutRejects},
		{Name: "AccessExpiry", Run: testAccessExpiry},
		{Name: "AccessAPIScoping", Run: testAccessAPIScoping},
		{Name: "AccessAuditTrail", Run: testAccessAuditTrail},
		{Name: "AccessPortal", Run: testAccessPortal},
		{Name: "ConsolePortal", Run: testConsolePortal},

		{Name: "PolicyPortalParity", Run: testPolicyPortalParity},
		{Name: "PolicyPortalUnderChurn", Run: testPolicyPortalUnderChurn},
		{Name: "PolicyPortalWithPolicyTrigger", Run: testPolicyPortalWithPolicyTrigger},

		{Name: "SecretDurability", Run: testSecretDurability},
		{Name: "SecretStoreVault", Run: testSecretStoreVault},
		{Name: "SecretStoreVaultFailure", Run: testSecretStoreVaultFailure},
		{Name: "SecretConsumersAfterRotation", Run: testSecretConsumersAfterRotation},

		{Name: "CollectorExportOTLP", Run: testCollectorExportOTLP},
		{Name: "CollectorExportOTLPHTTP", Run: testCollectorExportOTLPHTTP},
		{Name: "CollectorExportUnreachable", Run: testCollectorExportUnreachable},

		{Name: "LogStoreQueryPath", Run: testLogStoreQueryPath},
		{Name: "MetricStoreQueryPath", Run: testMetricStoreQueryPath},
		{Name: "RscStoreReconciliation", Run: testRscStoreReconciliation},
		{Name: "VisibilityScoping", Run: testVisibilityScoping},

		{Name: "Certificate", Run: testCertificateSetAndServe},
		{Name: "License", Run: testLicenseLifecycle},
		{Name: "ClusterInfo", Run: testClusterInfo},
	}
}

func TailPhases() []suite.Phase {
	return []suite.Phase{
		{Name: "EnterpriseRollingRestart", Run: testEnterpriseRollingRestart},
		{Name: "RscServerFailover", Run: testRscServerFailover},
		{Name: "StorePersistence", Run: testStorePersistence},
		{Name: "ClusterUpgrade", Run: testClusterUpgrade},
	}
}

func ReadyPhase() suite.Phase {
	return suite.Phase{Name: "EnterpriseReady", Run: testEnterpriseReady}
}

func ReclaimPhase() suite.Phase {
	return suite.Phase{Name: "ReclaimApply", Run: testReclaimApply}
}

const (
	EnvSkipApply = "OCTELIUM_E2E_SKIP_APPLY"
	EnvSuite     = "OCTELIUM_E2E_SUITE"
)

const (
	SuiteEnterprise = "enterprise"
	SuiteAll        = "all"
)

func All() []suite.Phase {
	if strings.EqualFold(os.Getenv(EnvSuite), SuiteAll) {
		return withCore()
	}

	return Enterprise()
}

func Enterprise() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())
	ret = append(ret, Phases()...)
	ret = append(ret, suite.Only(suite.Phases(), "ComponentHealth")...)

	return append(ret, TailPhases()...)
}

func withCore() []suite.Phase {
	ret := suite.InsertAfter(suite.Phases(), "ClusterReady", ReadyPhase())
	ret = suite.InsertBefore(ret, "Apply", Phases()...)
	ret = suite.InsertAfter(ret, "Apply", ReclaimPhase())

	if skip, _ := strconv.ParseBool(os.Getenv(EnvSkipApply)); skip {
		ret = suite.Without(ret, "Apply", "ReclaimApply")
	}

	return append(ret, TailPhases()...)
}
