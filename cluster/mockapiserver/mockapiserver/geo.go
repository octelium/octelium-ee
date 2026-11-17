// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockGeoLocation struct {
	countryCode   string
	countryName   string
	continentCode string
	continentName string
	regionCode    string
	regionName    string
	city          string
	lat           float64
	lng           float64
	timezone      string
	offset        float64
	asn           int64
	isp           string
}

var mockGeoLocations = []*mockGeoLocation{
	{"US", "United States", "NA", "North America", "CA", "California", "San Francisco", 37.7749, -122.4194, "America/Los_Angeles", -8, 7922, "Comcast Cable"},
	{"US", "United States", "NA", "North America", "NY", "New York", "New York", 40.7128, -74.006, "America/New_York", -5, 701, "Verizon Business"},
	{"DE", "Germany", "EU", "Europe", "BE", "Berlin", "Berlin", 52.52, 13.405, "Europe/Berlin", 1, 3320, "Deutsche Telekom AG"},
	{"GB", "United Kingdom", "EU", "Europe", "ENG", "England", "London", 51.5074, -0.1278, "Europe/London", 0, 2856, "British Telecommunications"},
	{"IN", "India", "AS", "Asia", "KA", "Karnataka", "Bengaluru", 12.9716, 77.5946, "Asia/Kolkata", 5.5, 24560, "Bharti Airtel"},
	{"US", "United States", "NA", "North America", "TX", "Texas", "Austin", 30.2672, -97.7431, "America/Chicago", -6, 11427, "Charter Communications"},
	{"FR", "France", "EU", "Europe", "IDF", "Île-de-France", "Paris", 48.8566, 2.3522, "Europe/Paris", 1, 3215, "Orange S.A."},
	{"JP", "Japan", "AS", "Asia", "13", "Tokyo", "Tokyo", 35.6762, 139.6503, "Asia/Tokyo", 9, 2516, "KDDI Corporation"},
	{"BR", "Brazil", "SA", "South America", "SP", "São Paulo", "São Paulo", -23.5505, -46.6333, "America/Sao_Paulo", -3, 28573, "Claro NXT"},
	{"CA", "Canada", "NA", "North America", "ON", "Ontario", "Toronto", 43.6532, -79.3832, "America/Toronto", -5, 812, "Rogers Communications"},
	{"SG", "Singapore", "AS", "Asia", "", "", "Singapore", 1.3521, 103.8198, "Asia/Singapore", 8, 9506, "Singtel Fibre"},
	{"AU", "Australia", "OC", "Oceania", "NSW", "New South Wales", "Sydney", -33.8688, 151.2093, "Australia/Sydney", 11, 1221, "Telstra"},
	{"NL", "Netherlands", "EU", "Europe", "NH", "North Holland", "Amsterdam", 52.3676, 4.9041, "Europe/Amsterdam", 1, 1136, "KPN B.V."},
	{"US", "United States", "NA", "North America", "WA", "Washington", "Seattle", 47.6062, -122.3321, "America/Los_Angeles", -8, 20057, "AT&T Mobility"},
	{"AE", "United Arab Emirates", "AS", "Asia", "DU", "Dubai", "Dubai", 25.2048, 55.2708, "Asia/Dubai", 4, 5384, "Emirates Telecommunications"},
	{"ZA", "South Africa", "AF", "Africa", "GP", "Gauteng", "Johannesburg", -26.2041, 28.0473, "Africa/Johannesburg", 2, 37457, "Telkom SA"},
	{"MX", "Mexico", "NA", "North America", "CMX", "Mexico City", "Mexico City", 19.4326, -99.1332, "America/Mexico_City", -6, 8151, "Telmex"},
	{"KR", "South Korea", "AS", "Asia", "11", "Seoul", "Seoul", 37.5665, 126.978, "Asia/Seoul", 9, 4766, "Korea Telecom"},
	{"PL", "Poland", "EU", "Europe", "14", "Mazovia", "Warsaw", 52.2297, 21.0122, "Europe/Warsaw", 1, 5617, "Orange Polska"},
	{"EG", "Egypt", "AF", "Africa", "C", "Cairo", "Cairo", 30.0444, 31.2357, "Africa/Cairo", 2, 8452, "Telecom Egypt"},
}

