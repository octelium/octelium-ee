import * as CoreP from "@/apis/corev1/corev1";
import {
  distanceKm,
  formatDistance,
  getCountryName,
  hasCoordinates,
  type LatLng,
} from "@/components/GeoMap/geo";
import WorldMap, {
  type WorldMapPoint,
  type WorldMapRoute,
} from "@/components/GeoMap/WorldMap";
import { formatSeconds, toDate, TonePill } from "@/components/ResourceLayout/overview";
import { CountryFlag } from "@/components/SelectCountry/CountryFlag";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import * as React from "react";

export const MAX_TRAVEL_KMH = 1000;
const MIN_TRAVEL_KM = 500;

export type SessionTrailEntry = {
  key: string;
  auth: CoreP.Session_Status_Authentication;
  at?: Date;
  geoip?: CoreP.GeoIP;
  point?: LatLng;
  label: string;
  isCurrent: boolean;
  fromPrevious?: {
    km: number;
    seconds: number;
    kmh?: number;
    isImpossible: boolean;
    isNewCountry: boolean;
  };
};

const getLabel = (geoip?: CoreP.GeoIP) => {
  if (!geoip) return "Unknown location";
  const code = geoip.country?.code;
  const country = code
    ? getCountryName(code, geoip.country?.name)
    : geoip.country?.name;
  return (
    [geoip.city?.name, country].filter(Boolean).join(", ") ||
    geoip.ip ||
    "Unknown location"
  );
};

export const getSessionTrail = (item: CoreP.Session): SessionTrailEntry[] => {
  const status = item.status;
  const current = status?.authentication;
  const seen = new Set<string>();
  const ret: SessionTrailEntry[] = [];

  for (const auth of [
    current,
    ...(status?.lastAuthentications ?? []),
    status?.initialAuthentication,
  ]) {
    if (!auth?.info?.geoip) continue;
    const at = toDate(auth.setAt);
    const key = `${auth.tokenID}-${at?.getTime() ?? ""}`;
    if (seen.has(key)) continue;
    seen.add(key);

    const geoip = auth.info.geoip;
    const coordinates = geoip.coordinates;
    ret.push({
      key,
      auth,
      at,
      geoip,
      point: hasCoordinates(coordinates)
        ? { lat: coordinates!.latitude, lng: coordinates!.longitude }
        : undefined,
      label: getLabel(geoip),
      isCurrent: auth === current,
    });
  }

  ret.sort((a, b) => (a.at?.getTime() ?? 0) - (b.at?.getTime() ?? 0));

  let previous: SessionTrailEntry | undefined;
  for (const entry of ret) {
    if (previous?.point && entry.point && previous.at && entry.at) {
      const km = distanceKm(previous.point, entry.point);
      const seconds = Math.max(0, (entry.at.getTime() - previous.at.getTime()) / 1000);
      const kmh = seconds > 0 ? km / (seconds / 3600) : undefined;
      entry.fromPrevious = {
        km,
        seconds,
        kmh,
        isImpossible:
          km >= MIN_TRAVEL_KM && (kmh === undefined || kmh > MAX_TRAVEL_KMH),
        isNewCountry:
          !!previous.geoip?.country?.code &&
          !!entry.geoip?.country?.code &&
          previous.geoip.country.code !== entry.geoip.country.code,
      };
    }
    if (entry.point) previous = entry;
  }

  return ret;
};

export const getImpossibleTravel = (item: CoreP.Session) =>
  getSessionTrail(item).filter((entry) => entry.fromPrevious?.isImpossible);

