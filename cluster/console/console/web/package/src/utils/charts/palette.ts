export const CHART_SURFACE = "#ffffff";

export const SERIES_COLORS = [
  "#2563eb",
  "#16a34a",
  "#0891b2",
  "#d97706",
  "#db2777",
  "#7c3aed",
  "#0d9488",
  "#4f46e5",
] as const;

export const ALL_PAIRS_SAFE_SERIES = 5;

export const seriesColor = (index: number) =>
  SERIES_COLORS[index % SERIES_COLORS.length];

export const STATUS_COLORS = {
  good: "#059669",
  warning: "#d97706",
  serious: "#ea580c",
  critical: "#dc2626",
} as const;

export const CHART_INK = {
  primary: "#0f172a",
  secondary: "#475569",
  muted: "#64748b",
  grid: "#e2e8f0",
  axis: "#cbd5e1",
  onDark: "#f8fafc",
  onDarkMuted: "#cbd5e1",
} as const;

export const CHART_FONT = "Ubuntu, sans-serif";

export const CHART_TOOLTIP = {
  backgroundColor: "#0f172a",
  borderColor: "#334155",
  borderWidth: 1,
  padding: [10, 12],
  textStyle: {
    color: CHART_INK.onDark,
    fontSize: 12,
    fontFamily: CHART_FONT,
  },
  extraCssText:
    "border-radius:10px;box-shadow:0 10px 24px rgba(15,23,42,.18);",
} as const;

export const axisLabelStyle = {
  color: CHART_INK.muted,
  fontSize: 10,
  fontFamily: CHART_FONT,
  fontWeight: 600,
  margin: 12,
} as const;

export const splitLineStyle = {
  lineStyle: { color: CHART_INK.grid, type: "dashed", opacity: 0.8 },
} as const;
