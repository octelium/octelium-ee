import { AuthenticationLog_Entry_Status } from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAccessLogTopCountryRequest,
  ListAuthenticationLogTopCityRequest,
  ListAuthenticationLogTopCountryRequest,
} from "@/apis/visibilityv1/visibilityv1";
import { useDashboardQuery } from "@/components/Dashboard/utils";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityCore,
} from "@/utils/client";
import { buildTimestamps, refKey, toTs } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

export const GEO_COUNTRY_LIMIT = 250;
export const GEO_CITY_LIMIT = 100;

export type GeoRefs = {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
};

const refsKey = (refs?: GeoRefs) =>
  refs
    ? Object.entries(refs)
        .map(([key, ref]) => `${key}:${refKey(ref) ?? ""}`)
        .join("|")
    : null;

export const useAuthCountries = (
  periodMinutes: number,
  status: AuthenticationLog_Entry_Status = AuthenticationLog_Entry_Status.STATUS_UNSET,
  refs?: GeoRefs,
  priority: number = QUERY_PRIORITY.normal,
) =>
  useDashboardQuery({
    queryKey: ["visibility", "geo", "authCountry", periodMinutes, status, refsKey(refs)],
    priority,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopCountry(
          ListAuthenticationLogTopCountryRequest.create({
            ...refs,
            from: toTs(curFrom),
            to: toTs(curTo),
            status,
            limit: GEO_COUNTRY_LIMIT,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useAuthCities = (
  periodMinutes: number,
  status: AuthenticationLog_Entry_Status = AuthenticationLog_Entry_Status.STATUS_UNSET,
  refs?: GeoRefs,
  priority: number = QUERY_PRIORITY.low,
) =>
  useDashboardQuery({
    queryKey: ["visibility", "geo", "authCity", periodMinutes, status, refsKey(refs)],
    priority,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopCity(
          ListAuthenticationLogTopCityRequest.create({
            ...refs,
            from: toTs(curFrom),
            to: toTs(curTo),
            status,
            limit: GEO_CITY_LIMIT,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useAccessCountries = (
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.normal,
) =>
  useDashboardQuery({
    queryKey: ["visibility", "geo", "accessCountry", periodMinutes],
    priority,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAccessLog().listAccessLogTopCountry(
          ListAccessLogTopCountryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            limit: GEO_COUNTRY_LIMIT,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useSessionCountries = (priority: number = QUERY_PRIORITY.normal) =>
  useDashboardQuery({
    queryKey: ["visibility", "geo", "sessionCountry"],
    priority,
    fetch: async (signal) =>
      (await getClientVisibilityCore().getSessionSummary({}, { abort: signal }))
        .response,
  });
