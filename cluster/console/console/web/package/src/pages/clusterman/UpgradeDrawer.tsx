import {
  UpgradeClusterRequest,
  UpgradeClusterRequest_Request_Core,
  UpgradeClusterRequest_Request_PackageCordium,
  UpgradeClusterRequest_Request_PackageEnterprise,
} from "@/apis/enterprisev1/enterprisev1";
import { getDomain, onError } from "@/utils";
import { getClientCluster } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import {
  Button,
  Checkbox,
  Drawer,
  SegmentedControl,
  TextInput,
} from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowUpCircle,
  Check,
  CheckCircle2,
  ListChecks,
  Sparkles,
  TriangleAlert,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { VersionDelta } from "./components";
import {
  clustermanKeys,
  PackageInfo,
  PackageKey,
  PACKAGES,
  packageInfo,
  useClusterVersionInfo,
} from "./queries";

type Choice = { selected: boolean; custom: boolean; version: string };

type Choices = Record<PackageKey, Choice>;

const EMPTY: Choice = { selected: false, custom: false, version: "" };

const emptyChoices = (): Choices => ({
  core: { ...EMPTY },
  packageEnterprise: { ...EMPTY },
  packageCordium: { ...EMPTY },
});

const targetVersion = (choice: Choice, info?: PackageInfo) =>
  choice.custom ? choice.version.trim() : (info?.latestVersion ?? "");

const pinned = (choice: Choice) => (choice.custom ? choice.version.trim() : "");

const buildRequest = (choices: Choices) =>
  UpgradeClusterRequest.create({
    request: {
      core: choices.core.selected
        ? UpgradeClusterRequest_Request_Core.create({
            version: pinned(choices.core),
          })
        : undefined,
      packageEnterprise: choices.packageEnterprise.selected
        ? UpgradeClusterRequest_Request_PackageEnterprise.create({
            version: pinned(choices.packageEnterprise),
          })
        : undefined,
      packageCordium: choices.packageCordium.selected
        ? UpgradeClusterRequest_Request_PackageCordium.create({
            version: pinned(choices.packageCordium),
          })
        : undefined,
    },
  });

