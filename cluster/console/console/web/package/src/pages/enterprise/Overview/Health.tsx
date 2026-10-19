import {
  GetClusterHealthResponse_Status,
  GetClusterHealthResponse_Subsystem_Type,
} from "@/apis/visibilityv1/visibilityv1";
import { EmptyHint, Panel } from "@/components/Dashboard/components";
import { StatusBadge, SubsystemGrid } from "@/components/Dashboard/health";
import { useClusterHealth } from "@/components/Dashboard/queries";
import { periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { HeartPulse } from "lucide-react";

const SUBSYSTEMS = [
  GetClusterHealthResponse_Subsystem_Type.CERTIFICATES,
  GetClusterHealthResponse_Subsystem_Type.DIRECTORY_PROVIDERS,
  GetClusterHealthResponse_Subsystem_Type.SECRET_STORES,
  GetClusterHealthResponse_Subsystem_Type.DEVICE_MANAGERS,
];

const STATUS_RANK: Record<number, number> = {
  [GetClusterHealthResponse_Status.CRITICAL]: 3,
  [GetClusterHealthResponse_Status.DEGRADED]: 2,
  [GetClusterHealthResponse_Status.UNKNOWN]: 1,
};

const Health = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const health = useClusterHealth(periodMinutes, QUERY_PRIORITY.high);
  const subsystems = (health.data?.subsystems ?? []).filter((item) =>
    SUBSYSTEMS.includes(item.type),
  );

  const worst = subsystems.reduce(
    (status, item) =>
      (STATUS_RANK[item.status] ?? 0) > (STATUS_RANK[status] ?? 0)
        ? item.status
        : status,
    GetClusterHealthResponse_Status.OK,
  );

  return (
    <Panel
      icon={HeartPulse}
      title="Integration health"
      description={`Certificates, directories, secret stores and device managers over the last ${rangeLabel}`}
      actions={
        subsystems.length > 0 ? <StatusBadge status={worst} /> : undefined
      }
    >
      {health.isLoading && subsystems.length === 0 ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2, 3].map((index) => (
            <div
              key={index}
              className="h-[86px] motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : subsystems.length === 0 ? (
        <EmptyHint>
          The Cluster health could not be evaluated right now.
        </EmptyHint>
      ) : (
        <SubsystemGrid subsystems={subsystems} />
      )}
    </Panel>
  );
};

export default Health;
