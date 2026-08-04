import {
  GetClusterInfoRequest,
  GetClusterInfoResponse,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import { getClientCluster } from "@/utils/client";
import { Alert, Button, Skeleton } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  CheckCircle2,
  PackageCheck,
  RefreshCw,
  Sparkles,
} from "lucide-react";

export const useClusterVersionInfo = () =>
  useQuery({
    queryKey: ["clusterInfo"],
    queryFn: async () => {
      const { response } = await getClientCluster().getClusterInfo(
        GetClusterInfoRequest.create({}),
      );
      return response;
    },
    refetchInterval: 60_000,
  });

type PackageVersion = {
  currentVersion: string;
  latestVersion: string;
  canUpgrade: boolean;
  setAt?: Timestamp;
};

const PackageCard = (props: { label: string; info: PackageVersion }) => {
  const { info } = props;
  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_1px_3px_rgba(15,23,42,0.035)]">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <PackageCheck size={15} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <div className="truncate text-[0.78rem] font-bold text-slate-800">
              {props.label}
            </div>
            {info.setAt && (
              <div className="text-[0.65rem] font-semibold text-slate-400">
                Installed <TimeAgo rfc3339={info.setAt} />
              </div>
            )}
          </div>
        </div>
        {info.canUpgrade ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[0.64rem] font-bold text-blue-700">
            <Sparkles size={10} /> Update available
          </span>
        ) : (
          <span className="inline-flex shrink-0 items-center gap-1 text-[0.65rem] font-bold text-emerald-600">
            <CheckCircle2 size={11} strokeWidth={2.5} /> Current
          </span>
        )}
      </div>

      <div className="flex min-w-0 items-center gap-2 rounded-lg border border-slate-100 bg-slate-50/70 px-3 py-2">
        <div className="min-w-0 flex-1">
          <div className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
            Installed
          </div>
          <div className="truncate text-[0.76rem] font-bold text-slate-700">
            {info.currentVersion || "Unknown"}
          </div>
        </div>
        {info.canUpgrade && (
          <>
            <ArrowRight size={13} className="shrink-0 text-slate-300" />
            <div className="min-w-0 flex-1 text-right">
              <div className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-blue-400">
                Available
              </div>
              <div className="truncate text-[0.76rem] font-bold text-blue-700">
                {info.latestVersion || "Latest"}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

const packages = (data: GetClusterInfoResponse) =>
  [
    data.core && { label: "Core", info: data.core },
    data.packageEnterprise && {
      label: "Enterprise",
      info: data.packageEnterprise,
    },
    data.packageCordium && { label: "Cordium", info: data.packageCordium },
  ].filter(Boolean) as { label: string; info: PackageVersion }[];

const ClusterVersionInfo = () => {
  const qry = useClusterVersionInfo();
  const items = qry.data ? packages(qry.data) : [];
  const updates = items.filter((item) => item.info.canUpgrade).length;

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 shadow-[0_1px_4px_rgba(15,23,42,0.04)]">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white px-4 py-3.5 sm:px-5">
        <div>
          <div className="text-[0.76rem] font-bold text-slate-800">
            Package readiness
          </div>
          <div className="mt-0.5 text-[0.67rem] font-semibold text-slate-400">
            Installed and available versions reported by the cluster
          </div>
        </div>
        <div className="flex items-center gap-2">
          {qry.data && (
            <span className="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-[0.64rem] font-bold text-slate-500">
              {updates > 0
                ? `${updates} ${updates === 1 ? "update" : "updates"}`
                : "All current"}
            </span>
          )}
          <Button
            variant="subtle"
            color="gray"
            size="compact-xs"
            leftSection={
              <RefreshCw
                size={11}
                className={qry.isFetching ? "animate-spin" : ""}
              />
            }
            loading={false}
            disabled={qry.isFetching}
            onClick={() => qry.refetch()}
          >
            Refresh
          </Button>
        </div>
      </div>

      <div className="p-3 sm:p-4">
        {qry.isLoading ? (
          <div className="grid gap-3 md:grid-cols-3">
            {[0, 1, 2].map((item) => (
              <Skeleton key={item} height={122} radius="lg" />
            ))}
          </div>
        ) : qry.isError ? (
          <Alert color="red" title="Could not load package versions">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span className="text-xs">{qry.error.message}</span>
              <Button size="compact-xs" variant="outline" onClick={() => qry.refetch()}>
                Try again
              </Button>
            </div>
          </Alert>
        ) : (
          <div className="grid gap-3 md:grid-cols-3">
            {items.map((item) => (
              <PackageCard key={item.label} {...item} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
};

export default ClusterVersionInfo;