var mockGeoFailureLocations = []*mockGeoLocation{
	{"RU", "Russia", "EU", "Europe", "MOW", "Moscow", "Moscow", 55.7558, 37.6173, "Europe/Moscow", 3, 12389, "Rostelecom"},
	{"NG", "Nigeria", "AF", "Africa", "LA", "Lagos", "Lagos", 6.5244, 3.3792, "Africa/Lagos", 1, 29465, "MTN Nigeria"},
	{"VN", "Vietnam", "AS", "Asia", "SG", "Ho Chi Minh", "Ho Chi Minh City", 10.8231, 106.6297, "Asia/Ho_Chi_Minh", 7, 7552, "Viettel Group"},
	{"CN", "China", "AS", "Asia", "SH", "Shanghai", "Shanghai", 31.2304, 121.4737, "Asia/Shanghai", 8, 4812, "China Telecom"},
	{"BR", "Brazil", "SA", "South America", "RJ", "Rio de Janeiro", "Rio de Janeiro", -22.9068, -43.1729, "America/Sao_Paulo", -3, 18881, "Telefônica Brasil"},
	{"US", "United States", "NA", "North America", "VA", "Virginia", "Ashburn", 39.0438, -77.4874, "America/New_York", -5, 14618, "Amazon.com"},
}

func (l *mockGeoLocation) toGeoIP(seed int) *corev1.GeoIP {
	ret := &corev1.GeoIP{
		Ip:        fmt.Sprintf("%d.%d.%d.%d", 23+seed%180, 10+seed%200, seed%250, 1+seed%253),
		IpVersion: corev1.GeoIP_V4,
		Country:   &corev1.GeoIP_Country{Code: l.countryCode, Name: l.countryName},
		Continent: &corev1.GeoIP_Continent{Code: l.continentCode, Name: l.continentName},
		City:      &corev1.GeoIP_City{Name: l.city},
		Network: &corev1.GeoIP_Network{
			Asn:          l.asn,
			Isp:          l.isp,
			Organization: l.isp,
		},
		Timezone: &corev1.GeoIP_Timezone{Id: l.timezone, Offset: l.offset},
		Coordinates: &corev1.GeoIP_Coordinates{
			Latitude:       l.lat + float64(seed%7-3)*0.01,
			Longitude:      l.lng + float64(seed%5-2)*0.01,
			AccuracyRadius: float64(5 + seed%4*5),
		},
	}

	if l.regionCode != "" {
		ret.Region = &corev1.GeoIP_Region{Code: l.regionCode, Name: l.regionName}
	}

	return ret
}

func getMockGeoLocation(idx int) *mockGeoLocation {
	weighted := []int{0, 0, 1, 1, 2, 3, 0, 4, 5, 6, 7, 2, 8, 9, 10, 1, 11, 12, 13, 3, 14, 15, 16, 17, 18, 19}
	return mockGeoLocations[weighted[idx%len(weighted)]]
}

