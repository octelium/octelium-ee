import { AuthenticationLog_Entry_Status } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Panel } from "@/components/Dashboard/components";
import { compact, exact } from "@/components/Dashboard/utils";
import {
  getCountryName,
  hasCoordinates,
} from "@/components/GeoMap/geo";
import WorldMap, {
  getToneColor,
  MapLegend,
  type WorldMapPoint,
  type WorldMapTone,
} from "@/components/GeoMap/WorldMap";
import { CountryFlag } from "@/components/SelectCountry/CountryFlag";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { useChartColorScheme } from "@/utils/charts/palette";
import { periodLabel } from "@/utils/visibility";
import { SegmentedControl } from "@mantine/core";
import { ChevronDown, Earth, LucideIcon, MapPinOff } from "lucide-react";
import * as React from "react";
import { Link, useNavigate } from "react-router";
import {
  GeoRefs,
  useAccessCountries,
  useAuthCities,
  useAuthCountries,
  useSessionCountries,
} from "./queries";

export type GeoSource = "signins" | "failures" | "requests" | "sessions";

export type GeoSegment = {
  label: string;
  value: number;
  tone: WorldMapTone;
};

export type GeoCountryRow = {
  code: string;
  name: string;
  value: number;
  segments?: GeoSegment[];
  meta?: string[];
  lastSeenAt?: Timestamp;
};

export type GeoCityPoint = WorldMapPoint & {
  label: string;
  countryCode?: string;
  meta?: string[];
  lastSeenAt?: Timestamp;
};

export type GeoView = {
  rows: GeoCountryRow[];
  points?: GeoCityPoint[];
  unknown: number;
  totalCountries: number;
  totalCities?: number;
  tone: WorldMapTone;
  noun: string;
  note: string;
  to?: (code: string) => string;
};

const SOURCES: Record<
  GeoSource,
  { label: string; noun: string; tone: WorldMapTone }
> = {
  signins: { label: "Sign-ins", noun: "sign-ins", tone: "primary" },
  failures: { label: "Failed", noun: "failed sign-ins", tone: "danger" },
  requests: { label: "Requests", noun: "requests", tone: "primary" },
  sessions: { label: "Sessions", noun: "Sessions", tone: "primary" },
};

const plural = (value: number, noun: string) =>
  `${compact(value)} ${value === 1 ? noun.replace(/s$/, "") : noun}`;

const cityCount = (value: number) =>
  `${compact(value)} ${value === 1 ? "city" : "cities"}`;

export const toCountryRows = <
  T extends {
    country?: { code: string; name: string };
    count: number;
    lastSeenAt?: Timestamp;
  },
>(
  items: T[] | undefined,
  build?: (item: T) => Pick<GeoCountryRow, "segments" | "meta">,
): GeoCountryRow[] =>
  (items ?? [])
    .filter((item) => !!item.country?.code)
    .map((item) => ({
      code: item.country!.code.toUpperCase(),
      name: getCountryName(item.country!.code, item.country!.name),
      value: Number(item.count),
      lastSeenAt: item.lastSeenAt,
      ...build?.(item),
    }))
    .sort((a, b) => b.value - a.value || a.name.localeCompare(b.name));

export const toCityPoints = (
  items:
    | {
        country?: { code: string; name: string };
        region?: { name: string };
        city?: { name: string };
        coordinates?: { latitude: number; longitude: number };
        count: number;
        countFailure: number;
        countSuccess: number;
        countUser: number;
        lastSeenAt?: Timestamp;
      }[]
    | undefined,
  tone: WorldMapTone,
): GeoCityPoint[] =>
  (items ?? [])
    .filter((item) => hasCoordinates(item.coordinates))
    .map((item, index) => {
      const country = getCountryName(item.country?.code, item.country?.name);
      const city = item.city?.name;
      return {
        id: `${item.country?.code ?? ""}/${city ?? index}`,
        lat: item.coordinates!.latitude,
        lng: item.coordinates!.longitude,
        value: Number(item.count),
        tone:
          tone === "primary" &&
          Number(item.countFailure) > Number(item.countSuccess)
            ? "danger"
            : tone,
        label: city ? `${city}, ${country}` : country,
        countryCode: item.country?.code,
        lastSeenAt: item.lastSeenAt,
        meta: [
          Number(item.countFailure) > 0 && tone !== "danger"
            ? `${compact(Number(item.countFailure))} failed`
            : undefined,
          Number(item.countUser) > 0
            ? plural(Number(item.countUser), "users")
            : undefined,
        ].filter(Boolean) as string[],
      };
    });

