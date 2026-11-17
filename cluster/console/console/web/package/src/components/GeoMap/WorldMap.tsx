import { twMerge } from "@/lib/utils";
import {
  CHART_INK,
  STATUS_COLORS,
  seriesColor,
  useChartColorScheme,
} from "@/utils/charts/palette";
import * as React from "react";
import {
  loadWorldMap,
  projectEqualEarth,
  type LatLng,
  type WorldMapData,
} from "./geo";

export type WorldMapTone = "primary" | "danger" | "warning" | "success";

export type WorldMapCountry = {
  code: string;
  value: number;
};

export type WorldMapPoint = LatLng & {
  id: string;
  value?: number;
  tone?: WorldMapTone;
  current?: boolean;
};

export type WorldMapRoute = {
  id: string;
  from: LatLng;
  to: LatLng;
  tone?: WorldMapTone;
};

type Hover = {
  code?: string;
  point?: WorldMapPoint;
  left: number;
  top: number;
  flipX: boolean;
  flipY: boolean;
};

const NO_POINTS: WorldMapPoint[] = [];
const NO_ROUTES: WorldMapRoute[] = [];

type ViewBox = { x: number; y: number; w: number; h: number };

const WIDTH = 1000;
const HEIGHT = 440;

export const getToneColor = (tone: WorldMapTone = "primary") => {
  switch (tone) {
    case "danger":
      return STATUS_COLORS.critical;
    case "warning":
      return STATUS_COLORS.warning;
    case "success":
      return STATUS_COLORS.good;
    default:
      return seriesColor(0);
  }
};

export const getIntensity = (value: number, max: number) =>
  value <= 0 || max <= 0
    ? 0
    : 0.22 + 0.78 * (Math.log1p(value) / Math.log1p(max));

