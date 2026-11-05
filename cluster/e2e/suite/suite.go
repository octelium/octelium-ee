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
		{Name: "CollectorExporterAPI", Run: testCollectorExporterAPI},
		{Name: "EnterpriseSecretAPI", Run: testEnterpriseSecretAPI},
		{Name: "EnterpriseAPIAuthorization", Run: testEnterpriseAPIAuthorization},

		{Name: "SCIM", Run: testSCIM},
		{Name: "SCIMProtocol", Run: testSCIMProtocol},
		{Name: "SCIMDisabledProvider", Run: testSCIMDisabledProvider},
		{Name: "SCIMToDataPlane", Run: testSCIMToDataPlane},
		{Name: "KeycloakDirectoryProvider", Run: testKeycloakDirectoryProvider},
		{Name: "KeycloakBadCredentials", Run: testKeycloakBadCredentials},
		{Name: "KeycloakFailureRecovery", Run: testKeycloakFailureRecovery},
		{Name: "DirectoryProviderIsolation", Run: testDirectoryProviderIsolation},
		{Name: "DirectoryLocalUserOwnership", Run: testDirectoryLocalUserOwnership},
		{Name: "DirectoryUserReactivation", Run: testDirectoryUserReactivation},

		{Name: "DeviceManagerAPI", Run: testDeviceManagerAPI},
		{Name: "DeviceManagerProbePlan", Run: testDeviceManagerProbePlan},
		{Name: "DeviceManagerSecretRotation", Run: testDeviceManagerSecretRotation},
		{Name: "DeviceManagerPostureLifecycle", Run: testDeviceManagerPostureLifecycle},
		{Name: "DeviceManagerReplacementUIDIsolation", Run: testDeviceManagerReplacementUIDIsolation},
		{Name: "DeviceManagerResilience", Run: testDeviceManagerResilience},

		{Name: "AccessAutoApprove", Run: testAccessAutoApprove},
		{Name: "AccessSingleStepReview", Run: testAccessSingleStepReview},
		{Name: "AccessMultiStepQuorum", Run: testAccessMultiStepQuorum},
		{Name: "AccessRejectCancelRevoke", Run: testAccessRejectCancelRevoke},
		{Name: "AccessTimeouts", Run: testAccessTimeouts},
		{Name: "AccessControllerRecovery", Run: testAccessControllerRecovery},
		{Name: "AccessDirectService", Run: testAccessDirectService},
		{Name: "AccessDelegatedSubject", Run: testAccessDelegatedSubject},
		{Name: "AccessOverlappingGrants", Run: testAccessOverlappingGrants},
		{Name: "AccessReviewRevision", Run: testAccessReviewRevision},
		{Name: "AccessPendingDeadline", Run: testAccessPendingDeadline},
		{Name: "AccessReviewerMembership", Run: testAccessReviewerMembership},
		{Name: "AccessCatalogResources", Run: testAccessCatalogResources},
		{Name: "AccessUIDBinding", Run: testAccessUIDBinding},
		{Name: "AccessPolicyPriority", Run: testAccessPolicyPriority},
		{Name: "AccessAPIScoping", Run: testAccessAPIScoping},
		{Name: "AccessActorTrail", Run: testAccessActorTrail},
		{Name: "AccessPortal", Run: testAccessPortal},
		{Name: "ConsolePortal", Run: testConsolePortal},

		{Name: "PolicyPortalParity", Run: testPolicyPortalParity},
		{Name: "PolicyPortalUnderChurn", Run: testPolicyPortalUnderChurn},
		{Name: "PolicyPortalWithPolicyTrigger", Run: testPolicyPortalWithPolicyTrigger},
		{Name: "PolicyPortalAdditionalIsolation", Run: testPolicyPortalAdditionalIsolation},

		{Name: "SecretDurability", Run: testSecretDurability},
		{Name: "SecretLiveRotation", Run: testSecretLiveRotation},
		{Name: "SecretStoreVault", Run: testSecretStoreVault},
		{Name: "SecretStoreVaultFailure", Run: testSecretStoreVaultFailure},
		{Name: "SecretStoreVaultKeyRotation", Run: testSecretStoreVaultKeyRotation},
		{Name: "SecretConsumersAfterRotation", Run: testSecretConsumersAfterRotation},
		{Name: "SecretConsumerIsolation", Run: testSecretConsumerIsolation},

		{Name: "CollectorExportOTLP", Run: testCollectorExportOTLP},
		{Name: "CollectorExportOTLPHTTP", Run: testCollectorExportOTLPHTTP},
		{Name: "CollectorExportUnreachable", Run: testCollectorExportUnreachable},
		{Name: "CollectorLiveUpdate", Run: testCollectorLiveUpdate},
		{Name: "CollectorOutageRecovery", Run: testCollectorOutageRecovery},
		{Name: "CollectorSignalRouting", Run: testCollectorSignalRouting},
		{Name: "CollectorSecretRotation", Run: testCollectorSecretRotation},
		{Name: "CollectorMetricsOnlyAndPipelineReactivation", Run: testCollectorMetricsOnlyAndPipelineReactivation},
		{Name: "CollectorExporterReplacementRouting", Run: testCollectorExporterReplacementRouting},

		{Name: "LogStoreQueryPath", Run: testLogStoreQueryPath},
		{Name: "AccessLogLifecycle", Run: testAccessLogLifecycle},
		{Name: "AccessLogAggregations", Run: testAccessLogAggregations},
		{Name: "AuthenticationLogTrail", Run: testAuthenticationLogTrail},
		{Name: "AuthenticationLogFailure", Run: testAuthenticationLogFailure},
		{Name: "ComponentLogAggregations", Run: testComponentLogAggregations},
		{Name: "LogStoreIngestionRecovery", Run: testLogStoreIngestionRecovery},
		{Name: "LogStoreWindowPagination", Run: testLogStoreWindowPagination},
		{Name: "LogStoreWindowedSeries", Run: testLogStoreWindowedSeries},
		{Name: "LogStoreAuditResourceTrail", Run: testLogStoreAuditResourceTrail},
		{Name: "MetricStoreQueryPath", Run: testMetricStoreQueryPath},
		{Name: "MetricStoreLimits", Run: testMetricStoreLimits},
		{Name: "MetricStoreDescriptorRoundTrip", Run: testMetricStoreDescriptorRoundTrip},
		{Name: "RscStoreReconciliation", Run: testRscStoreReconciliation},
		{Name: "RscStoreResourceReplacement", Run: testRscStoreResourceReplacement},
		{Name: "RscStoreAccessResources", Run: testRscStoreAccessResources},
		{Name: "RscStoreCordiumResources", Run: testRscStoreCordiumResources},
		{Name: "RscStoreSecretRedaction", Run: testRscStoreSecretRedaction},
		{Name: "RscStoreSearchLifecycle", Run: testRscStoreSearchLifecycle},
		{Name: "RscStoreSummaryWindows", Run: testRscStoreSummaryWindows},
		{Name: "RscStoreAccessActivity", Run: testRscStoreAccessActivity},
		{Name: "VisibilityScoping", Run: testVisibilityScoping},

		{Name: "ClusterSummaryFanOut", Run: testClusterSummaryFanOut},
		{Name: "ClusterHealthSubsystems", Run: testClusterHealthSubsystems},
		{Name: "NocturneReplicaScaling", Run: testNocturneReplicaScaling},

		{Name: "EnterpriseConditionCompiler", Run: testEnterpriseConditionCompiler},
		{Name: "PublicServerOIDC", Run: testPublicServerOIDC},
		{Name: "PublicServerIntegrationCallback", Run: testPublicServerIntegrationCallback},
		{Name: "PublicServerMetadataHeaders", Run: testPublicServerMetadataHeaders},

		{Name: "Certificate", Run: testCertificateSetAndServe},
		{Name: "CloudmanNamespaceReplacement", Run: testCloudmanNamespaceReplacement},
		{Name: "License", Run: testLicenseLifecycle},
		{Name: "ClusterInfo", Run: testClusterInfo},
		{Name: "RegionJoin", Run: testRegionJoin},
		{Name: "CollectorRemoteService", Run: testCollectorRemoteService},
		{Name: "Agent", Run: testAgent},
	}
}