const SegmentBar = (props: {
  segments?: GeoSegment[];
  value: number;
  max: number;
  tone: WorldMapTone;
}) => {
  const width = props.max > 0 ? (props.value / props.max) * 100 : 0;
  const segments =
    props.segments?.filter((item) => item.value > 0) ?? [
      { label: "", value: props.value, tone: props.tone },
    ];
  const total = segments.reduce((acc, item) => acc + item.value, 0) || 1;

  return (
    <span className="mt-1 flex h-1 w-full overflow-hidden rounded-full bg-slate-100">
      <span
        className="flex h-full overflow-hidden rounded-full transition-[width] duration-300"
        style={{ width: `${Math.max(width, props.value > 0 ? 3 : 0)}%` }}
      >
        {segments.map((item) => (
          <span
            key={item.label}
            className="h-full"
            style={{
              width: `${(item.value / total) * 100}%`,
              backgroundColor: getToneColor(item.tone),
            }}
          />
        ))}
      </span>
    </span>
  );
};

const CountryTooltip = (props: { row: GeoCountryRow; noun: string }) => (
  <div className="flex flex-col gap-1">
    <span className="flex items-center gap-1.5 font-semibold">
      <CountryFlag
        code={props.row.code}
        className="h-3 w-4 shrink-0 rounded-[2px]"
      />
      {props.row.name}
    </span>
    <span className="tabular-nums">{plural(props.row.value, props.noun)}</span>
    {props.row.segments && props.row.segments.some((item) => item.value > 0) && (
      <span className="flex flex-wrap gap-x-2.5 tabular-nums opacity-80">
        {props.row.segments
          .filter((item) => item.value > 0)
          .map((item) => (
            <span key={item.label} className="inline-flex items-center gap-1">
              <span
                className="h-1.5 w-1.5 rounded-full"
                style={{ backgroundColor: getToneColor(item.tone) }}
              />
              {compact(item.value)} {item.label}
            </span>
          ))}
      </span>
    )}
    {props.row.meta && props.row.meta.length > 0 && (
      <span className="opacity-80">{props.row.meta.join(" · ")}</span>
    )}
    {props.row.lastSeenAt && (
      <span className="opacity-70">
        Last seen <TimeAgo rfc3339={props.row.lastSeenAt} />
      </span>
    )}
  </div>
);

const PointTooltip = (props: { point: GeoCityPoint; noun: string }) => (
  <div className="flex flex-col gap-1">
    <span className="flex items-center gap-1.5 font-semibold">
      <CountryFlag
        code={props.point.countryCode}
        className="h-3 w-4 shrink-0 rounded-[2px]"
      />
      {props.point.label}
    </span>
    {props.point.value !== undefined && (
      <span className="tabular-nums">
        {plural(props.point.value, props.noun)}
      </span>
    )}
    {props.point.meta && props.point.meta.length > 0 && (
      <span className="opacity-80">{props.point.meta.join(" · ")}</span>
    )}
    {props.point.lastSeenAt && (
      <span className="opacity-70">
        Last seen <TimeAgo rfc3339={props.point.lastSeenAt} />
      </span>
    )}
  </div>
);

const GeoStat = (props: {
  label: string;
  value: React.ReactNode;
  hint?: React.ReactNode;
}) => (
  <div className="flex min-w-0 flex-col gap-0.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
    <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {props.label}
    </span>
    <span className="flex min-w-0 items-center gap-1.5 truncate text-lg font-bold leading-6 tabular-nums text-slate-800">
      {props.value}
    </span>
    {props.hint && (
      <span className="truncate text-micro font-normal text-slate-500">
        {props.hint}
      </span>
    )}
  </div>
);

