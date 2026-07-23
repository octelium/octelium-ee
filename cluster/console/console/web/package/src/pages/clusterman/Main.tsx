import {
  ClusterConfig_Status_UpgradeRequest,
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
import {
  ActionIcon,
  Button,
  Checkbox,
  Modal,
  Switch,
  TextInput,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation, useQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  AlertTriangle,
  ArrowUpCircle,
  CheckCircle2,
  ChevronDown,
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
import ClusterVersionInfo from "./ClusterVersionInfo";

const HISTORY_PREVIEW_COUNT = 3;

type UpgradeRequestSpec = ClusterConfig_Status_UpgradeRequest["request"];

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

const VersionChip = ({
  label,
  version,
}: {
  label: string;
  version: string;
}) => (
  <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border border-slate-200 bg-slate-50 text-[0.65rem] font-bold text-slate-600">
    {label}
    <span
      className={twMerge(
        "font-mono",
        version ? "text-slate-800" : "text-slate-400",
      )}
    >
      {version || "latest"}
    </span>
  </span>
);

const VersionChips = ({ request }: { request?: UpgradeRequestSpec }) => {
  if (!request) return null;
  return (
    <div className="flex flex-wrap items-center gap-1">
      {request.core && (
        <VersionChip label="Core" version={request.core.version} />
      )}
      {request.packageEnterprise && (
        <VersionChip
          label="Enterprise"
          version={request.packageEnterprise.version}
        />
      )}
      {request.packageCordium && (
        <VersionChip label="Cordium" version={request.packageCordium.version} />
      )}
    </div>
  );
};

const UpgradeHistory = ({
  items,
}: {
  items: ClusterConfig_Status_UpgradeRequest[];
}) => {
  const [expanded, setExpanded] = React.useState(false);

  const hasMore = items.length > HISTORY_PREVIEW_COUNT;
  const visible = expanded ? items : items.slice(0, HISTORY_PREVIEW_COUNT);

  return (
    <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50/60">
        <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
          Upgrade history
        </span>
        <span className="text-[0.65rem] font-bold px-2 py-0.5 rounded-full bg-slate-100 text-slate-500 border border-slate-200">
          {items.length}
        </span>
      </div>

      <div
        className={twMerge(
          "flex flex-col",
          expanded && items.length > 6 && "max-h-[340px] overflow-y-auto",
        )}
      >
        {visible.map((item, idx) => (
          <div
            key={idx}
            className="flex items-center justify-between gap-3 px-4 py-2.5 border-b border-slate-100 last:border-b-0"
          >
            <div className="flex items-center gap-2 min-w-0 flex-wrap">
              <UpgradeStateBadge state={item.state} />
              <VersionChips request={item.request} />
            </div>
            <span className="text-[0.7rem] font-semibold text-slate-400 shrink-0">
              <TimeAgo rfc3339={item.doneAt ?? item.createdAt} />
            </span>
          </div>
        ))}
      </div>

      {hasMore && (
        <button
          onClick={() => setExpanded((v) => !v)}
          className="w-full flex items-center justify-center gap-1.5 px-4 py-2 border-t border-slate-100 bg-slate-50/60 text-[0.72rem] font-bold text-slate-500 hover:text-slate-900 hover:bg-slate-100 transition-colors duration-150 cursor-pointer"
        >
          {expanded ? "Show less" : `Show all ${items.length}`}
          <motion.span
            animate={{ rotate: expanded ? 180 : 0 }}
            transition={{ duration: 0.18, ease: "easeInOut" }}
            className="flex items-center"
          >
            <ChevronDown size={12} strokeWidth={2.5} />
          </motion.span>
        </button>
      )}
    </div>
  );
};

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
}) => {
  const [customVersion, setCustomVersion] = React.useState(false);

  React.useEffect(() => {
    if (!enabled && customVersion) {
      setCustomVersion(false);
    }
  }, [enabled]);

  return (
    <div
      className={twMerge(
        "rounded-lg border transition-[border-color,background] duration-150",
        enabled
          ? "border-slate-300 bg-slate-50/60"
          : "border-slate-200 bg-white",
      )}
    >
      <div className="flex items-start justify-between px-4 py-3 gap-4">
        <div className="flex flex-col gap-0.5 min-w-0">
          <span className="text-[0.78rem] font-bold text-slate-800">
            {label}
          </span>
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
            <div className="px-4 pb-4 border-t border-slate-200 pt-3 flex flex-col gap-3">
              <Checkbox
                size="xs"
                checked={customVersion}
                onChange={(e) => {
                  const checked = e.currentTarget.checked;
                  setCustomVersion(checked);
                  if (!checked) {
                    onVersionChange("");
                  }
                }}
                label={
                  <span className="text-[0.74rem] font-semibold text-slate-600">
                    Override the default latest version
                  </span>
                }
              />

              <AnimatePresence initial={false}>
                {customVersion && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.15 }}
                    className="overflow-hidden"
                  >
                    <TextInput
                      label="Version"
                      placeholder="e.g. 1.2.3"
                      value={version}
                      onChange={(e) => onVersionChange(e.target.value)}
                      styles={{
                        input: {
                          fontSize: "0.78rem",
                        },
                      }}
                    />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

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
      <Button
        variant="filled"
        color="dark"
        leftSection={<ArrowUpCircle size={15} strokeWidth={2.5} />}
        onClick={open}
      >
        Upgrade cluster
      </Button>

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
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              onClick={handleClose}
            >
              <X size={13} strokeWidth={2.5} />
            </ActionIcon>
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
                Each component upgrades to the latest version unless you
                override it.
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
              description="Cordium: the sandbox platform package"
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
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              disabled={mutationUpgrade.isPending}
              onClick={handleClose}
            >
              Cancel
            </Button>

            <Button
              variant="filled"
              color="dark"
              size="sm"
              leftSection={<ArrowUpCircle size={13} strokeWidth={2.5} />}
              disabled={!confirmed || !hasSelection}
              loading={mutationUpgrade.isPending}
              onClick={() => mutationUpgrade.mutate()}
            >
              {mutationUpgrade.isPending ? "Upgrading..." : "Upgrade cluster"}
            </Button>
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
    refetchInterval: 5000,
  });

  if (!qry.isSuccess || !qry.data) return null;

  const cluster = qry.data.response;
  const status = cluster.status!;
  const current = status.upgradeRequest;
  const history = status.lastUpgradeRequests;
  const last = history[0];
  const isIdle = !current && history.length === 0;

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

      {history.length > 0 && <UpgradeHistory items={history} />}

      {isIdle && (
        <div className="flex items-center justify-center py-6">
          <span className="text-[0.75rem] font-semibold text-slate-400">
            No upgrades have been performed yet
          </span>
        </div>
      )}

      <div className="my-8">
        <ClusterVersionInfo />
      </div>

      <div className="flex justify-end">
        <UpgradeCluster />
      </div>
    </div>
  );
};
