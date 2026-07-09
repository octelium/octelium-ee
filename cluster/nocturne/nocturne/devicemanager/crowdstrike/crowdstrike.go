// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package crowdstrike

import (
	"context"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/hosts"
	"github.com/crowdstrike/gofalcon/falcon/client/zero_trust_assessment"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	csScrollLimit  int64 = 5000
	csDetailsBatch       = 100
	csZTABatch           = 100

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 8192
)

var (
	aidRe       = regexp.MustCompile(`[0-9a-f]{32}`)
	aidAnchored = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

type Manager struct {
	c          *client.CrowdStrikeAPISpecification
	hostFilter string
	ztaEnabled bool
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetCrowdStrike()
	if spec == nil {
		return nil, errors.Errorf("Not a CrowdStrike DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if spec.ClientSecret.GetFromSecret() == "" {
		return nil, errors.Errorf("Empty CrowdStrike clientSecret")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.ClientSecret.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	apiClient, err := falcon.NewClient(&falcon.ApiConfig{
		Context:      ctx,
		ClientId:     spec.ClientID,
		ClientSecret: uenterprisev1.ToSecret(sec).GetValueStr(),
		MemberCID:    spec.MemberCID,
		Cloud:        cloudFromRegion(spec.Region),
		RetryConfig: &falcon.RetryConfig{
			MaxTries:        5,
			InitialInterval: 2 * time.Second,
			MaxInterval:     30 * time.Second,
		},
		Debug: ldflags.IsDev(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not init CrowdStrike client")
	}

	return &Manager{
		c:          apiClient,
		hostFilter: spec.HostFilter,
		ztaEnabled: !spec.DisableZeroTrustAssessment,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_CROWDSTRIKE
}

func (m *Manager) Close() error { return nil }

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType:           corev1.Device_Status_WINDOWS,
			RequireElevation: true,
			ReadRegistry: &devicemgrcommon.ReadRegistry{
				Key:  `HKLM\SYSTEM\CurrentControlSet\Services\CSAgent\Sim`,
				Name: "AG",
			},
		},
		{
			OSType:           corev1.Device_Status_LINUX,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/opt/CrowdStrike/falconctl",
				Args:           []string{"-g", "--aid"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType:           corev1.Device_Status_MAC,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/Applications/Falcon.app/Contents/Resources/falconctl",
				Args:           []string{"stats", "agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	for _, r := range results {
		if r == nil || r.Err != nil || len(r.Output) == 0 {
			continue
		}
		if osType == corev1.Device_Status_WINDOWS {
			if aid := strings.ToLower(hex.EncodeToString(r.Output)); aidAnchored.MatchString(aid) {
				return aid, nil
			}
		}
		if aid := extractAID(string(r.Output)); aid != "" {
			return aid, nil
		}
	}
	return "", nil
}

func extractAID(s string) string {
	return aidRe.FindString(strings.ReplaceAll(strings.ToLower(s), "-", ""))
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	ids, err := m.listHostIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return devicemgrcommon.NewFleet(nil), nil
	}

	details, err := m.getDetails(ctx, ids)
	if err != nil {
		return nil, err
	}

	var zta map[string]*models.DomainSignalProperties
	if m.ztaEnabled {
		if z, zErr := m.getZTA(ctx, ids); zErr == nil {
			zta = z
		} else {
			zap.L().Warn("Could not get CrowdStrike ZTA", zap.Error(zErr))
		}
	}

	entries := make([]*devicemgrcommon.Entry, 0, len(details))
	for aid, d := range details {
		entries = append(entries, toEntry(d, zta[aid]))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

func (m *Manager) listHostIDs(ctx context.Context) ([]string, error) {
	limit := csScrollLimit
	var offset *string
	var out []string

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p := &hosts.QueryDevicesByFilterScrollParams{Context: ctx, Limit: &limit}
		if m.hostFilter != "" {
			f := m.hostFilter
			p.Filter = &f
		}
		if offset != nil && *offset != "" {
			p.Offset = offset
		}

		resp, err := m.c.Hosts.QueryDevicesByFilterScroll(p)
		if err != nil {
			return nil, errors.Wrap(err, "CrowdStrike list hosts")
		}
		if resp.Payload == nil {
			break
		}
		out = append(out, resp.Payload.Resources...)

		var pg *models.DeviceapiDevicePaging
		if resp.Payload.Meta != nil {
			pg = resp.Payload.Meta.Pagination
		}
		if pg == nil || pg.Offset == nil || *pg.Offset == "" || len(resp.Payload.Resources) == 0 {
			break
		}
		offset = pg.Offset
		if pg.Total != nil && int64(len(out)) >= *pg.Total {
			break
		}
	}
	return out, nil
}

func (m *Manager) getDetails(ctx context.Context, ids []string) (map[string]*models.DeviceapiDeviceSwagger, error) {
	out := make(map[string]*models.DeviceapiDeviceSwagger, len(ids))
	for _, batch := range chunk(ids, csDetailsBatch) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := m.c.Hosts.GetDeviceDetailsV2(&hosts.GetDeviceDetailsV2Params{Context: ctx, Ids: batch})
		if err != nil {
			return nil, errors.Wrap(err, "CrowdStrike get device details")
		}
		if resp.Payload == nil {
			continue
		}
		for _, d := range resp.Payload.Resources {
			if d == nil || d.DeviceID == nil {
				continue
			}
			out[strings.ToLower(*d.DeviceID)] = d
		}
	}
	return out, nil
}

func (m *Manager) getZTA(ctx context.Context, ids []string) (map[string]*models.DomainSignalProperties, error) {
	out := make(map[string]*models.DomainSignalProperties, len(ids))
	for _, batch := range chunk(ids, csZTABatch) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := m.c.ZeroTrustAssessment.GetAssessmentV1(&zero_trust_assessment.GetAssessmentV1Params{Context: ctx, Ids: batch})
		if err != nil {
			return nil, errors.Wrap(err, "CrowdStrike get zero trust assessment")
		}
		if resp.Payload == nil {
			continue
		}
		for _, z := range resp.Payload.Resources {
			if z == nil || z.Aid == nil {
				continue
			}
			out[strings.ToLower(*z.Aid)] = z
		}
	}
	return out, nil
}

func toEntry(d *models.DeviceapiDeviceSwagger, z *models.DomainSignalProperties) *devicemgrcommon.Entry {
	aid := ""
	if d.DeviceID != nil {
		aid = strings.ToLower(*d.DeviceID)
	}

	p := &corev1.Device_Status_Posture{
		ExternalID: aid,
		Signals:    csSignals(d),
		Attrs:      csAttrs(d, z),
	}
	if t, ok := parseTime(d.LastSeen); ok {
		p.LastSeenAt = timestamppb.New(t)
	}
	if z != nil && z.Assessment != nil && z.Assessment.Overall != nil {
		overall := *z.Assessment.Overall
		p.RiskLevel = csRisk(overall)
		p.Attrs = withField(p.Attrs, "ztaScoreOverall", float64(overall))

		if z.Assessment.Os != nil {
			p.Attrs = withField(p.Attrs, "ztaScoreOs", float64(*z.Assessment.Os))
		}
		if z.Assessment.SensorConfig != nil {
			p.Attrs = withField(p.Attrs, "ztaScoreSensor", float64(*z.Assessment.SensorConfig))
		}
	}

	return &devicemgrcommon.Entry{
		ExternalID: aid,
		Serial:     d.SerialNumber,
		MACs:       csMACs(d),
		Posture:    p,
	}
}

func csSignals(d *models.DeviceapiDeviceSwagger) map[string]corev1.Device_Status_Posture_SignalState {
	rfm := isYes(d.ReducedFunctionalityMode)
	running := strings.EqualFold(d.Status, "normal") || strings.EqualFold(d.Status, "containment")

	sig := map[string]corev1.Device_Status_Posture_SignalState{
		"agentRunning":  passFail(running),
		"sensorHealthy": passFail(running && !rfm),
	}
	switch strings.ToLower(strings.TrimSpace(d.FilesystemContainmentStatus)) {
	case "", "normal", "not_contained":
		sig["notContained"] = corev1.Device_Status_Posture_PASS
	default:
		sig["notContained"] = corev1.Device_Status_Posture_FAIL
	}
	return sig
}

func csAttrs(d *models.DeviceapiDeviceSwagger, z *models.DomainSignalProperties) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("agentVersion", d.AgentVersion)
	put("status", d.Status)
	put("reducedFunctionalityMode", d.ReducedFunctionalityMode)
	put("productTypeDesc", d.ProductTypeDesc)
	put("provisionStatus", d.ProvisionStatus)
	put("filesystemContainmentStatus", d.FilesystemContainmentStatus)
	if z != nil {
		if z.Cid != nil {
			put("cid", *z.Cid)
		}
		if z.SensorFileStatus != nil {
			put("sensorFileStatus", *z.SensorFileStatus)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return s
}

func csMACs(d *models.DeviceapiDeviceSwagger) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == ' '
		}) {
			nm := devicemgrcommon.NormalizeMAC(part)
			if nm == "" {
				continue
			}
			if _, ok := seen[nm]; ok {
				continue
			}
			seen[nm] = struct{}{}
			out = append(out, part)
		}
	}
	add(d.MacAddress)
	add(d.ConnectionMacAddress)
	return out
}

func cloudFromRegion(r enterprisev1.DeviceManager_Spec_CrowdStrike_Region) falcon.CloudType {
	switch r {
	case enterprisev1.DeviceManager_Spec_CrowdStrike_US_1:
		return falcon.CloudUs1
	case enterprisev1.DeviceManager_Spec_CrowdStrike_US_2:
		return falcon.CloudUs2
	case enterprisev1.DeviceManager_Spec_CrowdStrike_EU_1:
		return falcon.CloudEu1
	case enterprisev1.DeviceManager_Spec_CrowdStrike_US_GOV_1:
		return falcon.CloudUsGov1
	case enterprisev1.DeviceManager_Spec_CrowdStrike_US_GOV_2:
		return falcon.CloudUsGov2
	default:
		return falcon.CloudAutoDiscover
	}
}

func csRisk(score int32) corev1.Device_Status_Posture_RiskLevel {
	switch {
	case score >= 90:
		return corev1.Device_Status_Posture_LOW
	case score >= 70:
		return corev1.Device_Status_Posture_MEDIUM
	case score >= 40:
		return corev1.Device_Status_Posture_HIGH
	default:
		return corev1.Device_Status_Posture_CRITICAL
	}
}

func passFail(ok bool) corev1.Device_Status_Posture_SignalState {
	if ok {
		return corev1.Device_Status_Posture_PASS
	}
	return corev1.Device_Status_Posture_FAIL
}

func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "true", "1", "enabled":
		return true
	}
	return false
}

func parseTime(s string) (time.Time, bool) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func withField(s *structpb.Struct, key string, val interface{}) *structpb.Struct {
	if s == nil {
		s = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if s.Fields == nil {
		s.Fields = map[string]*structpb.Value{}
	}
	if v, err := structpb.NewValue(val); err == nil {
		s.Fields[key] = v
	}
	return s
}

func chunk(ids []string, size int) [][]string {
	if size <= 0 {
		size = 100
	}
	var out [][]string
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[i:end])
	}
	return out
}