const LIST_COUNT = 8;

const NO_ROWS: GeoCountryRow[] = [];

export const GeoCountryList = (props: {
  view: GeoView;
  focus?: string;
  onFocus: (code?: string) => void;
}) => {
  const { view } = props;
  const [expanded, setExpanded] = React.useState(false);
  const max = view.rows[0]?.value ?? 0;
  const rows = expanded ? view.rows : view.rows.slice(0, LIST_COUNT);

  if (view.rows.length === 0) return null;

  return (
    <div className="flex min-w-0 flex-col">
      <span className="mb-1.5 px-2.5 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
        Top countries
      </span>
      <ul
        className={twMerge(
          "flex flex-col gap-0.5",
          expanded && "max-h-96 overflow-y-auto overscroll-contain",
        )}
        onMouseLeave={() => props.onFocus(undefined)}
      >
        {rows.map((row) => {
          const content = (
            <>
              <CountryFlag
                code={row.code}
                className="h-3.5 w-[1.17rem] shrink-0 rounded-[2px] ring-1 ring-slate-200"
              />
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="flex min-w-0 items-baseline justify-between gap-2">
                  <span className="truncate text-xs font-semibold text-slate-700">
                    {row.name}
                  </span>
                  <span
                    className="shrink-0 text-xs font-bold tabular-nums text-slate-800"
                    title={exact(row.value)}
                  >
                    {compact(row.value)}
                  </span>
                </span>
                <SegmentBar
                  segments={row.segments}
                  value={row.value}
                  max={max}
                  tone={view.tone}
                />
                {row.meta && row.meta.length > 0 && (
                  <span className="mt-0.5 truncate text-micro font-normal text-slate-500">
                    {row.meta.join(" · ")}
                  </span>
                )}
              </span>
            </>
          );
          const className = twMerge(
            "flex w-full min-w-0 items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left outline-none transition-colors duration-150",
            "hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-slate-400",
            props.focus === row.code && "bg-slate-50",
          );

          return (
            <li
              key={row.code}
              onMouseEnter={() => props.onFocus(row.code)}
              onFocus={() => props.onFocus(row.code)}
              onBlur={() => props.onFocus(undefined)}
            >
              {view.to ? (
                <Link
                  to={view.to(row.code)}
                  className={className}
                  aria-label={`${row.name}: ${plural(row.value, view.noun)}`}
                >
                  {content}
                </Link>
              ) : (
                <div className={className}>{content}</div>
              )}
            </li>
          );
        })}
      </ul>
      {view.rows.length > LIST_COUNT && (
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          className="mt-1 inline-flex items-center gap-1 self-start rounded-md px-2.5 py-1 text-micro font-semibold text-slate-500 outline-none transition-colors hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          {expanded
            ? "Show fewer"
            : `Show all ${view.rows.length.toLocaleString()} countries`}
          <ChevronDown
            size={12}
            strokeWidth={2.5}
            className={twMerge("transition-transform", expanded && "rotate-180")}
          />
        </button>
      )}
    </div>
  );
};