export const getFitViewBox = (
  map: Pick<WorldMapData, "width" | "height" | "scale" | "translate">,
  points: LatLng[],
): ViewBox => {
  const full = { x: 0, y: 0, w: map.width, h: map.height };
  if (points.length === 0) return full;

  const xy = points.map((point) => projectEqualEarth(map, point));
  const xs = xy.map(([x]) => x);
  const ys = xy.map(([, y]) => y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const aspect = map.width / map.height;
  const pad = 36;

  let w = Math.max(maxX - minX + pad * 2, 170);
  let h = Math.max(maxY - minY + pad * 2, 170 / aspect);
  if (w / h > aspect) h = w / aspect;
  else w = h * aspect;
  if (w >= map.width) return full;

  const x = Math.min(Math.max((minX + maxX) / 2 - w / 2, 0), map.width - w);
  const y = Math.min(Math.max((minY + maxY) / 2 - h / 2, 0), map.height - h);

  return { x, y, w, h };
};

const useWorldMap = () => {
  const [data, setData] = React.useState<WorldMapData>();
  const [error, setError] = React.useState(false);

  React.useEffect(() => {
    let active = true;
    loadWorldMap()
      .then((ret) => active && setData(ret))
      .catch(() => active && setError(true));
    return () => {
      active = false;
    };
  }, []);

  return { data, error };
};

const LandLayer = React.memo((props: { map: WorldMapData }) => (
  <g className="fill-slate-200 stroke-white" aria-hidden="true">
    {props.map.countries.map(([code, d]) =>
      d ? (
        <path
          key={code}
          d={d}
          strokeWidth={0.6}
          vectorEffect="non-scaling-stroke"
          strokeLinejoin="round"
        />
      ) : null,
    )}
  </g>
));

const ActiveLayer = React.memo(
  (props: {
    map: WorldMapData;
    values: Map<string, number>;
    color: string;
    zoom: number;
    interactive: boolean;
    describe?: (code: string) => string;
  }) => {
    const max = Math.max(0, ...props.values.values());

    return (
      <g>
        {props.map.countries.map(([code, d, cx, cy]) => {
          const value = props.values.get(code) ?? 0;
          if (value <= 0) return null;
          const opacity = getIntensity(value, max);
          const common = {
            "data-code": code,
            fill: props.color,
            fillOpacity: opacity,
            className: twMerge(
              "stroke-white outline-none transition-[fill-opacity] duration-200",
              props.interactive && "cursor-pointer",
            ),
            tabIndex: props.interactive ? 0 : undefined,
            role: props.interactive ? "button" : undefined,
            "aria-label": props.describe?.(code),
          };

          return d ? (
            <path
              key={code}
              d={d}
              strokeWidth={0.6}
              vectorEffect="non-scaling-stroke"
              strokeLinejoin="round"
              {...common}
            />
          ) : (
            <circle
              key={code}
              cx={cx}
              cy={cy}
              r={2.6 / Math.sqrt(props.zoom)}
              strokeWidth={0.75}
              vectorEffect="non-scaling-stroke"
              {...common}
              fillOpacity={Math.max(opacity, 0.7)}
            />
          );
        })}
      </g>
    );
  },
);

const getRoutePath = (
  map: WorldMapData,
  route: WorldMapRoute,
): string | undefined => {
  const [x1, y1] = projectEqualEarth(map, route.from);
  const [x2, y2] = projectEqualEarth(map, route.to);
  const dx = x2 - x1;
  const dy = y2 - y1;
  const length = Math.hypot(dx, dy);
  if (length < 0.5) return undefined;

  const bend = Math.min(length * 0.22, 60);
  const cx = (x1 + x2) / 2 - (dy / length) * bend;
  const cy = (y1 + y2) / 2 + (dx / length) * bend;

  return `M${x1.toFixed(1)},${y1.toFixed(1)}Q${cx.toFixed(1)},${cy.toFixed(1)} ${x2.toFixed(1)},${y2.toFixed(1)}`;
};

const WorldMap = (props: {
  countries?: WorldMapCountry[];
  points?: WorldMapPoint[];
  routes?: WorldMapRoute[];
  tone?: WorldMapTone;
  selected?: string;
  fit?: "world" | "points";
  onSelect?: (code: string) => void;
  onSelectPoint?: (point: WorldMapPoint) => void;
  describeCountry?: (code: string) => string;
  renderTooltip?: (hover: {
    code?: string;
    point?: WorldMapPoint;
  }) => React.ReactNode;
  overlay?: React.ReactNode;
  label: string;
  className?: string;
}) => {
  useChartColorScheme();
  const { data: map, error } = useWorldMap();
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [hover, setHover] = React.useState<Hover>();
  const pointerType = React.useRef<string>("mouse");

  const values = React.useMemo(() => {
    const ret = new Map<string, number>();
    for (const itm of props.countries ?? []) {
      const code = itm.code?.toUpperCase();
      if (!code || !(itm.value > 0)) continue;
      ret.set(code, (ret.get(code) ?? 0) + itm.value);
    }
    return ret;
  }, [props.countries]);

  const points = props.points ?? NO_POINTS;
  const routes = props.routes ?? NO_ROUTES;

  const viewBox = React.useMemo<ViewBox>(
    () =>
      map
        ? props.fit === "points"
          ? getFitViewBox(map, points)
          : { x: 0, y: 0, w: map.width, h: map.height }
        : { x: 0, y: 0, w: WIDTH, h: HEIGHT },
    [map, props.fit, points],
  );
  const zoom = (map?.width ?? WIDTH) / viewBox.w;

  const color = getToneColor(props.tone);
  const interactive = !!props.onSelect;
  const byCode = React.useMemo(
    () => new Map((map?.countries ?? []).map((itm) => [itm[0], itm])),
    [map],
  );
  const pointByID = React.useMemo(
    () => new Map(points.map((itm) => [itm.id, itm])),
    [points],
  );
  const maxPoint = Math.max(1, ...points.map((itm) => itm.value ?? 1));

  const place = (x: number, y: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    const width = rect?.width ?? 0;
    const height = rect?.height ?? 0;
    return {
      left: Math.min(Math.max(x, 8), Math.max(width - 8, 8)),
      top: Math.min(Math.max(y, 8), Math.max(height - 8, 8)),
      flipX: x > width / 2,
      flipY: y > height / 2,
    };
  };

  const toLocal = (x: number, y: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) return place(0, 0);
    return place(
      ((x - viewBox.x) / viewBox.w) * rect.width,
      ((y - viewBox.y) / viewBox.h) * rect.height,
    );
  };

  const getTarget = (target: EventTarget | null) => {
    const el = (target as Element | null)?.closest?.("[data-code],[data-point]");
    if (!el) return undefined;
    const code = el.getAttribute("data-code") ?? undefined;
    const pointID = el.getAttribute("data-point") ?? undefined;
    const point = pointID ? pointByID.get(pointID) : undefined;
    if (!code && !point) return undefined;
    return { code, point };
  };

  const onPointerMove = (e: React.PointerEvent) => {
    pointerType.current = e.pointerType;
    const target = getTarget(e.target);
    const rect = containerRef.current?.getBoundingClientRect();
    if (!target || !rect) {
      if (e.pointerType === "mouse") setHover(undefined);
      return;
    }
    setHover({
      ...target,
      ...place(e.clientX - rect.left, e.clientY - rect.top),
    });
  };

  const onClick = (e: React.MouseEvent) => {
    const target = getTarget(e.target);
    if (!target) {
      setHover(undefined);
      return;
    }

    const rect = containerRef.current?.getBoundingClientRect();
    const isSame =
      hover &&
      ((target.code && hover.code === target.code) ||
        (target.point && hover.point?.id === target.point.id));
    if (pointerType.current === "touch" && !isSame && rect) {
      setHover({
        ...target,
        ...place(e.clientX - rect.left, e.clientY - rect.top),
      });
      return;
    }

    if (target.point && props.onSelectPoint) {
      props.onSelectPoint(target.point);
    } else if (target.code && props.onSelect && values.has(target.code)) {
      props.onSelect(target.code);
    }
  };

  const onFocus = (e: React.FocusEvent) => {
    const target = getTarget(e.target);
    if (!target || !map) return;
    let at = toLocal(0, 0);
    if (target.point) {
      const [x, y] = projectEqualEarth(map, target.point);
      at = toLocal(x, y);
    } else if (target.code) {
      const entry = byCode.get(target.code);
      if (entry) at = toLocal(entry[2], entry[3]);
    }
    setHover({ ...target, ...at });
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setHover(undefined);
      return;
    }
    if (e.key !== "Enter" && e.key !== " ") return;
    const target = getTarget(e.target);
    if (!target) return;
    e.preventDefault();
    if (target.point && props.onSelectPoint) props.onSelectPoint(target.point);
    else if (target.code && props.onSelect) props.onSelect(target.code);
  };

  const hovered = hover?.code ? byCode.get(hover.code) : undefined;
  const selected = props.selected ? byCode.get(props.selected) : undefined;
  const tooltip =
    hover && props.renderTooltip
      ? props.renderTooltip({ code: hover.code, point: hover.point })
      : undefined;

  return (
    <div
      ref={containerRef}
      className={twMerge(
        "relative aspect-[25/11] w-full select-none overflow-hidden",
        props.className,
      )}
      onPointerLeave={(e) => {
        if (e.pointerType === "mouse") setHover(undefined);
      }}
    >
      {!map ? (
        <div
          aria-hidden={!error}
          className={twMerge(
            "absolute inset-0 rounded-lg bg-slate-100",
            !error && "motion-safe:animate-pulse",
          )}
        >
          {error && (
            <p className="flex h-full items-center justify-center text-xs font-normal text-slate-500">
              The map could not be loaded.
            </p>
          )}
        </div>
      ) : (
        <svg
          viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.w} ${viewBox.h}`}
          preserveAspectRatio="xMidYMid meet"
          role="group"
          aria-label={props.label}
          className="absolute inset-0 h-full w-full"
          onPointerMove={onPointerMove}
          onClick={onClick}
          onFocus={onFocus}
          onBlur={() => setHover(undefined)}
          onKeyDown={onKeyDown}
        >
          <LandLayer map={map} />
          <ActiveLayer
            map={map}
            values={values}
            color={color}
            zoom={zoom}
            interactive={interactive}
            describe={props.describeCountry}
          />

          {selected?.[1] && (
            <path
              d={selected[1]}
              fill="none"
              stroke={CHART_INK.primary}
              strokeWidth={1.6}
              vectorEffect="non-scaling-stroke"
              strokeLinejoin="round"
              pointerEvents="none"
            />
          )}
          {hovered?.[1] && hovered[0] !== selected?.[0] && (
            <path
              d={hovered[1]}
              fill="none"
              stroke={CHART_INK.secondary}
              strokeWidth={1.2}
              vectorEffect="non-scaling-stroke"
              strokeLinejoin="round"
              pointerEvents="none"
            />
          )}

          {routes.length > 0 && (
            <g fill="none" pointerEvents="none">
              {routes.map((route) => {
                const d = getRoutePath(map, route);
                if (!d) return null;
                return (
                  <path
                    key={route.id}
                    d={d}
                    stroke={getToneColor(route.tone)}
                    strokeWidth={1.4}
                    strokeOpacity={0.75}
                    strokeDasharray="4 3"
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                );
              })}
            </g>
          )}

          {points.length > 0 && (
            <g>
              {[...points]
                .sort((a, b) => (b.value ?? 1) - (a.value ?? 1))
                .map((point) => {
                  const [x, y] = projectEqualEarth(map, point);
                  const r =
                    (2.6 + 6 * Math.sqrt((point.value ?? 1) / maxPoint)) /
                    Math.sqrt(zoom);
                  const pointColor = getToneColor(point.tone);
                  const isHovered = hover?.point?.id === point.id;

                  return (
                    <g key={point.id}>
                      {point.current && (
                        <circle
                          cx={x}
                          cy={y}
                          r={r * 1.6}
                          fill={pointColor}
                          fillOpacity={0.35}
                          pointerEvents="none"
                          className="origin-center [transform-box:fill-box] motion-safe:animate-ping"
                        />
                      )}
                      <circle
                        data-point={point.id}
                        cx={x}
                        cy={y}
                        r={isHovered ? r * 1.2 : r}
                        fill={pointColor}
                        fillOpacity={point.current ? 1 : 0.82}
                        strokeWidth={isHovered ? 1.75 : 1}
                        vectorEffect="non-scaling-stroke"
                        tabIndex={props.onSelectPoint ? 0 : undefined}
                        className={twMerge(
                          "stroke-white outline-none transition-[r] duration-150",
                          (props.onSelectPoint || props.renderTooltip) &&
                            "cursor-pointer",
                        )}
                      />
                    </g>
                  );
                })}
            </g>
          )}
        </svg>
      )}

      {props.overlay && map && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center p-4">
          {props.overlay}
        </div>
      )}

      {tooltip && hover && (
        <div
          role="tooltip"
          className="pointer-events-none absolute z-10 max-w-64 rounded-[10px] border border-slate-700 bg-slate-900 px-3 py-2 text-xs text-white shadow-raised"
          style={{
            left: hover.left,
            top: hover.top,
            transform: `translate(${hover.flipX ? "calc(-100% - 12px)" : "12px"}, ${hover.flipY ? "calc(-100% - 12px)" : "12px"})`,
          }}
        >
          {tooltip}
        </div>
      )}
    </div>
  );
};

export const MapLegend = (props: {
  tone?: WorldMapTone;
  label?: string;
  className?: string;
}) => {
  useChartColorScheme();
  const color = getToneColor(props.tone);

  return (
    <div
      className={twMerge(
        "flex items-center gap-2 text-micro font-normal text-slate-500",
        props.className,
      )}
    >
      {props.label && <span className="font-semibold">{props.label}</span>}
      <span>Fewer</span>
      <span
        aria-hidden="true"
        className="h-1.5 w-16 rounded-full"
        style={{
          background: `linear-gradient(90deg, color-mix(in srgb, ${color} 22%, transparent), ${color})`,
        }}
      />
      <span>More</span>
    </div>
  );
};

export default WorldMap;
