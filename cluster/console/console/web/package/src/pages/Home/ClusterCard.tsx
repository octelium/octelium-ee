import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import { useClusterVersionInfo } from "@/pages/clusterman/queries";
import { getDomain } from "@/utils";
import { CheckCircle2, PackageCheck, ServerCog, Sparkles } from "lucide-react";
import { Panel } from "@/components/Dashboard/components";

type PackageVersion = {
  currentVersion: string;
  latestVersion: string;
  canUpgrade: boolean;
  setAt?: Timestamp;
};

const Row = (props: { label: string; info?: PackageVersion }) => {
  const { info } = props;
  if (!info) return null;

  return (
    <div className="flex min-w-0 flex-col gap-1.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
      <div className="flex items-center justify-between gap-2">
        <span className="inline-flex min-w-0 items-center gap-2">
          <PackageCheck
            size={13}
            strokeWidth={2.2}
            className="shrink-0 text-slate-500"
          />
          <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-600">
            {props.label}
          </span>
        </span>

        {info.canUpgrade ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-1.5 py-px text-micro font-semibold text-blue-700">
            <Sparkles size={9} strokeWidth={2.5} />
            {info.latestVersion || "Update"}
          </span>
        ) : (
          <span className="inline-flex shrink-0 items-center gap-1 text-micro font-semibold text-emerald-600">
            <CheckCircle2 size={10} strokeWidth={2.5} />
            Current
          </span>
        )}
      </div>

      <span className="truncate text-body font-semibold text-slate-800">
        {info.currentVersion || "Unknown"}
      </span>

      {info.setAt && (
        <span className="text-micro font-normal text-slate-500">
          Installed <TimeAgo rfc3339={info.setAt} />
        </span>
      )}
    </div>
  );
};

const ClusterCard = () => {
  const query = useClusterVersionInfo();
  const data = query.data;
  const upgradable = [
    data?.core?.canUpgrade,
    data?.packageEnterprise?.canUpgrade,
    data?.packageCordium?.canUpgrade,
  ].filter(Boolean).length;

  return (
    <Panel
      icon={ServerCog}
      title="Cluster"
      description={
        upgradable > 0
          ? `${upgradable} package${upgradable === 1 ? "" : "s"} can be upgraded`
          : getDomain()
      }
      to="/clusterman"
      toLabel="Manage"
    >
      {query.isLoading ? (
        <div className="flex flex-col gap-2">
          {[0, 1].map((index) => (
            <div
              key={index}
              className="h-[78px] animate-pulse rounded-lg border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          <Row label="Core" info={data?.core} />
          <Row label="Enterprise" info={data?.packageEnterprise} />
          <Row label="Cordium" info={data?.packageCordium} />
        </div>
      )}
    </Panel>
  );
};

export default ClusterCard;