const SessionLocations = (props: { item: CoreP.Session }) => {
  const trail = React.useMemo(() => getSessionTrail(props.item), [props.item]);
  const [focus, setFocus] = React.useState<string>();

  const located = trail.filter((entry) => entry.point);
  const countries = React.useMemo(() => {
    const ret = new Map<string, number>();
    for (const entry of trail) {
      const code = entry.geoip?.country?.code?.toUpperCase();
      if (code) ret.set(code, (ret.get(code) ?? 0) + 1);
    }
    return [...ret.entries()].map(([code, value]) => ({ code, value }));
  }, [trail]);

  const points: WorldMapPoint[] = located.map((entry) => ({
    id: entry.key,
    lat: entry.point!.lat,
    lng: entry.point!.lng,
    value: entry.isCurrent ? 2 : 1,
    current: entry.isCurrent,
    tone: entry.fromPrevious?.isImpossible
      ? "danger"
      : entry.isCurrent
        ? "primary"
        : "warning",
  }));

  const routes: WorldMapRoute[] = [];
  for (let i = 1; i < located.length; i++) {
    routes.push({
      id: `${located[i - 1].key}->${located[i].key}`,
      from: located[i - 1].point!,
      to: located[i].point!,
      tone: located[i].fromPrevious?.isImpossible ? "danger" : "primary",
    });
  }

  if (trail.length === 0) return null;

  const byKey = new Map(trail.map((entry) => [entry.key, entry]));
  const longest = Math.max(0, ...trail.map((entry) => entry.fromPrevious?.km ?? 0));
  const impossible = trail.filter((entry) => entry.fromPrevious?.isImpossible);

  return (
    <div className="flex w-full min-w-0 flex-col gap-3">
      <div className="flex flex-wrap items-center gap-1.5">
        <TonePill>
          {located.length} {located.length === 1 ? "location" : "locations"}
        </TonePill>
        <TonePill>
          {countries.length} {countries.length === 1 ? "country" : "countries"}
        </TonePill>
        {longest > 0 && <TonePill>Longest hop {formatDistance(longest)}</TonePill>}
        {impossible.length > 0 && (
          <TonePill tone="danger">
            {impossible.length === 1
              ? "Impossible travel"
              : `${impossible.length} impossible travels`}
          </TonePill>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_16rem]">
        {located.length > 0 && (
          <WorldMap
            label="Map of the Session's sign-in locations"
            points={points}
            routes={routes}
            fit="points"
            selected={focus ? byKey.get(focus)?.geoip?.country?.code : undefined}
            renderTooltip={(hover) => {
              const entry = hover.point ? byKey.get(hover.point.id) : undefined;
              if (!entry) {
                return hover.code ? getCountryName(hover.code) : null;
              }
              return (
                <div className="flex flex-col gap-0.5">
                  <span className="flex items-center gap-1.5 font-semibold">
                    <CountryFlag
                      code={entry.geoip?.country?.code}
                      className="h-3 w-4 shrink-0 rounded-[2px]"
                    />
                    {entry.label}
                  </span>
                  {entry.geoip?.ip && (
                    <span className="font-mono opacity-80">{entry.geoip.ip}</span>
                  )}
                  {entry.auth.setAt && (
                    <span className="opacity-80">
                      {entry.isCurrent ? "Current sign-in" : "Signed in"}{" "}
                      <TimeAgo rfc3339={entry.auth.setAt} />
                    </span>
                  )}
                </div>
              );
            }}
            className="rounded-lg border border-slate-200 bg-slate-50/60"
          />
        )}

        <ol
          className={twMerge(
            "flex min-w-0 flex-col gap-1",
            located.length === 0 && "lg:col-span-2",
          )}
          aria-label="Sign-in locations, newest first"
          onMouseLeave={() => setFocus(undefined)}
        >
          {[...trail].reverse().map((entry) => {
            const hop = entry.fromPrevious;
            return (
              <li
                key={entry.key}
                onMouseEnter={() => setFocus(entry.key)}
                className={twMerge(
                  "flex min-w-0 flex-col gap-0.5 rounded-lg border border-transparent px-2.5 py-1.5 transition-colors",
                  focus === entry.key && "border-slate-200 bg-slate-50",
                )}
              >
                <span className="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-slate-700">
                  <CountryFlag
                    code={entry.geoip?.country?.code}
                    className="h-3 w-4 shrink-0 rounded-[2px] ring-1 ring-slate-200"
                  />
                  <span className="truncate">{entry.label}</span>
                </span>
                <span className="flex flex-wrap items-center gap-x-2 text-micro font-normal text-slate-500">
                  {entry.isCurrent && (
                    <span className="font-semibold text-emerald-700">Current</span>
                  )}
                  {entry.geoip?.ip && (
                    <span className="font-mono">{entry.geoip.ip}</span>
                  )}
                  {entry.auth.setAt && <TimeAgo rfc3339={entry.auth.setAt} />}
                </span>
                {hop && hop.km >= 1 && (
                  <span
                    className={twMerge(
                      "text-micro font-semibold",
                      hop.isImpossible ? "text-red-700" : "text-slate-500",
                    )}
                  >
                    {formatDistance(hop.km)} from previous in{" "}
                    {formatSeconds(hop.seconds)}
                    {hop.isImpossible && " · faster than any flight"}
                  </span>
                )}
              </li>
            );
          })}
        </ol>
      </div>
    </div>
  );
};

export default SessionLocations;