func setMockSessionGeoIP(ctx context.Context, octeliumC octeliumc.ClientInterface, sess *corev1.Session, idx int) error {
	if sess.Status.Authentication == nil {
		sess.Status.Authentication = &corev1.Session_Status_Authentication{
			SetAt:   sess.Metadata.CreatedAt,
			TokenID: fmt.Sprintf("mock-token-%d", idx),
		}
	}
	if sess.Status.Authentication.Info == nil {
		sess.Status.Authentication.Info = &corev1.Session_Status_Authentication_Info{
			Type: corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER,
			Aal:  corev1.Session_Status_Authentication_Info_AAL2,
		}
	}
	if sess.Status.Authentication.SetAt == nil {
		sess.Status.Authentication.SetAt = pbutils.Now()
	}

	sess.Status.Authentication.Info.Geoip = getMockGeoLocation(idx).toGeoIP(idx)
	if sess.Status.Authentication.Info.Downstream == nil {
		sess.Status.Authentication.Info.Downstream = &corev1.Session_Status_Authentication_Info_Downstream{}
	}
	sess.Status.Authentication.Info.Downstream.IpAddress = sess.Status.Authentication.Info.Geoip.Ip

	if idx%3 == 0 {
		setAt := sess.Status.Authentication.SetAt.AsTime()
		for i := 1; i <= 3; i++ {
			prev := pbutils.Clone(sess.Status.Authentication).(*corev1.Session_Status_Authentication)
			prevIdx := idx + i*(1+idx%4)
			if i == 1 && idx%2 == 0 {
				prevIdx = idx
			}
			prev.Info.Geoip = getMockGeoLocation(prevIdx).toGeoIP(prevIdx)
			prev.Info.Downstream.IpAddress = prev.Info.Geoip.Ip
			prev.SetAt = pbutils.Timestamp(setAt.Add(-time.Duration(i) * time.Duration(3+idx%5) * time.Hour))
			prev.TokenID = fmt.Sprintf("%s-%d", sess.Status.Authentication.TokenID, i)
			sess.Status.LastAuthentications = append(sess.Status.LastAuthentications, prev)
		}
		sess.Status.TotalAuthentications = uint32(len(sess.Status.LastAuthentications) + 1)
	}

	_, err := octeliumC.CoreC().UpdateSession(ctx, sess)
	return err
}

func getMockAuthLogCountry(itm *enterprisev1.AuthenticationLog) string {
	return itm.Entry.Authentication.GetInfo().GetGeoip().GetCountry().GetCode()
}

func filterMockAuthenticationLogsCountry(items []*enterprisev1.AuthenticationLog,
	code string) []*enterprisev1.AuthenticationLog {
	if code == "" {
		return items
	}

	var ret []*enterprisev1.AuthenticationLog
	for _, itm := range items {
		if strings.EqualFold(getMockAuthLogCountry(itm), code) {
			ret = append(ret, itm)
		}
	}

	return ret
}

func getMockTopLimit(limit uint32) int {
	switch {
	case limit == 0:
		return 10
	case limit > 250:
		return 250
	default:
		return int(limit)
	}
}

