// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

const resourceBackfillBatchSize = 20000

const (
	colCreatedAt      = "created_at"
	colName           = "name"
	colDisplayName    = "display_name"
	colIsSystemHidden = "is_system_hidden"
	colTags           = "tags"

	colSpecKeys              = "spec_keys"
	colSpecType              = "spec_type"
	colSpecState             = "spec_state"
	colSpecMode              = "spec_mode"
	colSpecUrgency           = "spec_urgency"
	colSpecDecision          = "spec_decision"
	colSpecDeadline          = "spec_deadline"
	colSpecEmail             = "spec_email"
	colSpecIsDisabled        = "spec_is_disabled"
	colSpecIsPublic          = "spec_is_public"
	colSpecIsAnonymous       = "spec_is_anonymous"
	colSpecGroups            = "spec_groups"
	colSpecServices          = "spec_services"
	colSpecNamespaces        = "spec_namespaces"
	colSpecLinkingStrategy   = "spec_linking_strategy"
	colSpecLinkingApproval   = "spec_linking_approval_mode"
	colSpecPollingIsDisabled = "spec_polling_is_disabled"
	colSpecSubjectUserUID    = "spec_subject_user_uid"
	colSpecSubjectUserName   = "spec_subject_user_name"
	colSpecServiceUID        = "spec_service_uid"
	colSpecServiceName       = "spec_service_name"
	colSpecCatalogUID        = "spec_catalog_uid"
	colSpecCatalogName       = "spec_catalog_name"

	colStatusKeys           = "status_keys"
	colStatusType           = "status_type"
	colStatusState          = "status_state"
	colStatusStateStatus    = "status_state_status"
	colStatusOsType         = "status_os_type"
	colStatusSyncState      = "status_sync_state"
	colStatusIsConnected    = "status_is_connected"
	colStatusIsBrowser      = "status_is_browser"
	colStatusIsRegistered   = "status_is_registered"
	colStatusAccessEndsAt   = "status_access_ends_at"
	colStatusCredentialType = "status_credential_type"
	colStatusIssuanceState  = "status_issuance_state"
	colStatusIssuanceExpiry = "status_issuance_expires_at"
	colStatusFidoType       = "status_fido_type"
	colStatusFidoIsPasskey  = "status_fido_is_passkey"
	colStatusFidoIsHardware = "status_fido_is_hardware"
	colStatusManagedDevices = "status_managed_devices"
	colStatusLinkedDevices  = "status_linked_devices"
	colStatusWaitingApprove = "status_waiting_approval"
	colStatusAmbiguous      = "status_ambiguous"
	colStatusFailedUpdates  = "status_failed_updates"

	colUserUID               = "user_uid"
	colUserName              = "user_name"
	colDeviceUID             = "device_uid"
	colDeviceName            = "device_name"
	colNamespaceUID          = "namespace_uid"
	colNamespaceName         = "namespace_name"
	colRegionUID             = "region_uid"
	colRegionName            = "region_name"
	colCredentialUID         = "credential_uid"
	colCredentialName        = "credential_name"
	colServiceUID            = "service_uid"
	colServiceName           = "service_name"
	colGroupUID              = "group_uid"
	colGroupName             = "group_name"
	colPolicyUID             = "policy_uid"
	colPolicyName            = "policy_name"
	colPolicyTriggerUID      = "policy_trigger_uid"
	colPolicyTriggerName     = "policy_trigger_name"
	colRequestUID            = "request_uid"
	colRequestName           = "request_name"
	colDirectoryProviderUID  = "directory_provider_uid"
	colDirectoryProviderName = "directory_provider_name"
	colCertificateIssuerUID  = "certificate_issuer_uid"
	colCertificateIssuerName = "certificate_issuer_name"
)

const (
	kindVarchar     = "VARCHAR"
	kindTimestamp   = "TIMESTAMP"
	kindBoolean     = "BOOLEAN"
	kindBigint      = "BIGINT"
	kindVarcharList = "VARCHAR[]"
	kindKeys        = "KEYS"
)

type resourceColumn struct {
	name string
	kind string
	path string
}

func (c *resourceColumn) sqlType() string {
	switch c.kind {
	case kindKeys:
		return kindVarcharList
	default:
		return c.kind
	}
}