const PackageOption = (props: {
  label: string;
  description: string;
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  info?: PackageInfo;
  choice: Choice;
  onChange: (choice: Choice) => void;
}) => {
  const { choice, info } = props;
  const Icon = props.icon;
  const target = targetVersion(choice, info);
  const missing = choice.selected && choice.custom && target === "";
  const changes = target !== "" && target !== info?.currentVersion;

  return (
    <div
      className={twMerge(
        "overflow-hidden rounded-xl border bg-white transition-[border-color,box-shadow] duration-150",
        choice.selected
          ? "border-slate-400 shadow-card"
          : "border-slate-200 hover:border-slate-300",
      )}
    >
      <button
        type="button"
        aria-pressed={choice.selected}
        onClick={() =>
          props.onChange({ ...choice, selected: !choice.selected })
        }
        className="flex w-full cursor-pointer items-start gap-3 px-4 py-3.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <span
          aria-hidden="true"
          className={twMerge(
            "mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border transition-colors duration-150",
            choice.selected
              ? "border-slate-900 bg-slate-900 text-white"
              : "border-slate-300 bg-white",
          )}
        >
          {choice.selected && <Check size={11} strokeWidth={3} />}
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="flex min-w-0 items-center gap-2">
            <Icon size={13} strokeWidth={2.2} />
            <span className="truncate text-body font-semibold text-slate-800">
              {props.label}
            </span>
          </span>
          <span className="text-micro font-normal text-slate-500">
            {props.description}
          </span>
          <span className="mt-1.5 flex min-w-0">
            {changes ? (
              <VersionDelta
                from={info?.currentVersion}
                to={target}
                highlight={choice.selected}
              />
            ) : (
              <span className="truncate text-body font-semibold tabular-nums text-slate-700">
                {info?.currentVersion || "Unknown"}
              </span>
            )}
          </span>
        </span>

        {info?.canUpgrade ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-micro font-semibold text-blue-700">
            <Sparkles size={10} strokeWidth={2.5} />
            Update available
          </span>
        ) : (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold text-slate-500">
            <CheckCircle2 size={10} strokeWidth={2.5} />
            Current
          </span>
        )}
      </button>

      <AnimatePresence initial={false}>
        {choice.selected && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.18, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <div className="flex flex-col gap-2.5 border-t border-slate-200 bg-slate-50/60 px-4 py-3">
              <SegmentedControl
                size="xs"
                fullWidth
                value={choice.custom ? "custom" : "latest"}
                onChange={(value) =>
                  props.onChange({
                    ...choice,
                    custom: value === "custom",
                    version: value === "custom" ? choice.version : "",
                  })
                }
                data={[
                  {
                    value: "latest",
                    label: info?.latestVersion
                      ? `Latest (${info.latestVersion})`
                      : "Latest",
                  },
                  { value: "custom", label: "Pin a version" },
                ]}
              />

              <AnimatePresence initial={false}>
                {choice.custom && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.15, ease: "easeOut" }}
                    className="overflow-hidden"
                  >
                    <TextInput
                      size="xs"
                      placeholder="e.g. 1.2.3"
                      aria-label={`${props.label} version`}
                      value={choice.version}
                      error={missing ? "A version is required" : undefined}
                      onChange={(event) =>
                        props.onChange({
                          ...choice,
                          version: event.currentTarget.value,
                        })
                      }
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

const UpgradeDrawer = (props: {
  opened: boolean;
  onClose: () => void;
  preselect?: PackageKey;
}) => {
  const versions = useClusterVersionInfo();
  const [choices, setChoices] = React.useState<Choices>(emptyChoices());
  const [confirmed, setConfirmed] = React.useState(false);

  const data = versions.data;

  React.useEffect(() => {
    if (!props.opened) return;

    const next = emptyChoices();
    for (const item of PACKAGES) {
      next[item.key].selected = props.preselect
        ? item.key === props.preselect
        : !!packageInfo(data, item.key)?.canUpgrade;
    }
    setChoices(next);
    setConfirmed(false);
  }, [props.opened, props.preselect]);

  const plan = PACKAGES.filter((item) => choices[item.key].selected).map(
    (item) => {
      const info = packageInfo(data, item.key);
      return {
        key: item.key,
        label: item.label,
        from: info?.currentVersion,
        to: targetVersion(choices[item.key], info),
      };
    },
  );

  const incomplete = PACKAGES.some(
    (item) =>
      choices[item.key].selected &&
      choices[item.key].custom &&
      choices[item.key].version.trim() === "",
  );

  const mutation = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().upgradeCluster(
        buildRequest(choices),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Cluster upgrade started");
      invalidateKey(clustermanKeys.config);
      invalidateKey(clustermanKeys.info);
      props.onClose();
    },
    onError,
  });

  const handleClose = () => {
    if (mutation.isPending) return;
    props.onClose();
  };

  return (
    <Drawer
      opened={props.opened}
      onClose={handleClose}
      position="right"
      size="min(720px, 100vw)"
      padding={0}
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      transitionProps={{
        transition: "slide-left",
        duration: 260,
        exitDuration: 220,
      }}
      title={
        <div className="flex min-w-0 flex-col">
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            Cluster
          </span>
          <span className="truncate text-sm font-bold text-slate-900">
            Upgrade {getDomain()}
          </span>
        </div>
      }
      styles={{
        header: {
          borderBottom: "1px solid var(--color-slate-200)",
          minHeight: "56px",
          paddingInline: "16px",
        },
        body: {
          height: "calc(100dvh - 56px)",
          padding: 0,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "var(--color-slate-50)",
        },
      }}
    >
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4">
        <div className="flex items-start gap-2.5 rounded-xl border border-amber-200 bg-amber-50/60 px-3.5 py-3">
          <TriangleAlert
            size={14}
            strokeWidth={2.4}
            className="mt-0.5 shrink-0 text-amber-600"
          />
          <p className="text-xs font-normal leading-5 text-amber-800">
            The selected components roll out one by one and their workloads
            restart while the rollout is applied. Sessions stay authenticated,
            but in-flight connections to the affected Services may drop.
          </p>
        </div>

        <section className="flex flex-col gap-2.5">
          <h3 className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            Components
          </h3>

          {PACKAGES.map((item) => (
            <PackageOption
              key={item.key}
              label={item.label}
              description={item.description}
              icon={item.icon}
              info={packageInfo(data, item.key)}
              choice={choices[item.key]}
              onChange={(choice) =>
                setChoices((current) => ({ ...current, [item.key]: choice }))
              }
            />
          ))}
        </section>

        <section className="flex flex-col gap-2.5">
          <h3 className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            <ListChecks size={12} strokeWidth={2.3} />
            Upgrade plan
          </h3>

          {plan.length === 0 ? (
            <div className="flex min-h-16 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white px-4 text-center">
              <p className="text-xs font-normal text-slate-500">
                Select at least one component to upgrade.
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
              {plan.map((item) => (
                <div
                  key={item.key}
                  className="flex min-w-0 items-center justify-between gap-3 py-1"
                >
                  <span className="truncate text-body font-semibold text-slate-700">
                    {item.label}
                  </span>
                  {item.to === item.from ? (
                    <span className="shrink-0 text-micro font-semibold text-slate-500">
                      Reinstalls {item.from || "the current version"}
                    </span>
                  ) : (
                    <VersionDelta from={item.from} to={item.to} highlight />
                  )}
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      <footer className="flex flex-col gap-3 border-t border-slate-200 bg-white px-4 py-3.5">
        <Checkbox
          size="xs"
          checked={confirmed}
          onChange={(event) => setConfirmed(event.currentTarget.checked)}
          label={
            <span className="text-xs font-normal text-slate-600">
              I understand that the selected components will restart during the
              rollout.
            </span>
          }
        />

        <div className="flex items-center justify-end gap-2">
          <Button
            variant="default"
            size="sm"
            leftSection={<X size={13} strokeWidth={2.5} />}
            disabled={mutation.isPending}
            onClick={handleClose}
          >
            Cancel
          </Button>

          <Button
            variant="filled"
            color="ink"
            size="sm"
            leftSection={<ArrowUpCircle size={13} strokeWidth={2.5} />}
            disabled={!confirmed || plan.length === 0 || incomplete}
            loading={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Starting…" : "Start upgrade"}
          </Button>
        </div>
      </footer>
    </Drawer>
  );
};

export default UpgradeDrawer;