export const GeoActivityBody = (props: {
  view?: GeoView;
  isLoading?: boolean;
  isError?: boolean;
  rangeLabel?: string;
  compact?: boolean;
  fit?: "world" | "points";
  emptyHint?: React.ReactNode;
}) => {
  useChartColorScheme();
  const navigate = useNavigate();
  const [focus, setFocus] = React.useState<string>();
  const view = props.view;
  const rows = view?.rows ?? NO_ROWS;
  const byCode = React.useMemo(
    () => new Map(rows.map((row) => [row.code, row])),
    [rows],
  );
  const pointByID = React.useMemo(
    () => new Map((view?.points ?? []).map((point) => [point.id, point])),
    [view?.points],
  );

  const located = rows.reduce((acc, row) => acc + row.value, 0);
  const total = located + (view?.unknown ?? 0);
  const top = rows[0];
  const rangeSuffix = props.rangeLabel ? ` in the last ${props.rangeLabel}` : "";
  const noun = view?.noun ?? "events";

  const overlay =
    !props.isLoading && !props.isError && view && rows.length === 0 ? (
      <div className="pointer-events-auto flex max-w-sm flex-col items-center gap-1.5 rounded-xl border border-slate-200 bg-white/90 px-4 py-3 text-center shadow-card backdrop-blur-sm">
        <MapPinOff size={16} className="text-slate-500" />
        <p className="text-xs font-semibold text-slate-700">
          No located {noun}
          {rangeSuffix}
        </p>
        {view.unknown > 0 && (
          <p className="text-micro font-normal text-slate-500">
            {plural(view.unknown, noun)} without GeoIP information.
          </p>
        )}
        {props.emptyHint}
      </div>
    ) : props.isError ? (
      <div className="pointer-events-auto rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-xs font-semibold text-red-700">
        Geography could not be loaded.
      </div>
    ) : undefined;

  return (
    <div className="flex flex-col gap-4">
      {!props.compact && (
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          <GeoStat
            label="Countries"
            value={compact(view?.totalCountries ?? 0)}
            hint={
              view?.totalCities
                ? cityCount(view.totalCities)
                : `with ${noun}`
            }
          />
          <GeoStat
            label="Top country"
            value={
              top ? (
                <>
                  <CountryFlag
                    code={top.code}
                    className="h-3.5 w-[1.17rem] shrink-0 rounded-[2px] ring-1 ring-slate-200"
                  />
                  <span className="truncate text-body">{top.name}</span>
                </>
              ) : (
                "—"
              )
            }
            hint={
              top && total > 0
                ? `${Math.round((top.value / total) * 100)}% of ${noun}`
                : undefined
            }
          />
          <GeoStat
            label="Located"
            value={total > 0 ? `${Math.round((located / total) * 100)}%` : "—"}
            hint={`${compact(located)} of ${compact(total)} ${noun}`}
          />
          <GeoStat
            label="No location"
            value={compact(view?.unknown ?? 0)}
            hint="Without GeoIP information"
          />
        </div>
      )}

      <div
        className={twMerge(
          "grid grid-cols-1 gap-4",
          rows.length > 0 && "xl:grid-cols-[minmax(0,1fr)_17rem]",
        )}
      >
        <div className="flex min-w-0 flex-col gap-2">
          <WorldMap
            label={`Map of ${noun} by country${rangeSuffix}`}
            countries={rows}
            points={view?.points}
            tone={view?.tone}
            selected={focus}
            fit={props.fit}
            onSelect={
              view?.to ? (code) => navigate(view.to!(code)) : undefined
            }
            describeCountry={(code) => {
              const row = byCode.get(code);
              return row
                ? `${row.name}: ${plural(row.value, noun)}`
                : getCountryName(code);
            }}
            renderTooltip={(hover) => {
              if (hover.point) {
                const point = pointByID.get(hover.point.id);
                return point ? <PointTooltip point={point} noun={noun} /> : null;
              }
              const row = hover.code ? byCode.get(hover.code) : undefined;
              return row ? <CountryTooltip row={row} noun={noun} /> : null;
            }}
            overlay={overlay}
            className={twMerge(
              "rounded-lg",
              props.isLoading && !view && "opacity-60",
            )}
          />
          <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
            <MapLegend tone={view?.tone} />
            {view?.note && (
              <span className="text-micro font-normal text-slate-500">
                {view.note}
              </span>
            )}
          </div>
        </div>

        {view && rows.length > 0 && (
          <GeoCountryList view={view} focus={focus} onFocus={setFocus} />
        )}
      </div>
    </div>
  );
};

const authStatus = (source: GeoSource) =>
  source === "failures"
    ? AuthenticationLog_Entry_Status.FAILURE
    : AuthenticationLog_Entry_Status.SUCCESS;

