import {
  ClusterConfig_Status_LicenseInfo,
  ClusterConfig_Status_LicenseInfo_State,
  DeleteLicenseRequest,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import CopyText from "@/components/CopyText";
import { Panel } from "@/components/Dashboard/components";
import TimeAgo from "@/components/TimeAgo";
import { onError } from "@/utils";
import { getClientCluster } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Alert, Button, Modal, Skeleton, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  BadgeCheck,
  Building2,
  Globe,
  ShieldOff,
  Trash2,
  TriangleAlert,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { Field, LICENSE_TYPE_LABEL, LicenseStateBadge } from "./components";
import LicenseDrawer from "./LicenseDrawer";
import { clustermanKeys, useLicense } from "./queries";

const formatDate = (value?: Timestamp) =>
  value ? dayjs(Timestamp.toDate(value)).format("MMM D, YYYY") : "—";

const RemoveLicense = (props: { opened: boolean; onClose: () => void }) => {
  const [confirmed, setConfirmed] = React.useState(false);

  React.useEffect(() => {
    if (!props.opened) setConfirmed(false);
  }, [props.opened]);

  const mutation = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().deleteLicense(
        DeleteLicenseRequest.create({}),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("License removed");
      invalidateKey(clustermanKeys.license);
      invalidateKey(clustermanKeys.config);
      props.onClose();
    },
    onError,
  });

  return (
    <Modal
      opened={props.opened}
      onClose={() => !mutation.isPending && props.onClose()}
      centered
      size="md"
      withCloseButton={false}
      padding={0}
      styles={{
        content: {
          borderRadius: "12px",
          border: "1px solid var(--color-slate-200)",
          overflow: "hidden",
        },
      }}
    >
      <div className="flex flex-col">
        <div className="flex items-start gap-3 px-5 py-4">
          <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 text-red-600">
            <TriangleAlert size={16} strokeWidth={2.3} />
          </span>
          <div className="min-w-0">
            <div className="text-sm font-bold text-slate-900">
              Remove the Cluster license
            </div>
            <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
              The Cluster keeps running, but the enterprise features stop being
              entitled until a new license is set.
            </p>
          </div>
        </div>

        <div className="px-5 pb-4">
          <Switch
            checked={confirmed}
            onChange={(event) => setConfirmed(event.currentTarget.checked)}
            color="red"
            size="sm"
            label={
              <span className="text-body font-normal text-slate-600">
                Yes, remove the license.
              </span>
            }
          />
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
          <Button
            variant="default"
            size="sm"
            leftSection={<X size={13} strokeWidth={2.5} />}
            disabled={mutation.isPending}
            onClick={props.onClose}
          >
            Cancel
          </Button>
          <Button
            color="red.8"
            size="sm"
            leftSection={<Trash2 size={13} strokeWidth={2.5} />}
            disabled={!confirmed}
            loading={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Remove license
          </Button>
        </div>
      </div>
    </Modal>
  );
};

const License = (props: { info?: ClusterConfig_Status_LicenseInfo }) => {
  const query = useLicense();
  const [drawerOpened, drawer] = useDisclosure(false);
  const [removeOpened, remove] = useDisclosure(false);

  const state =
    query.data?.state ??
    props.info?.state ??
    ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN;
  const license = query.data?.license;
  const hasLicense =
    state !== ClusterConfig_Status_LicenseInfo_State.NONE &&
    state !== ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN;

  return (
    <Panel
      icon={BadgeCheck}
      title="License"
      description="The commercial entitlement that unlocks the enterprise features"
      actions={
        <>
          {!query.isLoading && <LicenseStateBadge state={state} />}
          <Button
            variant="default"
            size="compact-xs"
            leftSection={<BadgeCheck size={11} strokeWidth={2.5} />}
            onClick={drawer.open}
          >
            {hasLicense ? "Replace" : "Set license"}
          </Button>
          {hasLicense && (
            <Button
              variant="default"
              color="red"
              size="compact-xs"
              leftSection={<Trash2 size={11} strokeWidth={2.5} />}
              onClick={remove.open}
            >
              Remove
            </Button>
          )}
        </>
      }
    >
      {query.isLoading ? (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2, 3, 4, 5].map((index) => (
            <Skeleton key={index} height={56} radius="md" />
          ))}
        </div>
      ) : query.isError ? (
        <Alert color="red" title="Could not load the license">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs">{query.error.message}</span>
            <Button
              size="compact-xs"
              variant="outline"
              onClick={() => query.refetch()}
            >
              Try again
            </Button>
          </div>
        </Alert>
      ) : !hasLicense ? (
        <div className="flex min-h-24 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-6 py-6 text-center">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500">
            <ShieldOff size={17} strokeWidth={2.2} />
          </span>
          <span className="text-body font-semibold text-slate-800">
            No license is set
          </span>
          <span className="max-w-md text-micro font-normal text-slate-500">
            Set the license issued for this Cluster domain to entitle the
            enterprise features.
          </span>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <Field label="Organization">
              <span className="inline-flex min-w-0 items-center gap-1.5">
                <Building2
                  size={12}
                  strokeWidth={2.3}
                  className="shrink-0 text-slate-500"
                />
                <span className="truncate">
                  {license?.organization?.displayName || "—"}
                </span>
              </span>
            </Field>
            <Field label="Type">
              {LICENSE_TYPE_LABEL[license?.type ?? 0] ?? "Unknown"}
            </Field>
            <Field label="Expires">
              {license?.notAfter ? (
                <span className="inline-flex items-center gap-1.5">
                  {formatDate(license.notAfter)}
                  <span className="text-micro font-normal text-slate-500">
                    <TimeAgo rfc3339={license.notAfter} />
                  </span>
                </span>
              ) : (
                "Never"
              )}
            </Field>
            <Field label="Valid from">{formatDate(license?.notBefore)}</Field>
            <Field label="Issued">{formatDate(license?.issuedAt)}</Field>
            <Field label="Set">
              {query.data?.setAt ? <TimeAgo rfc3339={query.data.setAt} /> : "—"}
            </Field>
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
            <span className="inline-flex items-center gap-1.5 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              <Globe size={12} strokeWidth={2.3} />
              Allowed domains
            </span>
            {(license?.allowedClusterDomains.length ?? 0) === 0 ? (
              <span className="text-body font-semibold text-slate-700">
                Any Cluster domain
              </span>
            ) : (
              license!.allowedClusterDomains.map((domain) => (
                <span
                  key={domain}
                  className="rounded border border-slate-200 bg-white px-1.5 py-0.5 text-micro font-semibold text-slate-700"
                >
                  {domain}
                </span>
              ))
            )}
          </div>

          {license?.uid && (
            <div className="flex flex-wrap items-center gap-2 text-micro font-normal text-slate-500">
              <span className="font-semibold uppercase tracking-[0.07em]">
                License ID
              </span>
              <CopyText value={license.uid} truncate={20} />
              {props.info?.stateCheckedAt && (
                <span>
                  · checked <TimeAgo rfc3339={props.info.stateCheckedAt} />
                </span>
              )}
            </div>
          )}
        </div>
      )}

      <LicenseDrawer
        opened={drawerOpened}
        onClose={drawer.close}
        hasLicense={hasLicense}
      />
      <RemoveLicense opened={removeOpened} onClose={remove.close} />
    </Panel>
  );
};

export default License;
