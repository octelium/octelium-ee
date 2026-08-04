import {
  Certificate,
  Certificate_Spec_Mode,
  Certificate_Status_Issuance_State,
} from "@/apis/enterprisev1/enterprisev1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Alert, Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle, Loader2, RefreshCcw, ShieldCheck, X } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getCertificatePresentation, getIssuanceState } from "./List";
import { SetCertificateC } from "./SetCertificateC";

export const IssueC = (props: { item: Certificate }) => {
  const { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const presentation = getCertificatePresentation(item);
  const mutationGenerate = useMutation({
    mutationFn: async () => {
      const { response } = await getClientEnterprise().issueCertificate({
        certificateRef: getResourceRef(item),
      });
      return response;
    },

    onSuccess: () => {
      invalidateResource(item);
      invalidateResourceList(item);
      close();
    },
    onError: onError,
  });

  return (
    <>
      <Button
        type="button"
        size="compact-sm"
        color="dark"
        leftSection={<RefreshCcw size={13} />}
        onClick={open}
      >
        {presentation.successfulIssuance ? "Re-issue certificate" : "Issue certificate"}
      </Button>
      <Modal
        opened={opened}
        onClose={close}
        size="md"
        centered
        withCloseButton={false}
        padding={0}
        styles={{ content: { borderRadius: 14, overflow: "hidden" } }}
      >
        <div className="overflow-hidden bg-white">
          <header className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <div className="flex items-center gap-2.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900 text-white">
                <ShieldCheck size={15} />
              </span>
              <div>
                <h2 className="text-[0.84rem] font-bold text-slate-900">
                  {presentation.successfulIssuance ? "Re-issue" : "Issue"} certificate
                </h2>
                <p className="text-[0.67rem] font-semibold text-slate-400">
                  {item.metadata?.displayName || item.metadata?.name}
                </p>
              </div>
            </div>
            <Button type="button" variant="subtle" color="gray" size="compact-xs" onClick={close}>
              <X size={14} />
            </Button>
          </header>
          <div className="space-y-4 px-5 py-5">
            <Alert color="amber" icon={<AlertTriangle size={15} />}>
              A successful issuance replaces the certificate currently served
              by the associated cluster resource.
            </Alert>
            <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-slate-200 bg-slate-100">
              <div className="bg-white p-3">
                <div className="text-[0.6rem] font-bold uppercase tracking-wide text-slate-400">Issuer</div>
                <div className="mt-1 text-[0.72rem] font-semibold text-slate-700">
                  {item.status?.certificateIssuerRef?.name || "Cluster default"}
                </div>
              </div>
              <div className="bg-white p-3">
                <div className="text-[0.6rem] font-bold uppercase tracking-wide text-slate-400">Current validity</div>
                <div className="mt-1 text-[0.72rem] font-semibold capitalize text-slate-700">
                  {presentation.expiryState}
                </div>
              </div>
            </div>
          </div>
          <footer className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
            <Button type="button" variant="default" size="sm" disabled={mutationGenerate.isPending} onClick={close}>
              Cancel
            </Button>
            <Button
              type="button"
              color="dark"
              size="sm"
              onClick={() => mutationGenerate.mutate()}
              loading={mutationGenerate.isPending}
              leftSection={<RefreshCcw size={13} />}
            >
              Confirm issuance
            </Button>
          </footer>
        </div>
      </Modal>
    </>
  );
};

export const ItemInfo = (props: { item: Certificate }) => {
  const { item } = props;
  const presentation = getCertificatePresentation(item);
  const issuance = presentation.issuance;
  const lastSuccess = presentation.successfulIssuance;

  return (
    <>
      {issuance && (
        <>
          {(issuance.state === Certificate_Status_Issuance_State.ISSUING ||
            issuance.state ===
              Certificate_Status_Issuance_State.ISSUANCE_REQUESTED) && (
            <InfoItem title="Issuance State">
              <Loader2 size={14} className="mr-1 animate-spin" />
              <span>{getIssuanceState(issuance.state)}</span>
            </InfoItem>
          )}
        </>
      )}

      {lastSuccess && (
        <>
          <InfoItem title="Last Issuance">
            <TimeAgo rfc3339={lastSuccess.createdAt} />
          </InfoItem>

          <InfoItem title="Expiration">
            <TimeAgo rfc3339={lastSuccess.expiresAt} />
          </InfoItem>
        </>
      )}

      {item.spec?.mode === Certificate_Spec_Mode.MANAGED && (
        <InfoItem title="Issue">
          <IssueC item={item} />
        </InfoItem>
      )}
    </>
  );
};

export default (props: { item: Certificate }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <div className="w-full mb-8">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

export const MainInfo = (props: { item: Certificate }): ResourceMainInfo => {
  const { item } = props;
  const presentation = getCertificatePresentation(item);
  const issuance = presentation.issuance;
  const lastSuccess = presentation.successfulIssuance;

  const isInProgress =
    issuance?.state === Certificate_Status_Issuance_State.ISSUING ||
    issuance?.state === Certificate_Status_Issuance_State.ISSUANCE_REQUESTED;

  return {
    items: [
      {
        label: "Mode",
        value: (
          <span className="text-sm font-semibold text-slate-700">
            {presentation.mode}
          </span>
        ),
      },
      ...(presentation.expiryState !== "unknown"
        ? [
            {
              label: "Certificate health",
              value: (
                <span
                  className={twMerge(
                    "font-semibold capitalize",
                    presentation.expiryState === "valid"
                      ? "text-emerald-600"
                      : presentation.expiryState === "expiring"
                        ? "text-amber-600"
                        : "text-red-600",
                  )}
                >
                  {presentation.expiryState === "expiring"
                    ? "Expiring soon"
                    : presentation.expiryState}
                </span>
              ),
            },
          ]
        : []),
      ...(item.status?.certificateIssuerRef?.name
        ? [
            {
              label: "Certificate issuer",
              value: (
                <ResourceListLabel itemRef={item.status.certificateIssuerRef} />
              ),
            },
          ]
        : []),
      ...(item.status?.secretRef?.name
        ? [
            {
              label: "Certificate secret",
              value: <ResourceListLabel itemRef={item.status.secretRef} />,
            },
          ]
        : []),
      ...(item.status?.namespaceRef?.name
        ? [
            {
              label: "Namespace",
              value: (
                <ResourceListLabel
                  itemRef={item.status!.namespaceRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),
      ...(item.status?.serviceRef?.name
        ? [
            {
              label: "Service",
              value: (
                <ResourceListLabel
                  itemRef={item.status!.serviceRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),
      ...(issuance
        ? [
            {
              label: "Issuance state",
              value: (
                <span className="flex items-center gap-1.5">
                  {isInProgress && (
                    <Loader2
                      size={13}
                      strokeWidth={2.5}
                      className="animate-spin text-blue-500 shrink-0"
                    />
                  )}
                  <span
                    className={twMerge(
                      "text-sm font-semibold",
                      match(issuance.state)
                        .with(
                          Certificate_Status_Issuance_State.SUCCESS,
                          () => "text-emerald-600",
                        )
                        .with(
                          Certificate_Status_Issuance_State.FAILED,
                          () => "text-red-500",
                        )
                        .with(
                          Certificate_Status_Issuance_State.ISSUING,
                          () => "text-blue-500",
                        )
                        .with(
                          Certificate_Status_Issuance_State.ISSUANCE_REQUESTED,
                          () => "text-amber-500",
                        )
                        .otherwise(() => "text-slate-600"),
                    )}
                  >
                    {getIssuanceState(issuance.state)}
                  </span>
                </span>
              ),
            },
          ]
        : []),

      ...(issuance?.issuanceStartedAt
        ? [
            {
              label: "Issuance started",
              value: <TimeAgo rfc3339={issuance.issuanceStartedAt} />,
            },
          ]
        : []),

      ...(issuance?.issuanceCompletedAt
        ? [
            {
              label: "Issuance completed",
              value: <TimeAgo rfc3339={issuance.issuanceCompletedAt} />,
            },
          ]
        : []),

      ...(lastSuccess?.createdAt
        ? [
            {
              label: "Last issuance",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={lastSuccess.createdAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(lastSuccess?.expiresAt
        ? [
            {
              label: "Expiration",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={lastSuccess.expiresAt} />
                </span>
              ),
            },
          ]
        : []),

      ...((item.status?.successfulIssuances ?? 0) > 0
        ? [
            {
              label: "Successful issuances",
              value: item.status!.successfulIssuances.toLocaleString(),
            },
          ]
        : []),
      ...((item.status?.failedIssuances ?? 0) > 0
        ? [
            {
              label: "Failed issuances",
              value: (
                <span className="font-semibold text-red-600">
                  {item.status!.failedIssuances.toLocaleString()}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status?.info
        ? [
            {
              label: "Certificate Info",
              span: "full" as const,
              value: (
                <div className="flex flex-col gap-1.5 w-full">
                  {item.status.info.commonName && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                        Common Name
                      </span>
                      <span className="text-[0.78rem] font-semibold text-slate-700">
                        {item.status.info.commonName}
                      </span>
                    </div>
                  )}
                  {item.status.info.subject && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                        Subject
                      </span>
                      <span className="truncate text-[0.78rem] font-semibold text-slate-700">
                        {item.status.info.subject}
                      </span>
                    </div>
                  )}
                  {item.status.info.issuer && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                        Issuer
                      </span>
                      <span className="truncate text-[0.78rem] font-semibold text-slate-700">
                        {item.status.info.issuer}
                      </span>
                    </div>
                  )}
                  {item.status.info.notBefore && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                        Not Before
                      </span>
                      <span className="text-[0.78rem] font-semibold text-slate-700">
                        <TimeAgo rfc3339={item.status.info.notBefore} />
                      </span>
                    </div>
                  )}
                  {item.status.info.notAfter && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                        Not After
                      </span>
                      <span className="text-[0.78rem] font-semibold text-slate-700">
                        <TimeAgo rfc3339={item.status.info.notAfter} />
                      </span>
                    </div>
                  )}
                  {item.status.info.dnsNames.length > 0 && (
                    <div className="flex items-start gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0 pt-0.5">
                        DNS Names
                      </span>
                      <div className="flex flex-wrap gap-1">
                        {item.status.info.dnsNames.map((name) => (
                          <span
                            key={name}
                            className="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-[0.68rem] font-semibold text-slate-600"
                          >
                            {name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ),
            },
          ]
        : []),

      ...(item.status?.lastIssuances.length
        ? [
            {
              label: "Issuance history",
              span: "full" as const,
              value: (
                <div className="w-full overflow-hidden rounded-lg border border-slate-200">
                  {[...item.status.lastIssuances]
                    .sort((a, b) => {
                      const aTime = a.issuanceCompletedAt ?? a.createdAt;
                      const bTime = b.issuanceCompletedAt ?? b.createdAt;
                      return (
                        (bTime ? Number(bTime.seconds) : 0) -
                        (aTime ? Number(aTime.seconds) : 0)
                      );
                    })
                    .slice(0, 5)
                    .map((entry, index) => (
                      <div
                        key={`${entry.createdAt?.seconds ?? 0}-${entry.state}-${index}`}
                        className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0"
                      >
                        <span
                          className={twMerge(
                            "text-[0.7rem] font-bold",
                            entry.state ===
                              Certificate_Status_Issuance_State.SUCCESS
                              ? "text-emerald-600"
                              : entry.state ===
                                  Certificate_Status_Issuance_State.FAILED
                                ? "text-red-600"
                                : "text-blue-600",
                          )}
                        >
                          {getIssuanceState(entry.state)}
                        </span>
                        <div className="flex items-center gap-3 text-[0.68rem] font-semibold text-slate-500">
                          {entry.issuanceCompletedAt && (
                            <span>
                              Completed <TimeAgo rfc3339={entry.issuanceCompletedAt} />
                            </span>
                          )}
                          {entry.expiresAt && (
                            <span>
                              Expires <TimeAgo rfc3339={entry.expiresAt} />
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                </div>
              ),
            },
          ]
        : []),

      ...(item.spec?.mode === Certificate_Spec_Mode.MANAGED
        ? [
            {
              label: "Issue",
              value: <IssueC item={item} />,
            },
          ]
        : []),

      {
        label: "Set",
        value: <SetCertificateC item={item} />,
      },
    ],
  };
};