func (c *resourceColumn) isScalar() bool {
	switch c.kind {
	case kindVarcharList, kindKeys:
		return false
	default:
		return true
	}
}

func (c *resourceColumn) sourceExpr(source string) string {
	switch c.kind {
	case kindKeys:
		return fmt.Sprintf(`json_keys(%s, '%s')`, source, c.path)
	case kindVarcharList:
		return fmt.Sprintf(`TRY_CAST(json_extract_string(%s, '%s') AS VARCHAR[])`, source, c.path)
	case kindVarchar:
		return fmt.Sprintf(`json_extract_string(%s, '%s')`, source, c.path)
	default:
		return fmt.Sprintf(`TRY_CAST(json_extract_string(%s, '%s') AS %s)`, source, c.path, c.kind)
	}
}

func (c *resourceColumn) scalarExpr(source string, idx int) string {
	if c.kind == kindVarchar {
		return fmt.Sprintf(`%s[%d]`, source, idx)
	}

	return fmt.Sprintf(`TRY_CAST(%s[%d] AS %s)`, source, idx, c.kind)
}

func getResourceColumns() []*resourceColumn {
	return []*resourceColumn{
		{name: colCreatedAt, kind: kindTimestamp, path: "$.metadata.createdAt"},
		{name: colName, kind: kindVarchar, path: "$.metadata.name"},
		{name: colDisplayName, kind: kindVarchar, path: "$.metadata.displayName"},
		{name: colIsSystemHidden, kind: kindBoolean, path: "$.metadata.isSystemHidden"},

		{name: colSpecType, kind: kindVarchar, path: "$.spec.type"},
		{name: colSpecState, kind: kindVarchar, path: "$.spec.state"},
		{name: colSpecMode, kind: kindVarchar, path: "$.spec.mode"},
		{name: colSpecUrgency, kind: kindVarchar, path: "$.spec.urgency"},
		{name: colSpecDecision, kind: kindVarchar, path: "$.spec.decision"},
		{name: colSpecDeadline, kind: kindTimestamp, path: "$.spec.deadline"},
		{name: colSpecEmail, kind: kindVarchar, path: "$.spec.email"},
		{name: colSpecIsDisabled, kind: kindBoolean, path: "$.spec.isDisabled"},
		{name: colSpecIsPublic, kind: kindBoolean, path: "$.spec.isPublic"},
		{name: colSpecIsAnonymous, kind: kindBoolean, path: "$.spec.isAnonymous"},
		{name: colSpecLinkingStrategy, kind: kindVarchar, path: "$.spec.linking.strategy"},
		{name: colSpecLinkingApproval, kind: kindVarchar, path: "$.spec.linking.approvalMode"},
		{name: colSpecPollingIsDisabled, kind: kindBoolean, path: "$.spec.polling.isDisabled"},
		{name: colSpecSubjectUserUID, kind: kindVarchar, path: "$.spec.subject.userRef.uid"},
		{name: colSpecSubjectUserName, kind: kindVarchar, path: "$.spec.subject.userRef.name"},
		{name: colSpecServiceUID, kind: kindVarchar, path: "$.spec.resource.serviceRef.uid"},
		{name: colSpecServiceName, kind: kindVarchar, path: "$.spec.resource.serviceRef.name"},
		{name: colSpecCatalogUID, kind: kindVarchar, path: "$.spec.resource.catalog.catalogRef.uid"},
		{name: colSpecCatalogName, kind: kindVarchar, path: "$.spec.resource.catalog.catalogRef.name"},

		{name: colStatusType, kind: kindVarchar, path: "$.status.type"},
		{name: colStatusState, kind: kindVarchar, path: "$.status.state"},
		{name: colStatusStateStatus, kind: kindVarchar, path: "$.status.state.status"},
		{name: colStatusOsType, kind: kindVarchar, path: "$.status.osType"},
		{name: colStatusSyncState, kind: kindVarchar, path: "$.status.synchronization.state"},
		{name: colStatusIsConnected, kind: kindBoolean, path: "$.status.isConnected"},
		{name: colStatusIsBrowser, kind: kindBoolean, path: "$.status.isBrowser"},
		{name: colStatusIsRegistered, kind: kindBoolean, path: "$.status.isRegistered"},
		{name: colStatusAccessEndsAt, kind: kindTimestamp, path: "$.status.accessEndsAt"},
		{name: colStatusCredentialType, kind: kindVarchar,
			path: "$.status.authentication.info.credential.type"},
		{name: colStatusIssuanceState, kind: kindVarchar, path: "$.status.issuance.state"},
		{name: colStatusIssuanceExpiry, kind: kindTimestamp, path: "$.status.issuance.expiresAt"},
		{name: colStatusFidoType, kind: kindVarchar, path: "$.status.info.fido.type"},
		{name: colStatusFidoIsPasskey, kind: kindBoolean, path: "$.status.info.fido.isPasskey"},
		{name: colStatusFidoIsHardware, kind: kindBoolean, path: "$.status.info.fido.isHardware"},
		{name: colStatusManagedDevices, kind: kindBigint, path: "$.status.collection.managedDevices"},
		{name: colStatusLinkedDevices, kind: kindBigint, path: "$.status.linking.linkedDevices"},
		{name: colStatusWaitingApprove, kind: kindBigint, path: "$.status.linking.waitingApproval"},
		{name: colStatusAmbiguous, kind: kindBigint, path: "$.status.linking.ambiguous"},
		{name: colStatusFailedUpdates, kind: kindBigint, path: "$.status.linking.failedUpdates"},

		{name: colUserUID, kind: kindVarchar, path: "$.status.userRef.uid"},
		{name: colUserName, kind: kindVarchar, path: "$.status.userRef.name"},
		{name: colDeviceUID, kind: kindVarchar, path: "$.status.deviceRef.uid"},
		{name: colDeviceName, kind: kindVarchar, path: "$.status.deviceRef.name"},
		{name: colNamespaceUID, kind: kindVarchar, path: "$.status.namespaceRef.uid"},
		{name: colNamespaceName, kind: kindVarchar, path: "$.status.namespaceRef.name"},
		{name: colRegionUID, kind: kindVarchar, path: "$.status.regionRef.uid"},
		{name: colRegionName, kind: kindVarchar, path: "$.status.regionRef.name"},
		{name: colCredentialUID, kind: kindVarchar, path: "$.status.credentialRef.uid"},
		{name: colCredentialName, kind: kindVarchar, path: "$.status.credentialRef.name"},
		{name: colServiceUID, kind: kindVarchar, path: "$.status.serviceRef.uid"},
		{name: colServiceName, kind: kindVarchar, path: "$.status.serviceRef.name"},
		{name: colGroupUID, kind: kindVarchar, path: "$.status.groupRef.uid"},
		{name: colGroupName, kind: kindVarchar, path: "$.status.groupRef.name"},
		{name: colPolicyUID, kind: kindVarchar, path: "$.status.policyRef.uid"},
		{name: colPolicyName, kind: kindVarchar, path: "$.status.policyRef.name"},
		{name: colPolicyTriggerUID, kind: kindVarchar, path: "$.status.policyTriggerRef.uid"},
		{name: colPolicyTriggerName, kind: kindVarchar, path: "$.status.policyTriggerRef.name"},
		{name: colRequestUID, kind: kindVarchar, path: "$.status.requestRef.uid"},
		{name: colRequestName, kind: kindVarchar, path: "$.status.requestRef.name"},
		{name: colDirectoryProviderUID, kind: kindVarchar, path: "$.status.directoryProviderRef.uid"},
		{name: colDirectoryProviderName, kind: kindVarchar, path: "$.status.directoryProviderRef.name"},
		{name: colCertificateIssuerUID, kind: kindVarchar, path: "$.status.certificateIssuerRef.uid"},
		{name: colCertificateIssuerName, kind: kindVarchar, path: "$.status.certificateIssuerRef.name"},

		{name: colTags, kind: kindVarcharList, path: "$.metadata.tags"},
		{name: colSpecGroups, kind: kindVarcharList, path: "$.spec.groups"},
		{name: colSpecServices, kind: kindVarcharList,
			path: "$.spec.resourceCollection.service.services"},
		{name: colSpecNamespaces, kind: kindVarcharList,
			path: "$.spec.resourceCollection.service.namespaces"},

		{name: colSpecKeys, kind: kindKeys, path: "$.spec"},
		{name: colStatusKeys, kind: kindKeys, path: "$.status"},
	}
}

