export const formatCount = (
  value: number,
  options: { compact?: boolean } = {},
) => {
  if (!Number.isFinite(value)) return "—";
  if (options.compact === false) return value.toLocaleString();
  return new Intl.NumberFormat(undefined, {
    notation: value >= 10_000 ? "compact" : "standard",
    maximumFractionDigits: 1,
  }).format(value);
};
