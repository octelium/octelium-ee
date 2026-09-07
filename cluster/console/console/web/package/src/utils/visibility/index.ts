import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import dayjs from "dayjs";
import { match } from "ts-pattern";

export interface PeriodOption {
  label: string;
  minutes: number;
}

export const PRIMARY_PERIODS: PeriodOption[] = [
  { label: "30m", minutes: 30 },
  { label: "1h", minutes: 60 },
  { label: "3h", minutes: 180 },
  { label: "6h", minutes: 360 },
  { label: "12h", minutes: 720 },
  { label: "24h", minutes: 1440 },
];

export const EXTENDED_PERIODS: PeriodOption[] = [
  { label: "5m", minutes: 5 },
  { label: "10m", minutes: 10 },
  { label: "15m", minutes: 15 },
  { label: "2d", minutes: 2880 },
  { label: "3d", minutes: 4320 },
  { label: "7d", minutes: 10080 },
  { label: "14d", minutes: 20160 },
];

export const ALL_PERIODS = [...PRIMARY_PERIODS, ...EXTENDED_PERIODS];

export const DEFAULT_PERIOD_MINUTES = 60;

export const isKnownPeriod = (minutes: number) =>
  ALL_PERIODS.some((option) => option.minutes === minutes);

export const periodLabel = (minutes: number) =>
  ALL_PERIODS.find((option) => option.minutes === minutes)?.label ?? "";

export const createDuration = (val: number, unit: string): Duration => {
  const typePayload = match(unit)
    .with("millisecond", () => ({
      oneofKind: "milliseconds" as const,
      milliseconds: val,
    }))
    .with("second", () => ({ oneofKind: "seconds" as const, seconds: val }))
    .with("minute", () => ({ oneofKind: "minutes" as const, minutes: val }))
    .with("hour", () => ({ oneofKind: "hours" as const, hours: val }))
    .with("day", () => ({ oneofKind: "days" as const, days: val }))
    .with("week", () => ({ oneofKind: "weeks" as const, weeks: val }))
    .with("month", () => ({ oneofKind: "months" as const, months: val }))
    .otherwise(() => ({ oneofKind: "seconds" as const, seconds: val }));
  return Duration.create({ type: typePayload as any });
};

export const getAutoInterval = (periodMinutes: number): Duration => {
  if (periodMinutes <= 15) return createDuration(30, "second");
  if (periodMinutes <= 60) return createDuration(1, "minute");
  if (periodMinutes <= 180) return createDuration(5, "minute");
  if (periodMinutes <= 360) return createDuration(10, "minute");
  if (periodMinutes <= 720) return createDuration(15, "minute");
  if (periodMinutes <= 1440) return createDuration(30, "minute");
  if (periodMinutes <= 4320) return createDuration(1, "hour");
  if (periodMinutes <= 10080) return createDuration(3, "hour");
  return createDuration(6, "hour");
};

export const buildTimestamps = (periodMinutes: number) => {
  const now = dayjs();
  const curFrom = now.subtract(periodMinutes, "minute").valueOf();
  const curTo = now.valueOf();
  const prevFrom = now.subtract(periodMinutes * 2, "minute").valueOf();
  const prevTo = curFrom;
  return { curFrom, curTo, prevFrom, prevTo };
};

export const toTs = (ms: number) => Timestamp.fromDate(new Date(ms));

export const refKey = (ref?: ObjectReference) => ref?.uid ?? ref?.name ?? null;

export const n = (v: unknown) => Number(v ?? 0);

export const pct = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100);

export const deltaPct = (cur: number, prev: number) =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

export const NO_REFS = {
  userRef: null,
  sessionRef: null,
  serviceRef: null,
  namespaceRef: null,
  regionRef: null,
  deviceRef: null,
  policyRef: null,
};

export const visibilityKeys = {
  accessSummary: (
    window: "current" | "previous",
    periodMinutes: number,
    status: string,
    refKeys: object,
  ) => ["accessLogSummary", window, periodMinutes, status, refKeys] as const,
  accessDataPoint: (periodMinutes: number, status: string, refKeys: object) =>
    ["accessLogDataPoint", periodMinutes, status, refKeys] as const,
  authSummary: (
    window: "current" | "previous",
    periodMinutes: number,
    refKeys: object,
  ) => ["authLogSummary", window, periodMinutes, refKeys] as const,
  authDataPoint: (periodMinutes: number, refKeys: object) =>
    ["authLogDataPoint", periodMinutes, refKeys] as const,
  auditSummary: (
    window: "current" | "previous",
    periodMinutes: number,
    refKeys: object,
  ) => ["auditLogSummary", window, periodMinutes, refKeys] as const,
  auditDataPoint: (periodMinutes: number, refKeys: object) =>
    ["auditLogDataPoint", periodMinutes, refKeys] as const,
  componentSummary: (window: "current" | "previous", periodMinutes: number) =>
    ["componentLogSummary", window, periodMinutes] as const,
  componentDataPoint: (periodMinutes: number) =>
    ["componentLogDataPoint", periodMinutes] as const,
  componentErrorDataPoint: (periodMinutes: number) =>
    ["componentLogDataPointErrors", periodMinutes] as const,
};
