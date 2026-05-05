import { GetClusterInfoRequest } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import { getClientCluster } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, CheckCircle2, RefreshCw } from "lucide-react";

const VersionBadge = ({ version }: { version: string }) => (
  <span className="inline-flex items-center px-2 py-0.5 rounded text-[0.72rem] font-mono font-bold bg-slate-100 border border-slate-200 text-slate-700">
    {version || "—"}
  </span>
);

const PackageRow = ({
  label,
  currentVersion,
  latestVersion,
  canUpgrade,
  setAt,
}: {
  label: string;
  currentVersion: string;
  latestVersion: string;
  canUpgrade: boolean;
  setAt?: Timestamp;
}) => (
  <div className="flex items-center justify-between gap-4 py-3 border-b border-slate-100 last:border-0">
    <div className="flex flex-col gap-0.5 min-w-0">
      <span className="text-[0.78rem] font-bold text-slate-700">{label}</span>
      {setAt && (
        <span className="text-[0.68rem] font-semibold text-slate-400">
          Updated <TimeAgo rfc3339={setAt} />
        </span>
      )}
    </div>

    <div className="flex items-center gap-2 shrink-0">
      <VersionBadge version={currentVersion} />

      {canUpgrade ? (
        <>
          <ArrowRight size={12} className="text-slate-400" strokeWidth={2.5} />
          <VersionBadge version={latestVersion} />
          <span className="inline-flex items-center px-2 py-0.5 rounded text-[0.68rem] font-bold bg-emerald-50 border border-emerald-200 text-emerald-700">
            Update available
          </span>
        </>
      ) : (
        <span className="inline-flex items-center gap-1 text-[0.68rem] font-bold text-slate-400">
          <CheckCircle2
            size={11}
            strokeWidth={2.5}
            className="text-emerald-500"
          />
          Up to date
        </span>
      )}
    </div>
  </div>
);

const ClusterVersionInfo = () => {
  const qry = useQuery({
    queryKey: ["clusterInfo"],
    queryFn: async () => {
      const { response } = await getClientCluster().getClusterInfo(
        GetClusterInfoRequest.create({}),
      );
      return response;
    },
    refetchInterval: 60_000,
  });

  return (
    <div className="w-full bg-white border border-slate-200 rounded-xl shadow-[0_1px_4px_rgba(15,23,42,0.06)]">
      <div className="flex items-center justify-between px-5 py-3.5 border-b border-slate-100 bg-slate-50/60">
        <span className="text-[0.72rem] font-bold uppercase tracking-[0.06em] text-slate-500">
          Installed Versions
        </span>
        <button
          onClick={() => qry.refetch()}
          disabled={qry.isFetching}
          className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer disabled:opacity-50"
        >
          <RefreshCw
            size={11}
            strokeWidth={2.5}
            className={qry.isFetching ? "animate-spin" : ""}
          />
        </button>
      </div>

      <div className="px-5">
        {qry.isLoading ? (
          <div className="flex flex-col gap-3 py-4">
            {[0, 1, 2].map((i) => (
              <div
                key={i}
                className="h-10 rounded-lg bg-slate-50 border border-slate-100 animate-pulse"
              />
            ))}
          </div>
        ) : qry.data ? (
          <div>
            {qry.data.core && (
              <PackageRow
                label="Core"
                currentVersion={qry.data.core.currentVersion}
                latestVersion={qry.data.core.latestVersion}
                canUpgrade={qry.data.core.canUpgrade}
                setAt={qry.data.core.setAt}
              />
            )}
            {qry.data.packageEnterprise && (
              <PackageRow
                label="Enterprise"
                currentVersion={qry.data.packageEnterprise.currentVersion}
                latestVersion={qry.data.packageEnterprise.latestVersion}
                canUpgrade={qry.data.packageEnterprise.canUpgrade}
                setAt={qry.data.packageEnterprise.setAt}
              />
            )}
            {qry.data.packageCordium && (
              <PackageRow
                label="Cordium"
                currentVersion={qry.data.packageCordium.currentVersion}
                latestVersion={qry.data.packageCordium.latestVersion}
                canUpgrade={qry.data.packageCordium.canUpgrade}
                setAt={qry.data.packageCordium.setAt}
              />
            )}
          </div>
        ) : (
          <div className="flex items-center justify-center py-8">
            <span className="text-[0.75rem] font-semibold text-slate-400">
              Failed to load version info
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default ClusterVersionInfo;
