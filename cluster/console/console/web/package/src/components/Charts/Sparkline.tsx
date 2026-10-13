import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { useId, useMemo } from "react";

const Sparkline = (props: {
  points?: { ts: Timestamp; value: number }[];
  color: string;
  height?: number;
}) => {
  const gradient = useId();
  const points = useMemo(() => {
    const data = (props.points ?? [])
      .map((point) => [Timestamp.toDate(point.ts).getTime(), point.value])
      .filter(
        ([time, value]) => Number.isFinite(time) && Number.isFinite(value),
      )
      .sort(([timeA], [timeB]) => timeA - timeB);
    if (data.length === 0) return [];

    const start = data[0][0];
    const duration = data[data.length - 1][0] - start;
    const [minimum, maximum] = data.reduce(
      ([minimum, maximum], [, value]) => [
        Math.min(minimum, value),
        Math.max(maximum, value),
      ],
      [0, 1],
    );
    const scale = Math.max(Math.abs(minimum), maximum);
    const range = maximum / scale - minimum / scale;

    return data.map(([time, value]) => [
      duration === 0 ? 50 : 1 + ((time - start) / duration) * 98,
      38 - ((value / scale - minimum / scale) / range) * 34,
    ]);
  }, [props.points]);

  if (points.length === 0) {
    return (
      <div
        className="flex w-full items-center"
        style={{ height: props.height ?? 40 }}
      >
        <span className="h-px w-full bg-slate-200" />
      </div>
    );
  }

  const line = points.map(([x, y]) => `${x},${y}`).join(" ");

  return (
    <svg
      aria-hidden="true"
      className="block w-full"
      viewBox="0 0 100 40"
      preserveAspectRatio="none"
      style={{ height: props.height ?? 40 }}
    >
      <defs>
        <linearGradient id={gradient} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={props.color} stopOpacity={0.2} />
          <stop offset="100%" stopColor={props.color} stopOpacity={0} />
        </linearGradient>
      </defs>
      <polygon
        points={`${points[0][0]},38 ${line} ${points[points.length - 1][0]},38`}
        fill={`url(#${gradient})`}
      />
      <polyline
        points={line}
        fill="none"
        stroke={props.color}
        strokeWidth={1.75}
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
};

export default Sparkline;
