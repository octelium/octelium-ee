// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

const (
	maxDeviceManagerStringBytes = 512
	maxDeviceManagerURLBytes    = 2048
	maxDeviceManagerFilterBytes = 4096
	maxDeviceManagerListItems   = 128
	maxDeviceManagerCELBytes    = 10000
	maxDeviceManagerOPABytes    = 100000

	defaultDeviceManagerPollInterval = 5 * time.Minute
	minDeviceManagerPollInterval     = 30 * time.Second
	defaultDeviceManagerStaleAfter   = time.Hour
)

var rgxDeviceBindingUID = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

func (s *Server) CreateDeviceManager(ctx context.Context, req *enterprisev1.DeviceManager) (*enterprisev1.DeviceManager, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.EnterpriseC().GetDeviceManager(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, serr.InvalidArg("The DeviceManager %s already exists", req.Metadata.Name)
	}

	if !grpcerr.IsNotFound(err) {
		return nil, serr.K8sInternal(err)
	}

	if err := s.validateDeviceManager(ctx, req); err != nil {
		return nil, err
	}

	item := &enterprisev1.DeviceManager{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status: &enterprisev1.DeviceManager_Status{
			Type: getDeviceManagerStatusType(req.Spec),
		},
	}

	item, err = s.octeliumC.EnterpriseC().CreateDeviceManager(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) GetDeviceManager(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.DeviceManager, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetDeviceManager(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListDeviceManager(ctx context.Context, req *enterprisev1.ListDeviceManagerOptions) (*enterprisev1.DeviceManagerList, error) {
	if req == nil {
		req = &enterprisev1.ListDeviceManagerOptions{}
	}

	itemList, err := s.octeliumC.EnterpriseC().ListDeviceManager(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) UpdateDeviceManager(ctx context.Context, req *enterprisev1.DeviceManager) (*enterprisev1.DeviceManager, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetDeviceManager(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if err := s.validateDeviceManager(ctx, req); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec
	if item.Status == nil {
		item.Status = &enterprisev1.DeviceManager_Status{}
	}
	item.Status.Type = getDeviceManagerStatusType(req.Spec)

	item, err = s.octeliumC.EnterpriseC().UpdateDeviceManager(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) DeleteDeviceManager(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	item, err := s.octeliumC.EnterpriseC().GetDeviceManager(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if slices.Contains(cc.GetSpec().GetDeviceManagers(), item.Metadata.Name) {
		return nil, grpcutils.InvalidArg(
			"The DeviceManager %s is used by the ClusterConfig deviceManagers list", item.Metadata.Name)
	}

	_, err = s.octeliumC.EnterpriseC().DeleteDeviceManager(ctx, &rmetav1.DeleteOptions{Uid: item.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) ResetDeviceBinding(ctx context.Context, req *enterprisev1.ResetDeviceBindingRequest) (*metav1.OperationResult, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	if err := apivalidation.CheckObjectRef(req.DeviceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	if !rgxDeviceBindingUID.MatchString(req.BindingUID) {
		return nil, grpcutils.InvalidArg("Invalid bindingUID")
	}

	dev, err := s.octeliumC.CoreC().GetDevice(ctx, apivalidation.ObjectReferenceToRGetOptions(req.DeviceRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if dev.Status.Binding == nil {
		return &metav1.OperationResult{}, nil
	}

	if dev.Status.Binding.Uid != req.BindingUID {
		return nil, grpcutils.InvalidArg("The Device Binding has changed")
	}

	dev.Status.Binding = nil
	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	if _, err := s.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) validateDeviceManager(ctx context.Context, req *enterprisev1.DeviceManager) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil DeviceManager")
	}

	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	switch spec.Type.(type) {
	case *enterprisev1.DeviceManager_Spec_CrowdStrike_:
		typ := spec.GetCrowdStrike()
		if typ == nil {
			return grpcutils.InvalidArg("Nil CrowdStrike spec")
		}

		if _, ok := enterprisev1.DeviceManager_Spec_CrowdStrike_Region_name[int32(typ.Region)]; !ok {
			return grpcutils.InvalidArg("Invalid CrowdStrike region")
		}
		if err := s.validateGenStr(typ.ClientID, true, "clientID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ClientSecret); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.MemberCID, false, "memberCID"); err != nil {
			return err
		}
		if err := validateDeviceManagerFilter(typ.HostFilter, "hostFilter"); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_SentinelOne_:
		typ := spec.GetSentinelOne()
		if typ == nil {
			return grpcutils.InvalidArg("Nil SentinelOne spec")
		}

		if err := validateDeviceManagerURL(typ.ManagementURL, true, "managementURL"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}
		if err := s.validateDeviceManagerIDs(typ.SiteIDs, "siteIDs"); err != nil {
			return err
		}
		if err := s.validateDeviceManagerIDs(typ.AccountIDs, "accountIDs"); err != nil {
			return err
		}
		if err := s.validateDeviceManagerAgentFilters(typ.AgentFilters); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_MicrosoftIntune_:
		typ := spec.GetMicrosoftIntune()
		if typ == nil {
			return grpcutils.InvalidArg("Nil MicrosoftIntune spec")
		}

		if err := s.validateGenStr(typ.TenantID, true, "tenantID"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ClientID, true, "clientID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ClientSecret); err != nil {
			return err
		}
		if _, ok := enterprisev1.DeviceManager_Spec_MicrosoftIntune_Cloud_name[int32(typ.Cloud)]; !ok {
			return grpcutils.InvalidArg("Invalid MicrosoftIntune cloud")
		}
		if err := validateDeviceManagerFilter(typ.Filter, "filter"); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_Jamf_:
		typ := spec.GetJamf()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Jamf spec")
		}

		if err := validateDeviceManagerURL(typ.BaseURL, true, "baseURL"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ClientID, true, "clientID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ClientSecret); err != nil {
			return err
		}
		if err := validateDeviceManagerFilter(typ.Filter, "filter"); err != nil {
			return err
		}
		if err := validateDeviceManagerNames(typ.CompliantGroups, "compliantGroups"); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_OnePassword_:
		typ := spec.GetOnePassword()
		if typ == nil {
			return grpcutils.InvalidArg("Nil OnePassword spec")
		}

		if err := validateDeviceManagerURL(typ.BaseURL, false, "baseURL"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_FleetDM_:
		typ := spec.GetFleetDM()
		if typ == nil {
			return grpcutils.InvalidArg("Nil FleetDM spec")
		}

		if err := validateDeviceManagerURL(typ.BaseURL, true, "baseURL"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	case *enterprisev1.DeviceManager_Spec_Huntress_:
		typ := spec.GetHuntress()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Huntress spec")
		}

		if err := validateDeviceManagerURL(typ.BaseURL, false, "baseURL"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ApiKey, true, "apiKey"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApiSecret); err != nil {
			return err
		}
		if len(typ.OrganizationIDs) > maxDeviceManagerListItems {
			return grpcutils.InvalidArg("Too many organizationIDs")
		}
		for _, organizationID := range typ.OrganizationIDs {
			if organizationID <= 0 {
				return grpcutils.InvalidArg("Invalid organizationID: %d", organizationID)
			}
		}

	case *enterprisev1.DeviceManager_Spec_Iru_:
		typ := spec.GetIru()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Iru spec")
		}

		if err := validateDeviceManagerURL(typ.BaseURL, true, "baseURL"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	default:
		return grpcutils.InvalidArg("You must set the DeviceManager type")
	}

	if spec.Condition != nil {
		if err := s.validateDeviceManagerCondition(ctx, spec.Condition, 0); err != nil {
			return err
		}
	}

	if err := validateDeviceManagerPolling(spec.Polling); err != nil {
		return err
	}

	return validateDeviceManagerLinking(spec)
}

func validateDeviceManagerPolling(polling *enterprisev1.DeviceManager_Spec_Polling) error {
	if polling == nil {
		return nil
	}

	for _, d := range []*metav1.Duration{
		polling.Interval,
		polling.Timeout,
		polling.StaleAfter,
		polling.MaxObservationAge,
	} {
		if err := apivalidation.ValidateDuration(d); err != nil {
			return err
		}
	}

	interval := defaultDeviceManagerPollInterval
	if polling.Interval != nil {
		interval = umetav1.ToDuration(polling.Interval).ToGo()
	}
	if interval < minDeviceManagerPollInterval {
		interval = minDeviceManagerPollInterval
	}

	staleAfter := defaultDeviceManagerStaleAfter
	if polling.StaleAfter != nil {
		staleAfter = umetav1.ToDuration(polling.StaleAfter).ToGo()
	}

	if staleAfter <= interval {
		return grpcutils.InvalidArg("staleAfter must be longer than the polling interval")
	}

	return nil
}

func validateDeviceManagerLinking(spec *enterprisev1.DeviceManager_Spec) error {
	linking := spec.Linking
	if linking == nil {
		return nil
	}

	strategy := linking.Strategy
	switch strategy {
	case enterprisev1.DeviceManager_Spec_Linking_STRATEGY_UNSET:
		strategy = enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE
	case enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
		enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
		enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE:
	default:
		return grpcutils.InvalidArg("Invalid linking strategy")
	}

	if err := apivalidation.ValidateDuration(linking.VerificationInterval); err != nil {
		return err
	}

	hasProbes := deviceManagerSupportsProbes(spec)

	if linking.RequireAgreement {
		if strategy != enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE {
			return grpcutils.InvalidArg("requireAgreement requires the IDENTITY_AND_PROBE strategy")
		}
		if !hasProbes {
			return grpcutils.InvalidArg("requireAgreement is not supported by this DeviceManager type")
		}
	}

	if strategy == enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY && !hasProbes {
		return grpcutils.InvalidArg("The PROBE_ONLY strategy is not supported by this DeviceManager type")
	}

	if linking.VerificationInterval != nil &&
		(strategy == enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY || !hasProbes) {
		return grpcutils.InvalidArg("verificationInterval requires a strategy that uses probes")
	}

	if linking.RequireOwnerMatch && !deviceManagerSupportsOwner(spec) {
		return grpcutils.InvalidArg("requireOwnerMatch is not supported by this DeviceManager type")
	}

	return nil
}

func deviceManagerSupportsProbes(spec *enterprisev1.DeviceManager_Spec) bool {
	switch spec.Type.(type) {
	case *enterprisev1.DeviceManager_Spec_Huntress_,
		*enterprisev1.DeviceManager_Spec_Iru_:
		return false
	default:
		return true
	}
}

func deviceManagerSupportsOwner(spec *enterprisev1.DeviceManager_Spec) bool {
	switch spec.Type.(type) {
	case *enterprisev1.DeviceManager_Spec_MicrosoftIntune_,
		*enterprisev1.DeviceManager_Spec_Jamf_,
		*enterprisev1.DeviceManager_Spec_OnePassword_,
		*enterprisev1.DeviceManager_Spec_Iru_:
		return true
	default:
		return false
	}
}

func (s *Server) validateDeviceManagerCondition(ctx context.Context, c *corev1.Condition, depth int) error {
	if c == nil {
		return grpcutils.InvalidArg("Nil Condition")
	}

	if depth > maxConditionDepth {
		return grpcutils.InvalidArg("Condition nesting is too deep")
	}

	validateChildren := func(children []*corev1.Condition, field string) error {
		if len(children) == 0 {
			return grpcutils.InvalidArg("Empty %s Condition", field)
		}
		if len(children) > maxConditionChildren {
			return grpcutils.InvalidArg("%s Condition has too many children", field)
		}
		for _, child := range children {
			if err := s.validateDeviceManagerCondition(ctx, child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	switch c.Type.(type) {
	case *corev1.Condition_All_:
		return validateChildren(c.GetAll().GetOf(), "all")
	case *corev1.Condition_Any_:
		return validateChildren(c.GetAny().GetOf(), "any")
	case *corev1.Condition_None_:
		return validateChildren(c.GetNone().GetOf(), "none")
	case *corev1.Condition_Match:
		return validateDeviceManagerCEL(ctx, c.GetMatch())
	case *corev1.Condition_Not:
		return validateDeviceManagerCEL(ctx, c.GetNot())
	case *corev1.Condition_MatchAny:
		return nil
	case *corev1.Condition_Opa:
		return validateDeviceManagerOPA(ctx, c.GetOpa().GetInline())
	default:
		return grpcutils.InvalidArg("Invalid Condition type")
	}
}

func validateDeviceManagerCEL(ctx context.Context, arg string) error {
	if strings.TrimSpace(arg) == "" {
		return grpcutils.InvalidArg("Empty CEL expression")
	}

	if len(arg) > maxDeviceManagerCELBytes {
		return grpcutils.InvalidArg("CEL expression is too long")
	}

	engine, err := celengine.New(ctx, &celengine.Opts{})
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	if err := engine.AddPolicy(ctx, arg); err != nil {
		return grpcutils.InvalidArgWithErr(err)
	}

	return nil
}

func validateDeviceManagerOPA(ctx context.Context, arg string) error {
	if strings.TrimSpace(arg) == "" {
		return grpcutils.InvalidArg("Empty OPA script")
	}

	if len(arg) > maxDeviceManagerOPABytes {
		return grpcutils.InvalidArg("OPA script is too large")
	}

	engine, err := celengine.New(ctx, &celengine.Opts{})
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	if err := engine.AddPolicyOPA(ctx, arg); err != nil {
		return grpcutils.InvalidArgWithErr(err)
	}

	return nil
}

func validateDeviceManagerURL(v string, required bool, field string) error {
	if strings.TrimSpace(v) == "" {
		if required {
			return grpcutils.InvalidArg("%s is required", field)
		}
		return nil
	}

	if len(v) > maxDeviceManagerURLBytes {
		return grpcutils.InvalidArg("%s is too long", field)
	}

	u, err := url.Parse(v)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return grpcutils.InvalidArg("Invalid %s", field)
	}

	if u.Scheme != "https" {
		return grpcutils.InvalidArg("%s must be an HTTPS URL", field)
	}

	return nil
}

func validateDeviceManagerFilter(v string, field string) error {
	if len(v) > maxDeviceManagerFilterBytes {
		return grpcutils.InvalidArg("%s is too long", field)
	}

	if strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return grpcutils.InvalidArg("%s contains invalid characters", field)
	}

	return nil
}

func validateDeviceManagerNames(names []string, field string) error {
	if len(names) > maxDeviceManagerListItems {
		return grpcutils.InvalidArg("Too many %s", field)
	}

	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return grpcutils.InvalidArg("Empty item in %s", field)
		}
		if len(name) > maxDeviceManagerStringBytes {
			return grpcutils.InvalidArg("Item in %s is too long", field)
		}
		if strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return grpcutils.InvalidArg("Item in %s contains invalid characters", field)
		}
	}

	return nil
}

func (s *Server) validateDeviceManagerIDs(ids []string, field string) error {
	if len(ids) > maxDeviceManagerListItems {
		return grpcutils.InvalidArg("Too many %s", field)
	}

	for _, id := range ids {
		if err := s.validateGenStr(id, true, field); err != nil {
			return err
		}
		if strings.Contains(id, ",") {
			return grpcutils.InvalidArg("Invalid item in %s: %s", field, id)
		}
	}

	return nil
}

func (s *Server) validateDeviceManagerAgentFilters(filters map[string]string) error {
	if len(filters) > maxDeviceManagerListItems {
		return grpcutils.InvalidArg("Too many agentFilters")
	}

	for k, v := range filters {
		if err := s.validateGenStr(k, true, "agentFilters key"); err != nil {
			return err
		}

		switch strings.ToLower(k) {
		case "siteids", "accountids", "cursor", "limit":
			return grpcutils.InvalidArg("The agentFilters key %s is not allowed", k)
		}

		if err := s.validateGenStr(v, true, "agentFilters value"); err != nil {
			return err
		}
	}

	return nil
}

func getDeviceManagerStatusType(spec *enterprisev1.DeviceManager_Spec) enterprisev1.DeviceManager_Status_Type {
	switch spec.GetType().(type) {
	case *enterprisev1.DeviceManager_Spec_CrowdStrike_:
		return enterprisev1.DeviceManager_Status_CROWDSTRIKE
	case *enterprisev1.DeviceManager_Spec_SentinelOne_:
		return enterprisev1.DeviceManager_Status_SENTINELONE
	case *enterprisev1.DeviceManager_Spec_MicrosoftIntune_:
		return enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE
	case *enterprisev1.DeviceManager_Spec_Jamf_:
		return enterprisev1.DeviceManager_Status_JAMF_PRO
	case *enterprisev1.DeviceManager_Spec_OnePassword_:
		return enterprisev1.DeviceManager_Status_ONEPASSWORD
	case *enterprisev1.DeviceManager_Spec_FleetDM_:
		return enterprisev1.DeviceManager_Status_FLEETDM
	case *enterprisev1.DeviceManager_Spec_Huntress_:
		return enterprisev1.DeviceManager_Status_HUNTRESS
	case *enterprisev1.DeviceManager_Spec_Iru_:
		return enterprisev1.DeviceManager_Status_IRU
	default:
		return enterprisev1.DeviceManager_Status_TYPE_UNKNOWN
	}
}