func MultiRegionPhases() []suite.Phase {
	return []suite.Phase{
		{Name: "MultiRegionJoin", Run: testMultiRegionJoin},
		{Name: "MultiRegionComponents", Run: testMultiRegionComponents},
		{Name: "MultiRegionTelemetry", Run: testMultiRegionTelemetry},
		{Name: "MultiRegionService", Run: testMultiRegionService},
		{Name: "MultiRegionUpgrade", Run: testMultiRegionUpgrade},
	}
}

func ClusterManagerPhases() []suite.Phase {
	return []suite.Phase{
		{Name: "ClusterManagerRestartDuringJoin", Run: testClusterManagerRestartDuringJoin},
		{Name: "ClusterManagerJobRetry", Run: testClusterManagerJobRetry},
		{Name: "ClusterManagerRestartDuringUpgrade", Run: testClusterManagerRestartDuringUpgrade},
	}
}

func ReleaseUpgradePhases() []suite.Phase {
	return []suite.Phase{
		{Name: "ReleaseUpgradeSource", Run: testReleaseUpgradeSource},
		{Name: "MultiRegionJoin", Run: testMultiRegionJoin},
		{Name: "ReleaseUpgradeClusterConfigApply", Run: testReleaseUpgradeClusterConfigApply},
		{Name: "ReleaseUpgrade", Run: testReleaseUpgrade},
		{Name: "MultiRegionComponents", Run: testMultiRegionComponents},
		{Name: "MultiRegionService", Run: testMultiRegionService},
	}
}