export const useAuthGeoView = (
  periodMinutes: number,
  source: "signins" | "failures" | "all",
  refs?: GeoRefs,
  to?: (code: string) => string,
) => {
  const status =
    source === "all"
      ? AuthenticationLog_Entry_Status.STATUS_UNSET
      : authStatus(source);
  const countries = useAuthCountries(periodMinutes, status, refs);
  const cities = useAuthCities(periodMinutes, status, refs);
  const tone: WorldMapTone = source === "failures" ? "danger" : "primary";
  const noun =
    source === "failures"
      ? SOURCES.failures.noun
      : source === "all"
        ? "authentications"
        : SOURCES.signins.noun;

  const view = React.useMemo<GeoView | undefined>(() => {
    if (!countries.data) return undefined;
    return {
      rows: toCountryRows(countries.data.items, (item) => ({
        segments:
          source === "all"
            ? [
                {
                  label: "succeeded",
                  value: Number(item.countSuccess),
                  tone: "primary" as const,
                },
                {
                  label: "failed",
                  value: Number(item.countFailure),
                  tone: "danger" as const,
                },
              ]
            : undefined,
        meta: [
          Number(item.countUser) > 0
            ? plural(Number(item.countUser), "users")
            : undefined,
          Number(item.countSession) > 0
            ? plural(Number(item.countSession), "Sessions")
            : undefined,
        ].filter(Boolean) as string[],
      })),
      points: toCityPoints(cities.data?.items, tone),
      unknown: Number(countries.data.totalUnknown),
      totalCountries: Number(countries.data.totalCount),
      totalCities: cities.data ? Number(cities.data.totalCount) : undefined,
      tone,
      noun,
      note: "Resolved from GeoIP at authentication time",
      to,
    };
  }, [countries.data, cities.data, source, tone, noun, to]);

  return {
    view,
    isLoading: countries.isLoading,
    isError: countries.isError,
  };
};

const toAuthLogs = (status: "success" | "failure") => (code: string) =>
  `/visibility/authenticationlogs?status=${status}&countryCode=${code}`;

const toSuccessLogs = toAuthLogs("success");
const toFailureLogs = toAuthLogs("failure");
const toAccessLogs = (code: string) => `/visibility/accesslogs?countryCode=${code}`;
const toSessions = (code: string) => `/core/sessions?countryCode=${code}`;

const AuthSourceView = (props: {
  periodMinutes: number;
  source: "signins" | "failures";
  compact?: boolean;
}) => {
  const ret = useAuthGeoView(
    props.periodMinutes,
    props.source,
    undefined,
    props.source === "failures" ? toFailureLogs : toSuccessLogs,
  );

  return (
    <GeoActivityBody
      {...ret}
      compact={props.compact}
      rangeLabel={periodLabel(props.periodMinutes)}
      emptyHint={
        ret.view && ret.view.unknown > 0 ? (
          <Link
            to="/core/clusterconfig"
            className="text-micro font-semibold text-slate-700 underline-offset-2 hover:underline"
          >
            Configure geolocation in the ClusterConfig
          </Link>
        ) : undefined
      }
    />
  );
};

const RequestSourceView = (props: {
  periodMinutes: number;
  compact?: boolean;
}) => {
  const countries = useAccessCountries(props.periodMinutes);

  const view = React.useMemo<GeoView | undefined>(() => {
    if (!countries.data) return undefined;
    return {
      rows: toCountryRows(countries.data.items, (item) => ({
        segments: [
          {
            label: "allowed",
            value: Number(item.countAllowed),
            tone: "primary" as const,
          },
          {
            label: "denied",
            value: Number(item.countDenied),
            tone: "danger" as const,
          },
        ],
        meta: [
          Number(item.countUser) > 0
            ? plural(Number(item.countUser), "users")
            : undefined,
          Number(item.countService) > 0
            ? plural(Number(item.countService), "Services")
            : undefined,
        ].filter(Boolean) as string[],
      })),
      unknown: Number(countries.data.totalUnknown),
      totalCountries: Number(countries.data.totalCount),
      tone: "primary",
      noun: SOURCES.requests.noun,
      note: "Located by the requesting Session's latest sign-in",
      to: toAccessLogs,
    };
  }, [countries.data]);

  return (
    <GeoActivityBody
      view={view}
      isLoading={countries.isLoading}
      isError={countries.isError}
      compact={props.compact}
      rangeLabel={periodLabel(props.periodMinutes)}
    />
  );
};

