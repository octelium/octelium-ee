export type WorldMapData = {
  width: number;
  height: number;
  scale: number;
  translate: [number, number];
  countries: [string, string, number, number][];
};

export type LatLng = { lat: number; lng: number };

let worldMap: Promise<WorldMapData> | undefined;

export const loadWorldMap = () => {
  worldMap ??= import("./world.json")
    .then((mod) => mod.default as unknown as WorldMapData)
    .catch((err) => {
      worldMap = undefined;
      throw err;
    });
  return worldMap;
};

const A1 = 1.340264;
const A2 = -0.081106;
const A3 = 0.000893;
const A4 = 0.003796;
const M = Math.sqrt(3) / 2;

export const projectEqualEarth = (
  map: Pick<WorldMapData, "scale" | "translate">,
  point: LatLng,
): [number, number] => {
  const lambda = (Math.max(-180, Math.min(180, point.lng)) * Math.PI) / 180;
  const phi = (Math.max(-90, Math.min(90, point.lat)) * Math.PI) / 180;
  const l = Math.asin(M * Math.sin(phi));
  const l2 = l * l;
  const l6 = l2 * l2 * l2;
  const x =
    (lambda * Math.cos(l)) / (M * (A1 + 3 * A2 * l2 + l6 * (7 * A3 + 9 * A4 * l2)));
  const y = l * (A1 + A2 * l2 + l6 * (A3 + A4 * l2));

  return [
    map.translate[0] + x * map.scale,
    map.translate[1] - y * map.scale,
  ];
};

const EARTH_RADIUS_KM = 6371;

export const distanceKm = (a: LatLng, b: LatLng) => {
  const rad = Math.PI / 180;
  const dLat = (b.lat - a.lat) * rad;
  const dLng = (b.lng - a.lng) * rad;
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(a.lat * rad) * Math.cos(b.lat * rad) * Math.sin(dLng / 2) ** 2;
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.min(1, Math.sqrt(h)));
};

export const hasCoordinates = (
  coordinates?: { latitude: number; longitude: number },
) =>
  !!coordinates &&
  Number.isFinite(coordinates.latitude) &&
  Number.isFinite(coordinates.longitude) &&
  (coordinates.latitude !== 0 || coordinates.longitude !== 0);

export const normalizeCountryCode = (code?: string) => {
  const ret = code?.trim().toUpperCase();
  return ret && /^[A-Z]{2}$/.test(ret) ? ret : undefined;
};

let displayNames: Intl.DisplayNames | null | undefined;

export const getCountryName = (code?: string, fallback?: string) => {
  const normalized = normalizeCountryCode(code);
  if (!normalized) return fallback || code || "Unknown";

  if (displayNames === undefined) {
    try {
      displayNames = new Intl.DisplayNames(["en"], { type: "region" });
    } catch {
      displayNames = null;
    }
  }

  try {
    const name = displayNames?.of(normalized);
    if (name && name !== normalized) return name;
  } catch {
    return fallback || normalized;
  }

  return fallback || normalized;
};

export const formatLatLng = (point: LatLng) =>
  `${Math.abs(point.lat).toFixed(2)}°${point.lat >= 0 ? "N" : "S"}, ${Math.abs(point.lng).toFixed(2)}°${point.lng >= 0 ? "E" : "W"}`;

export const formatDistance = (km: number) =>
  km < 1
    ? "<1 km"
    : km < 100
      ? `${km.toFixed(0)} km`
      : `${Math.round(km).toLocaleString()} km`;
