import Meta from "@/components/Meta";
import { getDomain } from "@/utils";
import { ActionIcon, Alert, Button, Tooltip } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  ArrowUpCircle,
  CheckCircle2,
  Loader2,
  RefreshCw,
  ServerCog,
  Sparkles,
} from "lucide-react";
import * as React from "react";
import License from "./License";
import Packages from "./Packages";
import {
  isUpgradeActive,
  PackageKey,
  upgradableCount,
  useClusterConfig,
  useClusterVersionInfo,
} from "./queries";
import Rollout from "./Rollout";
import UpgradeDrawer from "./UpgradeDrawer";

const StatusPill = (props: { active: boolean; updates: number }) => {
  if (props.active) {
    return (
      <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-micro font-semibold text-blue-700">
        <Loader2 size={10} className="animate-spin" />
        Upgrading
      </span>
    );
  }

  if (props.updates > 0) {
    return (
      <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-micro font-semibold text-blue-700">
        <Sparkles size={10} strokeWidth={2.5} />
        {props.updates} update{props.updates === 1 ? "" : "s"} available
      </span>
    );
  }

  return (
    <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-micro font-semibold text-emerald-700">
      <CheckCircle2 size={10} strokeWidth={2.5} />
      Up to date
    </span>
  );
};

const Main = () => {
  const config = useClusterConfig();
  const versions = useClusterVersionInfo();
  const queryClient = useQueryClient();
  const fetching = useIsFetching();

  const [opened, drawer] = useDisclosure(false);
  const [preselect, setPreselect] = React.useState<PackageKey | undefined>();

  const status = config.data?.status;
  const active = isUpgradeActive(status?.upgradeRequest?.state);
  const updates = upgradableCount(versions.data);

  const openUpgrade = (key?: PackageKey) => {
    setPreselect(key);
    drawer.open();
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex w-full flex-col gap-4 py-4"
    >
      <Meta title="Cluster management" />

      <header className="rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-card sm:px-5">
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
              <ServerCog size={18} strokeWidth={2.1} />
            </span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-xl font-bold tracking-[-0.02em] text-slate-950">
                  Cluster management
                </h1>
                {!versions.isLoading && (
                  <StatusPill active={active} updates={updates} />
                )}
              </div>
              <p className="mt-1 text-xs font-normal leading-5 text-slate-500">
                Package versions, upgrade rollouts and the license of{" "}
                <span className="font-semibold text-slate-700">
                  {getDomain()}
                </span>
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            {fetching > 0 && (
              <span className="text-micro font-normal tabular-nums text-slate-500">
                Updating {fetching}…
              </span>
            )}

            <Tooltip label="Refresh everything" withArrow>
              <ActionIcon
                type="button"
                variant="default"
                size="sm"
                aria-label="Refresh the page"
                onClick={() => queryClient.invalidateQueries()}
              >
                <RefreshCw
                  size={12}
                  strokeWidth={2.5}
                  className={fetching > 0 ? "animate-spin" : ""}
                />
              </ActionIcon>
            </Tooltip>

            <Button
              variant="filled"
              color="ink"
              leftSection={<ArrowUpCircle size={15} strokeWidth={2.5} />}
              disabled={active || versions.isLoading}
              onClick={() => openUpgrade(undefined)}
            >
              {active
                ? "Upgrade in progress"
                : versions.isLoading
                  ? "Checking versions…"
                  : "Upgrade Cluster"}
            </Button>
          </div>
        </div>
      </header>

      <Packages upgradeActive={active} onUpgrade={openUpgrade} />

      {config.isError ? (
        <Alert color="red" title="Could not load the Cluster status">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs">
              {config.error?.message ?? "The Cluster API returned no status."}
            </span>
            <Button
              size="compact-xs"
              variant="outline"
              onClick={() => config.refetch()}
            >
              Try again
            </Button>
          </div>
        </Alert>
      ) : status ? (
        <>
          <Rollout status={status} />
          <License info={status.licenseInfo} />
        </>
      ) : (
        <div
          aria-hidden="true"
          className="h-72 w-full animate-pulse rounded-xl border border-slate-200 bg-slate-50/70"
        />
      )}

      <UpgradeDrawer
        opened={opened}
        onClose={drawer.close}
        preselect={preselect}
      />
    </motion.div>
  );
};

export default Main;
