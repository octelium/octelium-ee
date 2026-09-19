import { Panel } from "@/components/Dashboard/components";
import TimeAgo from "@/components/TimeAgo";
import { ActionIcon, Alert, Button, Skeleton, Tooltip } from "@mantine/core";
import {
  ArrowUpCircle,
  Boxes,
  CheckCircle2,
  RefreshCw,
  Sparkles,
} from "lucide-react";
import { VersionDelta } from "./components";
import {
  PackageInfo,
  PackageKey,
  PACKAGES,
  packageInfo,
  upgradableCount,
  useClusterVersionInfo,
} from "./queries";

const PackageCard = (props: {
  label: string;
  description: string;
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  info: PackageInfo;
  disabled: boolean;
  onUpgrade: () => void;
}) => {
  const { info } = props;
  const Icon = props.icon;

  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3.5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <Icon size={14} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <div className="truncate text-body font-semibold text-slate-800">
              {props.label}
            </div>
            <div className="truncate text-micro font-normal text-slate-500">
              {props.description}
            </div>
          </div>
        </div>

        {info.canUpgrade ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-micro font-semibold text-blue-700">
            <Sparkles size={10} strokeWidth={2.5} />
            Update
          </span>
        ) : (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-micro font-semibold text-emerald-700">
            <CheckCircle2 size={10} strokeWidth={2.5} />
            Current
          </span>
        )}
      </div>

      <div className="rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2.5">
        <div className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {info.canUpgrade ? "Installed → available" : "Installed"}
        </div>
        <div className="mt-0.5">
          {info.canUpgrade ? (
            <VersionDelta
              from={info.currentVersion}
              to={info.latestVersion}
              highlight
            />
          ) : (
            <span className="truncate text-body font-semibold tabular-nums text-slate-700">
              {info.currentVersion || "Unknown"}
            </span>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-micro font-normal text-slate-500">
          {info.setAt ? (
            <>
              Installed <TimeAgo rfc3339={info.setAt} />
            </>
          ) : (
            "Installation date unknown"
          )}
        </span>

        {info.canUpgrade && (
          <Button
            variant="default"
            size="compact-xs"
            disabled={props.disabled}
            leftSection={<ArrowUpCircle size={11} strokeWidth={2.5} />}
            onClick={props.onUpgrade}
          >
            Upgrade
          </Button>
        )}
      </div>
    </div>
  );
};

const Packages = (props: {
  upgradeActive: boolean;
  onUpgrade: (key: PackageKey) => void;
}) => {
  const versions = useClusterVersionInfo();
  const updates = upgradableCount(versions.data);

  const items = PACKAGES.map((item) => ({
    meta: item,
    info: packageInfo(versions.data, item.key),
  })).filter((item) => !!item.info);

  return (
    <Panel
      icon={Boxes}
      title="Packages"
      description="Installed and available versions reported by the Cluster"
      actions={
        <>
          {versions.data && (
            <span className="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold text-slate-600">
              {updates > 0
                ? `${updates} update${updates === 1 ? "" : "s"}`
                : "All current"}
            </span>
          )}
          <Tooltip label="Re-check versions" withArrow>
            <ActionIcon
              type="button"
              variant="default"
              size="sm"
              aria-label="Re-check the package versions"
              disabled={versions.isFetching}
              onClick={() => versions.refetch()}
            >
              <RefreshCw
                size={12}
                strokeWidth={2.5}
                className={versions.isFetching ? "animate-spin" : ""}
              />
            </ActionIcon>
          </Tooltip>
        </>
      }
    >
      {versions.isLoading ? (
        <div className="grid gap-3 md:grid-cols-3">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} height={158} radius="lg" />
          ))}
        </div>
      ) : versions.isError ? (
        <Alert color="red" title="Could not load the package versions">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs">{versions.error.message}</span>
            <Button
              size="compact-xs"
              variant="outline"
              onClick={() => versions.refetch()}
            >
              Try again
            </Button>
          </div>
        </Alert>
      ) : (
        <div className="grid gap-3 md:grid-cols-3">
          {items.map((item) => (
            <PackageCard
              key={item.meta.key}
              label={item.meta.label}
              description={item.meta.description}
              icon={item.meta.icon}
              info={item.info!}
              disabled={props.upgradeActive}
              onUpgrade={() => props.onUpgrade(item.meta.key)}
            />
          ))}
        </div>
      )}
    </Panel>
  );
};

export default Packages;