var resourceColumnsByPath = func() map[string]*resourceColumn {
	ret := make(map[string]*resourceColumn)
	for _, column := range getResourceColumns() {
		if column.kind == kindKeys {
			continue
		}
		ret[column.path] = column
	}

	return ret
}()

func getResourceColumnByPath(path string) (*resourceColumn, bool) {
	column, ok := resourceColumnsByPath[path]
	return column, ok
}

func getResourceColumnNameByPath(path string) (string, bool) {
	column, ok := getResourceColumnByPath(path)
	if !ok {
		return "", false
	}

	return column.name, true
}

func getResourceInsertQuery() string {
	columns := getResourceColumns()

	names := []string{"api", "version", "kind", "uid", "resource_version", "rsc", "rsc_str"}
	values := []string{"$1", "$2", "$3", "$4", "$5", "j", "$7"}

	var scalarPaths []string
	for _, column := range columns {
		if column.isScalar() {
			scalarPaths = append(scalarPaths, fmt.Sprintf("'%s'", column.path))
		}
	}

	idx := 0
	for _, column := range columns {
		names = append(names, column.name)

		if !column.isScalar() {
			values = append(values, column.sourceExpr("j"))
			continue
		}

		idx++
		values = append(values, column.scalarExpr("v", idx))
	}

	return fmt.Sprintf(`
INSERT INTO resources
    (%s)
SELECT
    %s
FROM (
    SELECT j, json_extract_string(j, [%s]) AS v FROM (SELECT CAST($6 AS JSON) AS j)
)
`, strings.Join(names, ", "), strings.Join(values, ", "), strings.Join(scalarPaths, ", "))
}

