import { useComputedColorScheme } from "@mantine/core";

const LIGHT = {
  series: [
    "#2563eb",
    "#16a34a",
    "#0891b2",
    "#d97706",
    "#db2777",
    "#7c3aed",
    "#0d9488",
    "#4f46e5",
  ],
  status: {
    good: "#059669",
    warning: "#d97706",
    serious: "#ea580c",
    critical: "#dc2626",
  },
  ink: {
    primary: "#0f172a",
    secondary: "#475569",
    muted: "#64748b",
    grid: "#e2e8f0",
    axis: "#cbd5e1",
    onDark: "#f8fafc",
    onDarkMuted: "#cbd5e1",
    surface: "#ffffff",
    track: "#f1f5f9",
    marker: "#60a5fa",
    onAccent: "#ffffff",
    shadow: "rgba(15, 23, 42, 0.14)",
  },
  slices: ["#1d4ed8", "#60a5fa", "#bfdbfe", "#dbeafe"],
  tooltip: {
    backgroundColor: "#0f172a",
    borderColor: "#334155",
    shadow: "0 10px 24px rgba(15,23,42,.18)",
  },
} as const;

const DARK = {
  series: [
    "#7aa2ff",
    "#4ade80",
    "#22d3ee",
    "#fbbf24",
    "#f472b6",
    "#a78bfa",
    "#2dd4bf",
    "#98a4ff",
  ],
  status: {
    good: "#34d399",
    warning: "#fbbf24",
    serious: "#fb923c",
    critical: "#f87171",
  },
  ink: {
    primary: "#eaeff5",
    secondary: "#9facbe",
    muted: "#8593a8",
    grid: "#29313c",
    axis: "#3a4350",
    onDark: "#141920",
    onDarkMuted: "#3a4350",
    surface: "#141920",
    track: "#212730",
    marker: "#7aa2ff",
    onAccent: "#0a0f15",
    shadow: "rgba(0, 0, 0, 0.55)",
  },
  slices: ["#bfd4ff", "#7aa2ff", "#4a6fd0", "#2b3f70"],
  tooltip: {
    backgroundColor: "#eaeff5",
    borderColor: "#d4dbe6",
    shadow: "0 10px 24px rgba(0,0,0,.55)",
  },
} as const;

const current = () =>
  typeof document !== "undefined" &&
  document.documentElement.getAttribute("data-mantine-color-scheme") === "dark"
    ? DARK
    : LIGHT;

export const useChartColorScheme = () => useComputedColorScheme("light");

export const ALL_PAIRS_SAFE_SERIES = 5;

export const seriesColors = () => [...current().series];

export const sliceColors = () => [...current().slices];

export const seriesColor = (index: number) => {
  const series = current().series;
  return series[index % series.length];
};

export const STATUS_COLORS = {
  get good() {
    return current().status.good;
  },
  get warning() {
    return current().status.warning;
  },
  get serious() {
    return current().status.serious;
  },
  get critical() {
    return current().status.critical;
  },
};

export const CHART_INK = {
  get primary() {
    return current().ink.primary;
  },
  get secondary() {
    return current().ink.secondary;
  },
  get muted() {
    return current().ink.muted;
  },
  get grid() {
    return current().ink.grid;
  },
  get axis() {
    return current().ink.axis;
  },
  get onDark() {
    return current().ink.onDark;
  },
  get onDarkMuted() {
    return current().ink.onDarkMuted;
  },
  get surface() {
    return current().ink.surface;
  },
  get track() {
    return current().ink.track;
  },
  get marker() {
    return current().ink.marker;
  },
  get onAccent() {
    return current().ink.onAccent;
  },
  get shadow() {
    return current().ink.shadow;
  },
};

export const withAlpha = (hex: string, alpha: number) => {
  const value = hex.replace("#", "");
  const r = parseInt(value.slice(0, 2), 16);
  const g = parseInt(value.slice(2, 4), 16);
  const b = parseInt(value.slice(4, 6), 16);
  return `rgba(${r},${g},${b},${alpha})`;
};

export const CHART_FONT = "Ubuntu, sans-serif";

export const CHART_TOOLTIP = {
  get backgroundColor() {
    return current().tooltip.backgroundColor;
  },
  get borderColor() {
    return current().tooltip.borderColor;
  },
  borderWidth: 1,
  padding: [10, 12],
  get textStyle() {
    return {
      color: current().ink.onDark,
      fontSize: 12,
      fontFamily: CHART_FONT,
    };
  },
  get extraCssText() {
    return `border-radius:10px;box-shadow:${current().tooltip.shadow};`;
  },
};

export const axisLabelStyle = {
  get color() {
    return current().ink.muted;
  },
  fontSize: 10,
  fontFamily: CHART_FONT,
  fontWeight: 600,
  margin: 12,
};

export const splitLineStyle = {
  get lineStyle() {
    return { color: current().ink.grid, type: "dashed", opacity: 0.8 };
  },
};