func (s *tstVisibilityLog) ListAuthenticationLogTopCountry(ctx context.Context,
	req *visibilityv1.ListAuthenticationLogTopCountryRequest) (*visibilityv1.ListAuthenticationLogTopCountryResponse, error) {
	from, to := getMockAuthLogWindow(req.From, req.To, time.Hour)

	items, err := s.getAuthenticationLogs(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items = filterMockAuthenticationLogs(items, req.Status, req.Reason, req.UserRef)

	type entry struct {
		item     *visibilityv1.ListAuthenticationLogTopCountryResponse_Item
		users    map[string]struct{}
		sessions map[string]struct{}
	}

	byCode := make(map[string]*entry)
	ret := &visibilityv1.ListAuthenticationLogTopCountryResponse{}

	for _, itm := range items {
		if req.SessionRef != nil && itm.Entry.SessionRef.GetUid() != req.SessionRef.Uid {
			continue
		}

		geoip := itm.Entry.Authentication.GetInfo().GetGeoip()
		code := geoip.GetCountry().GetCode()
		if code == "" {
			ret.TotalUnknown++
			continue
		}

		e, ok := byCode[code]
		if !ok {
			e = &entry{
				item: &visibilityv1.ListAuthenticationLogTopCountryResponse_Item{
					Country:   geoip.Country,
					Continent: geoip.Continent,
				},
				users:    make(map[string]struct{}),
				sessions: make(map[string]struct{}),
			}
			byCode[code] = e
		}

		e.item.Count++
		if itm.Entry.Status == enterprisev1.AuthenticationLog_Entry_FAILURE {
			e.item.CountFailure++
		} else {
			e.item.CountSuccess++
		}
		if uid := itm.Entry.UserRef.GetUid(); uid != "" {
			e.users[uid] = struct{}{}
		}
		if uid := itm.Entry.SessionRef.GetUid(); uid != "" {
			e.sessions[uid] = struct{}{}
		}
		if e.item.LastSeenAt == nil || itm.Metadata.CreatedAt.AsTime().After(e.item.LastSeenAt.AsTime()) {
			e.item.LastSeenAt = itm.Metadata.CreatedAt
		}
	}

	var all []*visibilityv1.ListAuthenticationLogTopCountryResponse_Item
	for _, e := range byCode {
		e.item.CountUser = uint64(len(e.users))
		e.item.CountSession = uint64(len(e.sessions))
		all = append(all, e.item)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Count == all[j].Count {
			return all[i].Country.Code < all[j].Country.Code
		}
		return all[i].Count > all[j].Count
	})

	ret.TotalCount = uint64(len(all))
	for idx, itm := range all {
		if idx < getMockTopLimit(req.Limit) {
			ret.Items = append(ret.Items, itm)
		} else {
			ret.TotalOther += itm.Count
		}
	}

	return ret, nil
}

func (s *tstVisibilityLog) ListAuthenticationLogTopCity(ctx context.Context,
	req *visibilityv1.ListAuthenticationLogTopCityRequest) (*visibilityv1.ListAuthenticationLogTopCityResponse, error) {
	from, to := getMockAuthLogWindow(req.From, req.To, time.Hour)

	items, err := s.getAuthenticationLogs(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items = filterMockAuthenticationLogs(items, req.Status, req.Reason, req.UserRef)
	items = filterMockAuthenticationLogsCountry(items, req.CountryCode)

	type entry struct {
		item     *visibilityv1.ListAuthenticationLogTopCityResponse_Item
		lat      float64
		lng      float64
		users    map[string]struct{}
		sessions map[string]struct{}
	}

	byKey := make(map[string]*entry)
	ret := &visibilityv1.ListAuthenticationLogTopCityResponse{}

	for _, itm := range items {
		if req.SessionRef != nil && itm.Entry.SessionRef.GetUid() != req.SessionRef.Uid {
			continue
		}

		geoip := itm.Entry.Authentication.GetInfo().GetGeoip()
		if geoip.GetCoordinates() == nil {
			ret.TotalUnknown++
			continue
		}

		key := fmt.Sprintf("%s/%s", geoip.GetCountry().GetCode(), geoip.GetCity().GetName())
		e, ok := byKey[key]
		if !ok {
			e = &entry{
				item: &visibilityv1.ListAuthenticationLogTopCityResponse_Item{
					Country: geoip.Country,
					Region:  geoip.Region,
					City:    geoip.City,
				},
				users:    make(map[string]struct{}),
				sessions: make(map[string]struct{}),
			}
			byKey[key] = e
		}

		e.item.Count++
		e.lat += geoip.Coordinates.Latitude
		e.lng += geoip.Coordinates.Longitude
		if e.item.Coordinates == nil || geoip.Coordinates.AccuracyRadius > e.item.Coordinates.AccuracyRadius {
			e.item.Coordinates = &corev1.GeoIP_Coordinates{AccuracyRadius: geoip.Coordinates.AccuracyRadius}
		}
		if itm.Entry.Status == enterprisev1.AuthenticationLog_Entry_FAILURE {
			e.item.CountFailure++
		} else {
			e.item.CountSuccess++
		}
		if uid := itm.Entry.UserRef.GetUid(); uid != "" {
			e.users[uid] = struct{}{}
		}
		if uid := itm.Entry.SessionRef.GetUid(); uid != "" {
			e.sessions[uid] = struct{}{}
		}
		if e.item.LastSeenAt == nil || itm.Metadata.CreatedAt.AsTime().After(e.item.LastSeenAt.AsTime()) {
			e.item.LastSeenAt = itm.Metadata.CreatedAt
		}
	}

	var all []*visibilityv1.ListAuthenticationLogTopCityResponse_Item
	for _, e := range byKey {
		e.item.Coordinates.Latitude = e.lat / float64(e.item.Count)
		e.item.Coordinates.Longitude = e.lng / float64(e.item.Count)
		e.item.CountUser = uint64(len(e.users))
		e.item.CountSession = uint64(len(e.sessions))
		all = append(all, e.item)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Count == all[j].Count {
			return all[i].City.GetName() < all[j].City.GetName()
		}
		return all[i].Count > all[j].Count
	})

	ret.TotalCount = uint64(len(all))
	for idx, itm := range all {
		if idx < getMockTopLimit(req.Limit) {
			ret.Items = append(ret.Items, itm)
		} else {
			ret.TotalOther += itm.Count
		}
	}

	return ret, nil
}