func getResourceInsertArgs(api, version, kind, uid, resourceVersion, rscJSON, rscStr string) []any {
	return []any{api, version, kind, uid, resourceVersion, rscJSON, rscStr}
}

func (s *Server) initResourceColumns(ctx context.Context) error {
	columns := getResourceColumns()

	for _, column := range columns {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
			`ALTER TABLE resources ADD COLUMN IF NOT EXISTS %s %s`,
			column.name, column.sqlType())); err != nil {
			return err
		}
	}

	return s.backfillResourceColumns(ctx, columns)
}

func getResourceBackfillAssignments(columns []*resourceColumn) string {
	var scalarPaths []string
	var assignments []string

	for _, column := range columns {
		if !column.isScalar() {
			continue
		}
		scalarPaths = append(scalarPaths, fmt.Sprintf("'%s'", column.path))
	}

	scalars := fmt.Sprintf(`json_extract_string(rsc, [%s])`, strings.Join(scalarPaths, ", "))

	idx := 0
	for _, column := range columns {
		if !column.isScalar() {
			assignments = append(assignments,
				fmt.Sprintf("%s = %s", column.name, column.sourceExpr("rsc")))
			continue
		}

		idx++
		assignments = append(assignments,
			fmt.Sprintf("%s = %s", column.name, column.scalarExpr(scalars, idx)))
	}

	return strings.Join(assignments, ", ")
}

func (s *Server) backfillResourceColumns(ctx context.Context, columns []*resourceColumn) error {
	assignments := getResourceBackfillAssignments(columns)

	remaining := int64(-1)
	total := int64(0)

	for {
		var pending int64
		if err := s.db.QueryRowContext(ctx, fmt.Sprintf(
			`SELECT COUNT(*) FROM resources WHERE %s IS NULL`, colCreatedAt)).Scan(&pending); err != nil {
			return err
		}

		if pending == 0 || (remaining >= 0 && pending >= remaining) {
			break
		}

		remaining = pending

		result, err := s.db.ExecContext(ctx, fmt.Sprintf(`
UPDATE resources SET %s WHERE rowid IN (
	SELECT rowid FROM resources WHERE %s IS NULL ORDER BY rowid LIMIT %d
)`, assignments, colCreatedAt, resourceBackfillBatchSize))
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err == nil {
			total += count
		}
	}

	if total > 0 {
		zap.L().Info("Backfilled the RscStore indexed columns", zap.Int64("resources", total))
	}

	return nil
}