func InstallRecoveryPhases() []suite.Phase {
	return []suite.Phase{
		{Name: "InstallRecoveryRetries", Run: testInstallRecoveryRetries},
		{Name: "ClusterUpgrade", Run: testClusterUpgrade},
	}
}

func TailPhases() []suite.Phase {
	return []suite.Phase{
		{Name: "EnterpriseRollingRestart", Run: testEnterpriseRollingRestart},
		{Name: "RscServerFailover", Run: testRscServerFailover},
		{Name: "StorePersistence", Run: testStorePersistence},
		{Name: "ClusterUpgrade", Run: testClusterUpgrade},
		{Name: "ClusterManagerUpgradeLifecycle", Run: testClusterManagerUpgradeLifecycle},
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
	EnvSuite     = "OCTELIUM_EE_E2E_SUITE"
)

const (
	SuiteEnterprise  = "enterprise"
	SuiteAll         = "all"
	SuiteMultiRegion = "multi-region"

	SuiteClusterManager = "cluster-manager"

	SuiteReleaseUpgrade  = "release-upgrade"
	SuiteInstallRecovery = "install-recovery"
)

func All() []suite.Phase {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvSuite))) {
	case SuiteAll:
		return withCore()
	case SuiteMultiRegion:
		return MultiRegion()
	case SuiteClusterManager:
		return ClusterManager()
	case SuiteReleaseUpgrade:
		return ReleaseUpgrade()
	case SuiteInstallRecovery:
		return InstallRecovery()
	default:
		return Enterprise()
	}
}

func Enterprise() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())
	ret = append(ret, Phases()...)
	ret = append(ret, suite.Only(suite.Phases(), "ComponentHealth")...)

	return append(ret, TailPhases()...)
}

func MultiRegion() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())

	return append(ret, MultiRegionPhases()...)
}

func ClusterManager() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())

	return append(ret, ClusterManagerPhases()...)
}

func ReleaseUpgrade() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())

	return append(ret, ReleaseUpgradePhases()...)
}

func InstallRecovery() []suite.Phase {
	ret := suite.Only(suite.Phases(), "ClusterReady")
	ret = append(ret, ReadyPhase())

	return append(ret, InstallRecoveryPhases()...)
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
