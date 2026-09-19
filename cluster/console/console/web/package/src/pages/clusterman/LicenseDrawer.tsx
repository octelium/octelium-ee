import {
  ClusterConfig_Status_LicenseInfo_State,
  License,
  SetLicenseRequest,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { onError } from "@/utils";
import { getClientCluster } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Button, Drawer, Textarea } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  BadgeCheck,
  ScanSearch,
  ShieldCheck,
  TriangleAlert,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { Field, LICENSE_TYPE_LABEL, LicenseStateBadge } from "./components";
import { clustermanKeys } from "./queries";

const formatDate = (value?: Timestamp) =>
  value ? dayjs(Timestamp.toDate(value)).format("MMM D, YYYY") : "—";

const Preview = (props: { license?: License; state: number }) => (
  <div className="flex flex-col gap-3 rounded-xl border border-emerald-200 bg-emerald-50/50 px-3.5 py-3">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <span className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-emerald-700">
        <BadgeCheck size={12} strokeWidth={2.4} />
        Verified license
      </span>
      <LicenseStateBadge state={props.state} />
    </div>

    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <Field label="Organization">
        {props.license?.organization?.displayName || "—"}
      </Field>
      <Field label="Type">
        {LICENSE_TYPE_LABEL[props.license?.type ?? 0] ?? "Unknown"}
      </Field>
      <Field label="Valid from">{formatDate(props.license?.notBefore)}</Field>
      <Field label="Expires">{formatDate(props.license?.notAfter)}</Field>
    </div>

    {(props.license?.allowedClusterDomains.length ?? 0) > 0 && (
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          Domains
        </span>
        {props.license!.allowedClusterDomains.map((domain) => (
          <span
            key={domain}
            className="rounded border border-slate-200 bg-white px-1.5 py-0.5 text-micro font-semibold text-slate-700"
          >
            {domain}
          </span>
        ))}
      </div>
    )}
  </div>
);

const LicenseDrawer = (props: {
  opened: boolean;
  onClose: () => void;
  hasLicense: boolean;
}) => {
  const [jwt, setJwt] = React.useState("");
  const [verified, setVerified] = React.useState<
    { license?: License; state: number } | undefined
  >(undefined);

  React.useEffect(() => {
    if (props.opened) return;
    setJwt("");
    setVerified(undefined);
  }, [props.opened]);

  const verify = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().setLicense(
        SetLicenseRequest.create({ jwt: jwt.trim(), dryRun: true }),
      );
      return response;
    },
    onSuccess: (response) => {
      setVerified({ license: response.license, state: response.state });
    },
    onError: (err) => {
      setVerified(undefined);
      onError(err as any);
    },
  });

  const apply = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().setLicense(
        SetLicenseRequest.create({ jwt: jwt.trim() }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("License applied");
      invalidateKey(clustermanKeys.license);
      invalidateKey(clustermanKeys.config);
      props.onClose();
    },
    onError,
  });

  const pending = verify.isPending || apply.isPending;

  const handleClose = () => {
    if (pending) return;
    props.onClose();
  };

  const handleChange = (value: string) => {
    setJwt(value);
    setVerified(undefined);
  };

  const applicable =
    !!verified &&
    verified.state !== ClusterConfig_Status_LicenseInfo_State.EXPIRED &&
    verified.state !== ClusterConfig_Status_LicenseInfo_State.INVALID;

  return (
    <Drawer
      opened={props.opened}
      onClose={handleClose}
      position="right"
      size="min(640px, 100vw)"
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
            {props.hasLicense ? "Replace the license" : "Set the license"}
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
        <div className="flex items-start gap-2.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
          <ShieldCheck
            size={14}
            strokeWidth={2.4}
            className="mt-0.5 shrink-0 text-slate-500"
          />
          <p className="text-xs font-normal leading-5 text-slate-600">
            Paste the signed license that Octelium Labs issued for this Cluster.
            It is verified against the Cluster domain before anything is stored,
            so you can check its claims first.
          </p>
        </div>

        <Textarea
          label="License"
          placeholder="eyJhbGciOi…"
          autosize
          minRows={6}
          maxRows={12}
          value={jwt}
          onChange={(event) => handleChange(event.currentTarget.value)}
          styles={{ input: { fontFamily: "monospace", fontSize: "0.72rem" } }}
        />

        {verified ? (
          <Preview license={verified.license} state={verified.state} />
        ) : (
          <div className="flex min-h-16 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white px-4 text-center">
            <p className="text-xs font-normal text-slate-500">
              Verify the license to review its claims before applying it.
            </p>
          </div>
        )}

        {verified && !applicable && (
          <div className="flex items-start gap-2.5 rounded-xl border border-red-200 bg-red-50/60 px-3.5 py-3">
            <TriangleAlert
              size={14}
              strokeWidth={2.4}
              className="mt-0.5 shrink-0 text-red-600"
            />
            <p className="text-xs font-normal leading-5 text-red-700">
              This license cannot be applied to the Cluster in its current
              state.
            </p>
          </div>
        )}
      </div>

      <footer className="flex items-center justify-end gap-2 border-t border-slate-200 bg-white px-4 py-3.5">
        <Button
          variant="default"
          size="sm"
          leftSection={<X size={13} strokeWidth={2.5} />}
          disabled={pending}
          onClick={handleClose}
        >
          Cancel
        </Button>

        <Button
          variant="default"
          size="sm"
          leftSection={<ScanSearch size={13} strokeWidth={2.5} />}
          disabled={jwt.trim() === "" || apply.isPending}
          loading={verify.isPending}
          onClick={() => verify.mutate()}
        >
          Verify
        </Button>

        <Button
          variant="filled"
          color="ink"
          size="sm"
          leftSection={<BadgeCheck size={13} strokeWidth={2.5} />}
          disabled={!applicable || verify.isPending}
          loading={apply.isPending}
          onClick={() => apply.mutate()}
        >
          Apply license
        </Button>
      </footer>
    </Drawer>
  );
};

export default LicenseDrawer;