func (s *tstVisibilityLog) ListAccessLogTopCountry(ctx context.Context,
	req *visibilityv1.ListAccessLogTopCountryRequest) (*visibilityv1.ListAccessLogTopCountryResponse, error) {
	sessList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	from, to := getMockAuthLogWindow(req.From, req.To, time.Hour)
	scale := uint64(to.Sub(from)/(10*time.Minute)) + 1

	byCode := make(map[string]*visibilityv1.ListAccessLogTopCountryResponse_Item)
	users := make(map[string]map[string]struct{})
	ret := &visibilityv1.ListAccessLogTopCountryResponse{}

	for idx, sess := range sessList.Items {
		if req.SessionRef != nil && sess.Metadata.Uid != req.SessionRef.Uid {
			continue
		}
		if req.UserRef != nil && sess.Status.UserRef.GetUid() != req.UserRef.Uid {
			continue
		}

		count := scale * uint64(3+idx%9)
		denied := count / uint64(6+idx%5)
		if req.Status == corev1.AccessLog_Entry_Common_DENIED {
			count = denied
		} else if req.Status == corev1.AccessLog_Entry_Common_ALLOWED {
			count = count - denied
		}

		code := sess.Status.GetAuthentication().GetInfo().GetGeoip().GetCountry().GetCode()
		if code == "" {
			ret.TotalUnknown += count
			continue
		}

		itm, ok := byCode[code]
		if !ok {
			itm = &visibilityv1.ListAccessLogTopCountryResponse_Item{
				Country:    &corev1.GeoIP_Country{Code: code},
				LastSeenAt: timestamppb.New(to.Add(-time.Duration(idx%30) * time.Minute)),
			}
			byCode[code] = itm
			users[code] = make(map[string]struct{})
		}

		itm.Count += count
		itm.CountDenied += denied
		itm.CountAllowed += count - denied
		itm.CountSession++
		itm.CountService = uint64(2 + len(users[code])%6)
		users[code][sess.Status.UserRef.GetUid()] = struct{}{}
		itm.CountUser = uint64(len(users[code]))
	}

	var all []*visibilityv1.ListAccessLogTopCountryResponse_Item
	for _, itm := range byCode {
		all = append(all, itm)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Count == all[j].Count {
			return all[i].Country.Code < all[j].Country.Code
		}
		return all[i].Count > all[j].Count
	})

	ret.TotalCount = uint64(len(all))
	for idx, itm := range all {
		if idx < getMockTopLimit(req.Limit) {
			ret.Items = append(ret.Items, itm)
		} else {
			ret.TotalOther += itm.Count
		}
	}

	return ret, nil
}
