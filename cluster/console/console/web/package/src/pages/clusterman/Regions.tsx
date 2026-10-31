import { Region } from "@/apis/corev1/corev1";
import {
  ClusterConfig_Status,
  ClusterConfig_Status_JoinRequest,
  ClusterConfig_Status_JoinRequest_State,
} from "@/apis/enterprisev1/enterprisev1";
import { PILL, Panel } from "@/components/Dashboard/components";
import RpcError from "@/components/RpcError";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { getResourcePath } from "@/utils/pb";
import { Alert, Button, Skeleton } from "@mantine/core";
import { Globe2, Loader2, PlusCircle, Star, TriangleAlert } from "lucide-react";
import { Link } from "react-router-dom";
import { JoinStateBadge } from "./components";
import {
  isDefaultRegion,
  isJoinActive,
  isJoinIncomplete,
  regionVersion,
  sortRegions,
  useRegions,
} from "./queries";

const VERSIONS = [
  { key: "octelium", label: "Core" },
  { key: "octeliumee", label: "Enterprise" },
  { key: "cordium", label: "Cordium" },
];

const RegionCard = (props: { region: Region }) => {
  const { region } = props;
  const isDefault = isDefaultRegion(region);
  const incomplete = isJoinIncomplete(region);
  const versions = VERSIONS.map((item) => ({
    ...item,
    version: regionVersion(region, item.key),
  })).filter((item) => !!item.version);

  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3.5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <Globe2 size={14} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <Link
              to={getResourcePath(region)}
              className="block truncate text-body font-semibold text-slate-800 hover:underline"
            >
              {region.metadata?.displayName || region.metadata?.name}
            </Link>
            <div className="truncate text-micro font-normal text-slate-500">
              {region.metadata?.name} · index {region.status?.index ?? 0}
            </div>
          </div>
        </div>

        {isDefault ? (
          <span
            className={twMerge(
              PILL,
              "border-slate-200 bg-slate-50 text-slate-600",
            )}
          >
            <Star size={9} strokeWidth={2.5} />
            Default
          </span>
        ) : incomplete ? (
          <span
            className={twMerge(
              PILL,
              "border-amber-200 bg-amber-50 text-amber-700",
            )}
          >
            <TriangleAlert size={9} strokeWidth={2.5} />
            Incomplete
          </span>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-1">
        {versions.length === 0 ? (
          <span className="text-micro font-normal text-slate-500">
            No package is installed yet
          </span>
        ) : (
          versions.map((item) => (
            <span
              key={item.key}
              className="inline-flex items-center gap-1 rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-micro font-normal text-slate-600"
            >
              {item.label}
              <span className="font-semibold text-slate-800">
                {item.version}
              </span>
            </span>
          ))
        )}
      </div>

      <span className="truncate text-micro font-normal text-slate-500">
        {incomplete ? (
          "The join did not complete. Join the Region again to retry."
        ) : region.metadata?.createdAt ? (
          <>
            Added <TimeAgo rfc3339={region.metadata.createdAt} />
          </>
        ) : (
          "Creation date unknown"
        )}
      </span>
    </div>
  );
};

const ActiveJoin = (props: { request: ClusterConfig_Status_JoinRequest }) => (
  <div className="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50/60 px-3.5 py-3">
    <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-white text-blue-700">
      <Loader2 size={14} className="motion-safe:animate-spin" />
    </span>
    <div className="min-w-0 flex-1">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-body font-semibold text-slate-800">
          Joining the Region {props.request.request?.name}
        </span>
        <JoinStateBadge state={props.request.state} />
      </div>
      <div className="mt-1 text-micro font-normal text-slate-600">
        Started <TimeAgo rfc3339={props.request.createdAt} /> · the Cluster
        components are being installed in the new Kubernetes cluster.
      </div>
    </div>
  </div>
);

const FailedJoin = (props: { request: ClusterConfig_Status_JoinRequest }) => (
  <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50/60 px-3.5 py-3">
    <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-red-200 bg-white text-red-600">
      <TriangleAlert size={14} strokeWidth={2.3} />
    </span>
    <div className="min-w-0 flex-1">
      <div className="text-body font-semibold text-slate-800">
        The last join of the Region {props.request.request?.name} did not
        complete
      </div>
      <div className="mt-1 text-micro font-normal text-slate-600">
        It failed <TimeAgo rfc3339={props.request.doneAt} />. Fix the cause
        below and join the Region again to retry.
      </div>
      {props.request.message && (
        <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-red-200 bg-white px-2.5 py-2 font-mono text-micro text-red-700">
          {props.request.message}
        </pre>
      )}
    </div>
  </div>
);

const Regions = (props: {
  status?: ClusterConfig_Status;
  disabled: boolean;
  onJoin: () => void;
}) => {
  const regions = useRegions();

  const current = props.status?.joinRequest;
  const active = isJoinActive(current?.state) ? current : undefined;
  const last = props.status?.lastJoinRequests[0];
  const failed =
    !active && last?.state === ClusterConfig_Status_JoinRequest_State.FAILED
      ? last
      : undefined;

  const items = sortRegions(regions.data?.items ?? []);

  return (
    <Panel
      icon={Globe2}
      title="Regions"
      description="The Kubernetes clusters that make up the Cluster"
      actions={
        <>
          {regions.data && (
            <span className="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold text-slate-600">
              {items.length} Region{items.length === 1 ? "" : "s"}
            </span>
          )}
          <Button
            variant="default"
            size="compact-xs"
            disabled={props.disabled}
            leftSection={<PlusCircle size={11} strokeWidth={2.5} />}
            onClick={props.onJoin}
          >
            {active ? "Join in progress" : "Join Region"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {active && <ActiveJoin request={active} />}
        {failed && <FailedJoin request={failed} />}

        {regions.isLoading ? (
          <div className="grid gap-3 md:grid-cols-3">
            {[0, 1, 2].map((index) => (
              <Skeleton key={index} height={128} radius="lg" />
            ))}
          </div>
        ) : regions.isError ? (
          <Alert color="red" title="Could not load the Regions">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <RpcError error={regions.error} />
              <Button
                size="compact-xs"
                variant="outline"
                onClick={() => regions.refetch()}
              >
                Try again
              </Button>
            </div>
          </Alert>
        ) : (
          <div className="grid gap-3 md:grid-cols-3">
            {items.map((item) => (
              <RegionCard key={item.metadata?.uid} region={item} />
            ))}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Regions;
