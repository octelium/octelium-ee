import { PILL } from "@/components/Dashboard/components";
import Meta from "@/components/Meta";
import {
  PageActions,
  PageTitleAddon,
} from "@/components/PageHeader/context";
import RefreshButton from "@/components/RefreshButton";
import { Alert, Button } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  ArrowUpCircle,
  CheckCircle2,
  Loader2,
  Sparkles,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";
import License from "./License";
import RpcError from "@/components/RpcError";
import Packages from "./Packages";
import {
  clustermanKeys,
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
      <span
        className={twMerge(PILL, "border-blue-200 bg-blue-50 text-blue-700")}
      >
        <Loader2 size={9} className="motion-safe:animate-spin" />
        Upgrading
      </span>
    );
  }

  if (props.updates > 0) {
    return (
      <span
        className={twMerge(PILL, "border-blue-200 bg-blue-50 text-blue-700")}
      >
        <Sparkles size={9} strokeWidth={2.5} />
        {props.updates} update{props.updates === 1 ? "" : "s"} available
      </span>
    );
  }

  return (
    <span
      className={twMerge(
        PILL,
        "border-emerald-200 bg-emerald-50 text-emerald-700",
      )}
    >
      <CheckCircle2 size={9} strokeWidth={2.5} />
      Up to date
    </span>
  );
};

const Main = () => {
  const config = useClusterConfig();
  const versions = useClusterVersionInfo();
  const queryClient = useQueryClient();
  const fetching = useIsFetching({ queryKey: clustermanKeys.all });

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

      <PageTitleAddon>
        {!versions.isLoading && <StatusPill active={active} updates={updates} />}
      </PageTitleAddon>

      <PageActions>
        <RefreshButton
          label="Refresh the page"
          loading={fetching > 0}
          onClick={() =>
            queryClient.invalidateQueries({ queryKey: clustermanKeys.all })
          }
        />

        <Button
          variant="filled"
          color="ink"
          size="xs"
          leftSection={<ArrowUpCircle size={14} strokeWidth={2.5} />}
          disabled={active || versions.isLoading}
          onClick={() => openUpgrade(undefined)}
        >
          {active
            ? "Upgrade in progress"
            : versions.isLoading
              ? "Checking versions…"
              : "Upgrade Cluster"}
        </Button>
      </PageActions>

      <Packages upgradeActive={active} onUpgrade={openUpgrade} />

      {config.isError ? (
        <Alert color="red" title="Could not load the Cluster status">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <RpcError error={config.error} />
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
          className="h-72 w-full motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50/70"
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
