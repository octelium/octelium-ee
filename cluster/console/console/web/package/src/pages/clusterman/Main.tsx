import {
  ClusterConfig_Status_UpgradeRequest_State,
  UpgradeClusterRequest,
  UpgradeClusterRequest_Request_Core,
  UpgradeClusterRequest_Request_PackageCordium,
  UpgradeClusterRequest_Request_PackageEnterprise,
} from "@/apis/enterprisev1/enterprisev1";
import TimeAgo from "@/components/TimeAgo";
import { onError } from "@/utils";
import { getClientCluster, getClientEnterprise } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Modal, Switch, TextInput } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation, useQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  AlertTriangle,
  ArrowUpCircle,
  CheckCircle2,
  ChevronRight,
  Clock,
  Loader2,
  X,
  XCircle,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

const StatCard = ({
  label,
  value,
  variant = "neutral",
}: {
  label: string;
  value: React.ReactNode;
  variant?: "success" | "danger" | "neutral";
}) => (
  <div className="flex flex-col gap-0.5 px-4 py-3 rounded-lg border border-slate-200 bg-white">
    <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {label}
    </span>
    <span
      className={twMerge(
        "text-[0.85rem] font-bold",
        variant === "success" && "text-emerald-600",
        variant === "danger" && "text-red-600",
        variant === "neutral" && "text-slate-700",
      )}
    >
      {value}
    </span>
  </div>
);