const SessionSourceView = (props: { compact?: boolean }) => {
  const summary = useSessionCountries();

  const view = React.useMemo<GeoView | undefined>(() => {
    if (!summary.data) return undefined;
    const rows = Object.entries(summary.data.totalByCountry ?? {})
      .map(([code, value]) => ({
        code: code.toUpperCase(),
        name: getCountryName(code),
        value: Number(value),
      }))
      .filter((row) => row.value > 0)
      .sort((a, b) => b.value - a.value || a.name.localeCompare(b.name));
    const located = rows.reduce((acc, row) => acc + row.value, 0);

    return {
      rows,
      unknown: Math.max(0, Number(summary.data.totalNumber) - located),
      totalCountries: rows.length,
      tone: "primary",
      noun: SOURCES.sessions.noun,
      note: "Current Sessions by their latest sign-in",
      to: toSessions,
    };
  }, [summary.data]);

  return (
    <GeoActivityBody
      view={view}
      isLoading={summary.isLoading}
      isError={summary.isError}
      compact={props.compact}
    />
  );
};

const GeoActivity = (props: {
  periodMinutes: number;
  sources: GeoSource[];
  title?: string;
  description?: string;
  icon?: LucideIcon;
  compact?: boolean;
}) => {
  const [source, setSource] = React.useState<GeoSource>(props.sources[0]);
  const rangeLabel = periodLabel(props.periodMinutes);

  return (
    <Panel
      icon={props.icon ?? Earth}
      title={props.title ?? "Geography"}
      description={
        props.description ??
        (source === "sessions"
          ? "Where the current Sessions signed in from"
          : `Where ${SOURCES[source].noun} came from over the last ${rangeLabel}`)
      }
      actions={
        props.sources.length > 1 ? (
          <SegmentedControl
            size="xs"
            value={source}
            onChange={(value) => setSource(value as GeoSource)}
            aria-label="Geography source"
            data={props.sources.map((item) => ({
              value: item,
              label: SOURCES[item].label,
            }))}
          />
        ) : undefined
      }
    >
      {source === "requests" ? (
        <RequestSourceView
          periodMinutes={props.periodMinutes}
          compact={props.compact}
        />
      ) : source === "sessions" ? (
        <SessionSourceView compact={props.compact} />
      ) : (
        <AuthSourceView
          key={source}
          periodMinutes={props.periodMinutes}
          source={source}
          compact={props.compact}
        />
      )}
    </Panel>
  );
};

export default GeoActivity;

export const ResourceGeoActivity = (props: {
  periodMinutes: number;
  refs: GeoRefs;
}) => {
  const params = React.useMemo(() => {
    const ret = new URLSearchParams();
    Object.entries(props.refs).forEach(([key, ref]) => {
      const value = ref?.name || ref?.uid;
      if (value) ret.set(`${key}.${ref?.name ? "name" : "uid"}`, value);
    });
    return ret.toString();
  }, [props.refs]);
  const to = React.useCallback(
    (code: string) =>
      `/visibility/authenticationlogs?countryCode=${code}${params ? `&${params}` : ""}`,
    [params],
  );
  const ret = useAuthGeoView(props.periodMinutes, "all", props.refs, to);

  if (!ret.view || ret.view.rows.length === 0) return null;

  return (
    <section className="flex w-full flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-raised">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-sm font-bold text-slate-800">Sign-in locations</h2>
          <p className="mt-1 text-xs font-medium text-slate-500">
            Where the authentications came from over the last{" "}
            {periodLabel(props.periodMinutes)}.
          </p>
        </div>
      </div>
      <GeoActivityBody
        {...ret}
        compact
        fit="points"
        rangeLabel={periodLabel(props.periodMinutes)}
      />
    </section>
  );
};