const UpgradeStateBadge = ({
  state,
}: {
  state: ClusterConfig_Status_UpgradeRequest_State;
}) =>
  match(state)
    .with(ClusterConfig_Status_UpgradeRequest_State.SUCCESS, () => (
      <span className="inline-flex items-center gap-1 text-[0.72rem] font-bold px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200">
        <CheckCircle2 size={11} strokeWidth={2.5} />
        Succeeded
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.FAILED, () => (
      <span className="inline-flex items-center gap-1 text-[0.72rem] font-bold px-2 py-0.5 rounded-full bg-red-50 text-red-700 border border-red-200">
        <XCircle size={11} strokeWidth={2.5} />
        Failed
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.UPGRADING, () => (
      <span className="inline-flex items-center gap-1 text-[0.72rem] font-bold px-2 py-0.5 rounded-full bg-blue-50 text-blue-700 border border-blue-200">
        <Loader2 size={11} strokeWidth={2.5} className="animate-spin" />
        Upgrading
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.UPGRADE_REQUESTED, () => (
      <span className="inline-flex items-center gap-1 text-[0.72rem] font-bold px-2 py-0.5 rounded-full bg-amber-50 text-amber-700 border border-amber-200">
        <Clock size={11} strokeWidth={2.5} />
        Requested
      </span>
    ))
    .otherwise(() => null);

const PackageRow = ({
  label,
  description,
  enabled,
  version,
  onToggle,
  onVersionChange,
}: {
  label: string;
  description: string;
  enabled: boolean;
  version: string;
  onToggle: (checked: boolean) => void;
  onVersionChange: (v: string) => void;
}) => (
  <div
    className={twMerge(
      "rounded-lg border transition-[border-color,background] duration-150",
      enabled ? "border-slate-300 bg-slate-50/60" : "border-slate-200 bg-white",
    )}
  >
    <div className="flex items-start justify-between px-4 py-3 gap-4">
      <div className="flex flex-col gap-0.5 min-w-0">
        <span className="text-[0.78rem] font-bold text-slate-800">{label}</span>
        <span className="text-[0.7rem] font-semibold text-slate-400">
          {description}
        </span>
      </div>
      <Switch
        checked={enabled}
        onChange={(e) => onToggle(e.currentTarget.checked)}
        size="sm"
      />
    </div>

    <AnimatePresence initial={false}>
      {enabled && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.18 }}
          className="overflow-hidden"
        >
          <div className="px-4 pb-4 border-t border-slate-200 pt-3">
            <TextInput
              label="Version"
              placeholder="e.g. 1.2.3 — leave empty for latest"
              value={version}
              onChange={(e) => onVersionChange(e.target.value)}
              styles={{
                input: {
                  fontSize: "0.78rem",
                },
              }}
            />
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  </div>
);

const UpgradeCluster = () => {
  const [opened, { open, close }] = useDisclosure(false);
  const [confirmed, setConfirmed] = React.useState(false);

  const [req, setReq] = React.useState(
    UpgradeClusterRequest.create({ request: {} }),
  );

  const handleClose = () => {
    close();
    setConfirmed(false);
    setReq(UpgradeClusterRequest.create({ request: {} }));
  };

  const mutationUpgrade = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().upgradeCluster(req);
      return response;
    },
    onSuccess: () => {
      toast.success("Cluster upgrade started");
      invalidateKey(["clusterman", "main", "getCluster"]);
      handleClose();
    },
    onError,
  });

  const hasSelection =
    !!req.request?.core ||
    !!req.request?.packageEnterprise ||
    !!req.request?.packageCordium;

  const update = (fn: (r: UpgradeClusterRequest) => void) => {
    const next = UpgradeClusterRequest.clone(req);
    fn(next);
    setReq(next);
  };

  return (
    <>
      <button
        onClick={open}
        className="flex items-center gap-2 px-4 py-2 rounded-lg text-[0.82rem] font-bold bg-slate-900 text-white border border-slate-900 hover:bg-slate-800 transition-colors duration-150 shadow-[0_2px_8px_rgba(15,23,42,0.18)] cursor-pointer"
      >
        <ArrowUpCircle size={15} strokeWidth={2.5} />
        Upgrade cluster
      </button>

      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        size="lg"
        withCloseButton={false}
        padding={0}
        styles={{
          content: {
            borderRadius: "12px",
            border: "1px solid #e2e8f0",
            overflow: "hidden",
            display: "flex",
            flexDirection: "column",
            maxHeight: "90vh",
          },
          body: {
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
            flex: 1,
          },
        }}
      >
        <div className="flex flex-col h-full overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 bg-slate-50/60 shrink-0">
            <div className="flex items-center gap-2">
              <ArrowUpCircle
                size={15}
                className="text-slate-600 shrink-0"
                strokeWidth={2.5}
              />
              <span className="text-[0.85rem] font-bold text-slate-800">
                Upgrade cluster
              </span>
            </div>
            <button
              onClick={handleClose}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors duration-150 cursor-pointer"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto px-5 py-4 flex flex-col gap-3">
            <div className="flex items-start gap-2 px-3 py-2.5 rounded-lg bg-amber-50 border border-amber-200">
              <AlertTriangle
                size={13}
                className="text-amber-600 shrink-0 mt-0.5"
                strokeWidth={2.5}
              />
              <p className="text-[0.72rem] font-semibold text-amber-700">
                Select the components you want to upgrade and confirm below.
              </p>
            </div>

            <p className="text-[0.68rem] font-bold uppercase tracking-[0.07em] text-slate-400 mt-1">
              Components
            </p>

            <PackageRow
              label="Core"
              description="The core cluster runtime and control plane"
              enabled={!!req.request?.core}
              version={req.request?.core?.version ?? ""}
              onToggle={(checked) =>
                update((r) => {
                  r.request!.core = checked
                    ? UpgradeClusterRequest_Request_Core.create()
                    : undefined;
                })
              }
              onVersionChange={(v) =>
                update((r) => {
                  r.request!.core!.version = v;
                })
              }
            />

            <PackageRow
              label="Enterprise package"
              description="Enterprise features and integrations"
              enabled={!!req.request?.packageEnterprise}
              version={req.request?.packageEnterprise?.version ?? ""}
              onToggle={(checked) =>
                update((r) => {
                  r.request!.packageEnterprise = checked
                    ? UpgradeClusterRequest_Request_PackageEnterprise.create()
                    : undefined;
                })
              }
              onVersionChange={(v) =>
                update((r) => {
                  r.request!.packageEnterprise!.version = v;
                })
              }
            />

            <PackageRow
              label="Cordium package"
              description="Cordium networking and proxy layer"
              enabled={!!req.request?.packageCordium}
              version={req.request?.packageCordium?.version ?? ""}
              onToggle={(checked) =>
                update((r) => {
                  r.request!.packageCordium = checked
                    ? UpgradeClusterRequest_Request_PackageCordium.create()
                    : undefined;
                })
              }
              onVersionChange={(v) =>
                update((r) => {
                  r.request!.packageCordium!.version = v;
                })
              }
            />

            {hasSelection && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                transition={{ duration: 0.18 }}
                className="overflow-hidden"
              >
                <div className="flex items-start gap-2 px-3 py-2.5 rounded-lg bg-slate-50 border border-slate-200">
                  <ChevronRight
                    size={13}
                    className="text-slate-400 shrink-0 mt-0.5"
                    strokeWidth={2.5}
                  />
                  <div className="text-[0.72rem] font-semibold text-slate-600">
                    Selected:{" "}
                    {[
                      req.request?.core &&
                        `Core${req.request.core.version ? ` (${req.request.core.version})` : " (latest)"}`,
                      req.request?.packageEnterprise &&
                        `Enterprise${req.request.packageEnterprise.version ? ` (${req.request.packageEnterprise.version})` : " (latest)"}`,
                      req.request?.packageCordium &&
                        `Cordium${req.request.packageCordium.version ? ` (${req.request.packageCordium.version})` : " (latest)"}`,
                    ]
                      .filter(Boolean)
                      .join(", ")}
                  </div>
                </div>
              </motion.div>
            )}

            <Switch
              checked={confirmed}
              onChange={(e) => setConfirmed(e.currentTarget.checked)}
              color="red"
              size="sm"
              label={
                <span className="text-[0.78rem] font-semibold text-slate-600">
                  Yes, upgrade the Cluster.
                </span>
              }
            />
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-slate-200 bg-slate-50/60 shrink-0">
            <button
              onClick={handleClose}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.78rem] font-bold text-slate-600 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer"
            >
              Cancel
            </button>

            <button
              disabled={
                !confirmed || !hasSelection || mutationUpgrade.isPending
              }
              onClick={() => mutationUpgrade.mutate()}
              className={twMerge(
                "flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.78rem] font-bold",
                "border transition-colors duration-150",
                "disabled:opacity-40 disabled:cursor-not-allowed",
                confirmed && hasSelection
                  ? "bg-slate-900 text-white border-slate-900 hover:bg-slate-800 shadow-[0_2px_8px_rgba(15,23,42,0.2)]"
                  : "bg-slate-100 text-slate-400 border-slate-200",
              )}
            >
              {mutationUpgrade.isPending ? (
                <>
                  <Loader2
                    size={12}
                    className="animate-spin"
                    strokeWidth={2.5}
                  />{" "}
                  Upgrading…
                </>
              ) : (
                <>
                  <ArrowUpCircle size={12} strokeWidth={2.5} /> Upgrade cluster
                </>
              )}
            </button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default () => {
  const qry = useQuery({
    queryKey: ["clusterman", "main", "getCluster"],
    queryFn: async () => getClientEnterprise().getClusterConfig({} as any),
    refetchInterval: 15000,
  });

  if (!qry.isSuccess || !qry.data) return null;

  const cluster = qry.data.response;
  const status = cluster.status!;
  const current = status.upgradeRequest;
  const last = status.lastUpgradeRequests[0];
  const isIdle = !current && status.lastUpgradeRequests.length === 0;

  return (
    <div className="w-full flex flex-col gap-6">
      {(status.totalSuccessfulUpgrades > 0 ||
        status.totalFailedUpgrades > 0) && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {status.totalSuccessfulUpgrades > 0 && (
            <StatCard
              label="Successful upgrades"
              value={status.totalSuccessfulUpgrades}
              variant="success"
            />
          )}
          {status.totalFailedUpgrades > 0 && (
            <StatCard
              label="Failed upgrades"
              value={status.totalFailedUpgrades}
              variant="danger"
            />
          )}
          {last && (
            <StatCard
              label="Last upgrade"
              value={<TimeAgo rfc3339={last.doneAt} />}
            />
          )}
        </div>
      )}

      {current && (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50/60">
            <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
              Current upgrade
            </span>
            <UpgradeStateBadge state={current.state} />
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 px-4 py-3 text-[0.75rem]">
            <div className="flex flex-col gap-0.5">
              <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                Started
              </span>
              <span className="font-semibold text-slate-700">
                <TimeAgo rfc3339={current.createdAt} />
              </span>
            </div>
            {current.doneAt && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Completed
                </span>
                <span className="font-semibold text-slate-700">
                  <TimeAgo rfc3339={current.doneAt} />
                </span>
              </div>
            )}
            {current.request?.core?.version && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Core version
                </span>
                <span className="font-semibold text-slate-700 font-mono">
                  {current.request.core.version}
                </span>
              </div>
            )}
            {current.request?.packageEnterprise?.version && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Enterprise version
                </span>
                <span className="font-semibold text-slate-700 font-mono">
                  {current.request.packageEnterprise.version}
                </span>
              </div>
            )}
            {current.request?.packageCordium?.version && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Cordium version
                </span>
                <span className="font-semibold text-slate-700 font-mono">
                  {current.request.packageCordium.version}
                </span>
              </div>
            )}
          </div>
        </div>
      )}

      {last && !current && (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50/60">
            <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
              Last upgrade
            </span>
            <UpgradeStateBadge state={last.state} />
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 px-4 py-3 text-[0.75rem]">
            <div className="flex flex-col gap-0.5">
              <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                Completed
              </span>
              <span className="font-semibold text-slate-700">
                <TimeAgo rfc3339={last.doneAt} />
              </span>
            </div>
            {last.request?.core?.version && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Core version
                </span>
                <span className="font-semibold text-slate-700 font-mono">
                  {last.request.core.version}
                </span>
              </div>
            )}
          </div>
        </div>
      )}

      {isIdle && (
        <div className="flex items-center justify-center py-6">
          <span className="text-[0.75rem] font-semibold text-slate-400">
            No upgrades have been performed yet
          </span>
        </div>
      )}

      <div className="flex justify-end">
        <UpgradeCluster />
      </div>
    </div>
  );
};
