import {
  Certificate,
  Certificate_Spec_Mode,
  Certificate_Status_Issuance_State,
} from "@/apis/enterprisev1/enterprisev1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
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
import {
  Activity,
  AlertTriangle,
  Clock3,
  FileBadge,
  Globe2,
  Landmark,
  Loader2,
  RefreshCcw,
  Server,
  ShieldCheck,
  Upload,
  X,
} from "lucide-react";
import CopyText from "@/components/CopyText";
import TimeAgo, { useNow } from "@/components/TimeAgo";
import {
  formatSeconds,
  HeroStat,
  HistoryList,
  Meter,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  remainingPercent,
  ResourceActionButton,
  toDate,
  TonePill,
} from "@/components/ResourceLayout/overview";
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
      <ResourceActionButton
        tone="primary"
        icon={RefreshCcw}
        loading={presentation.isInProgress}
        onClick={open}
      >
        {presentation.successfulIssuance ? "Re-issue" : "Issue certificate"}
      </ResourceActionButton>
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
                <h2 className="text-sm font-bold text-slate-900">
                  {presentation.successfulIssuance ? "Re-issue" : "Issue"} certificate
                </h2>
                <p className="text-xs font-normal text-slate-500">
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
                <div className="text-micro font-semibold uppercase tracking-wide text-slate-500">Issuer</div>
                <div className="mt-1 text-xs font-semibold text-slate-700">
                  {item.status?.certificateIssuerRef?.name || "Cluster default"}
                </div>
              </div>
              <div className="bg-white p-3">
                <div className="text-micro font-semibold uppercase tracking-wide text-slate-500">Current validity</div>
                <div className="mt-1 text-xs font-semibold capitalize text-slate-700">
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
              color="ink"
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
              <Loader2 size={14} className="mr-1 motion-safe:animate-spin" />
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

const issuanceTone = (state?: Certificate_Status_Issuance_State) =>
  match(state)
    .with(Certificate_Status_Issuance_State.SUCCESS, () => "success" as const)
    .with(Certificate_Status_Issuance_State.FAILED, () => "danger" as const)
    .with(
      Certificate_Status_Issuance_State.ISSUING,
      Certificate_Status_Issuance_State.ISSUANCE_REQUESTED,
      () => "info" as const,
    )
    .otherwise(() => "neutral" as const);

const expiryTone = (state: string) =>
  state === "valid"
    ? ("success" as const)
    : state === "expiring"
      ? ("warning" as const)
      : state === "expired"
        ? ("danger" as const)
        : ("neutral" as const);

const MAX_DNS_NAMES = 4;

const CertificateHero = (props: { item: Certificate }) => {
  const { item } = props;
  const now = useNow();
  const p = getCertificatePresentation(item);
  const info = item.status?.info;
  const notBefore = toDate(info?.notBefore ?? p.successfulIssuance?.issuanceCompletedAt);
  const left = p.expiresAt ? (p.expiresAt.getTime() - now) / 1000 : undefined;
  const validity = remainingPercent(notBefore, p.expiresAt, now);
  const isManaged = item.spec?.mode === Certificate_Spec_Mode.MANAGED;
  const tone = p.isInProgress && p.expiryState === "unknown" ? "info" : expiryTone(p.expiryState);
  const dnsNames = info?.dnsNames ?? [];
  const failed =
    p.issuance?.state === Certificate_Status_Issuance_State.FAILED;
  const expiresAtTimestamp =
    p.successfulIssuance?.expiresAt ?? info?.notAfter;

  return (
    <OverviewHero
      tone={failed && p.expiryState === "unknown" ? "danger" : tone}
      icon={<ShieldCheck size={18} strokeWidth={2.2} />}
      title={info?.commonName || dnsNames[0] || item.metadata?.name}
      detail={[
        isManaged
          ? `Managed by ${item.status?.certificateIssuerRef?.name || "the default CertificateIssuer"}`
          : "Manually provided",
        info?.issuer && `issued by ${info.issuer}`,
      ]
        .filter(Boolean)
        .join(" · ")}
      meta={
        dnsNames.length > 0 && (
          <>
            {dnsNames.slice(0, MAX_DNS_NAMES).map((name) => (
              <TonePill key={name} icon={Globe2}>
                {name}
              </TonePill>
            ))}
            {dnsNames.length > MAX_DNS_NAMES && (
              <TonePill>+{dnsNames.length - MAX_DNS_NAMES} more</TonePill>
            )}
          </>
        )
      }
      meter={
        validity !== undefined &&
        left !== undefined &&
        left > 0 && (
          <Meter
            value={validity}
            label="Certificate validity left"
            tone={p.expiryState === "expiring" ? "warning" : "success"}
            caption={`${formatSeconds(left)} of validity left`}
          />
        )
      }
      aside={
        <HeroStat
          label={left !== undefined && left <= 0 ? "Expired" : "Expires"}
          tone={expiryTone(p.expiryState)}
          value={
            expiresAtTimestamp ? (
              <TimeAgo rfc3339={expiresAtTimestamp} />
            ) : (
              "Unknown"
            )
          }
        />
      }
    >
      {p.isInProgress && (
        <OverviewNotice tone="info" icon={Loader2} title="Issuance in progress.">
          The CertificateIssuer is issuing a new certificate.
        </OverviewNotice>
      )}
      {failed && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="The last issuance failed.">
          {p.successfulIssuance
            ? "The previous certificate is still served."
            : "No certificate is served yet."}
        </OverviewNotice>
      )}
      {p.expiryState === "expired" && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="Expired.">
          {isManaged ? "Re-issue the certificate" : "Set a new certificate"} to
          restore TLS for the associated resource.
        </OverviewNotice>
      )}
      {p.expiryState === "expiring" && !isManaged && (
        <OverviewNotice tone="warning" icon={Clock3} title="Expiring soon.">
          Set a renewed certificate before it expires.
        </OverviewNotice>
      )}
      {p.expiryState === "unknown" && !p.isInProgress && !failed && (
        <OverviewNotice tone="warning" icon={ShieldCheck} title="No certificate yet.">
          {isManaged
            ? "Issue the certificate to start serving it."
            : "Set a certificate and its private key to start serving it."}
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: { item: Certificate }): ResourceMainInfo => {
  const { item } = props;
  const presentation = getCertificatePresentation(item);
  const issuance = presentation.issuance;
  const lastSuccess = presentation.successfulIssuance;
  const info = item.status?.info;
  const isManaged = item.spec?.mode === Certificate_Spec_Mode.MANAGED;
  const usedBy = item.status?.serviceRef?.name
    ? item.status.serviceRef
    : item.status?.namespaceRef?.name
      ? item.status.namespaceRef
      : undefined;
  const history = [
    ...(issuance ? [issuance] : []),
    ...(item.status?.lastIssuances ?? []),
  ]
    .sort((a, b) => {
      const aTime = a.issuanceCompletedAt ?? a.createdAt;
      const bTime = b.issuanceCompletedAt ?? b.createdAt;
      return (
        (bTime ? Number(bTime.seconds) : 0) - (aTime ? Number(aTime.seconds) : 0)
      );
    })
    .slice(0, 6);

  return {
    header: <CertificateHero item={item} />,
    actions: isManaged ? <IssueC item={item} /> : <SetCertificateC item={item} />,
    groupOrder: [
      { title: "Certificate", icon: FileBadge },
      { title: "Issuance", icon: RefreshCcw },
    ],
    items: [
      {
        label: "Mode",
        primary: true,
        icon: isManaged ? RefreshCcw : Upload,
        value: (
          <span className="flex flex-col">
            <span className="font-semibold">{presentation.mode}</span>
            <span className="text-xs font-normal text-slate-500">
              {isManaged ? "Issued via ACME" : "Uploaded by an administrator"}
            </span>
          </span>
        ),
      },
      {
        label: "Issuer",
        primary: true,
        icon: Landmark,
        value: item.status?.certificateIssuerRef?.name ? (
          <ResourceListLabel
            label="Issuer"
            itemRef={item.status.certificateIssuerRef}
          />
        ) : (
          <MutedValue>{isManaged ? "Cluster default" : "External CA"}</MutedValue>
        ),
      },
      {
        label: "Serves",
        primary: true,
        icon: Server,
        value: usedBy ? (
          <ResourceListLabel itemRef={usedBy} />
        ) : (
          <MutedValue>The Cluster</MutedValue>
        ),
      },
      {
        label: "Issuances",
        primary: true,
        icon: Activity,
        value: (
          <span className="font-semibold tabular-nums">
            {(item.status?.successfulIssuances ?? 0).toLocaleString()}
            {(item.status?.failedIssuances ?? 0) > 0 && (
              <span className="ml-1.5 text-xs font-normal text-red-600">
                {item.status!.failedIssuances.toLocaleString()} failed
              </span>
            )}
          </span>
        ),
      },

      ...(info?.commonName
        ? [
            {
              label: "Common name",
              group: "Certificate",
              value: <CopyText value={info.commonName} />,
            },
          ]
        : []),
      ...(info?.notBefore
        ? [
            {
              label: "Valid from",
              group: "Certificate",
              value: <TimeAgo rfc3339={info.notBefore} />,
            },
          ]
        : []),
      ...(info?.notAfter
        ? [
            {
              label: "Valid until",
              group: "Certificate",
              value: <TimeAgo rfc3339={info.notAfter} />,
            },
          ]
        : []),
      ...(info?.subject
        ? [
            {
              label: "Subject",
              group: "Certificate",
              span: "full" as const,
              value: info.subject,
            },
          ]
        : []),
      ...(info?.issuer
        ? [
            {
              label: "Issued by",
              group: "Certificate",
              span: "full" as const,
              value: info.issuer,
            },
          ]
        : []),
      ...(info?.dnsNames.length
        ? [
            {
              label: "DNS names",
              group: "Certificate",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {info.dnsNames.map((name) => (
                    <ResourceListLabel key={name} label="DNS">
                      <CopyText value={name} />
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),
      ...(item.status?.secretRef?.name
        ? [
            {
              label: "Stored in",
              group: "Certificate",
              value: <ResourceListLabel itemRef={item.status.secretRef} />,
            },
          ]
        : []),
      ...(!info
        ? [
            {
              label: "Details",
              group: "Certificate",
              value: <MutedValue>No certificate is available yet.</MutedValue>,
            },
          ]
        : []),

      ...(issuance
        ? [
            {
              label: "Current state",
              group: "Issuance",
              value: (
                <TonePill
                  tone={issuanceTone(issuance.state)}
                  icon={presentation.isInProgress ? Loader2 : undefined}
                >
                  {getIssuanceState(issuance.state)}
                </TonePill>
              ),
            },
          ]
        : []),
      ...(lastSuccess?.issuanceCompletedAt || lastSuccess?.createdAt
        ? [
            {
              label: "Last success",
              group: "Issuance",
              value: (
                <TimeAgo
                  rfc3339={lastSuccess.issuanceCompletedAt ?? lastSuccess.createdAt}
                />
              ),
            },
          ]
        : []),
      ...(history.length > 0
        ? [
            {
              label: "History",
              group: "Issuance",
              layout: "stacked" as const,
              value: (
                <HistoryList
                  entries={history.map((entry, index) => ({
                    key: `${entry.createdAt?.seconds ?? 0}-${entry.state}-${index}`,
                    tone: issuanceTone(entry.state),
                    label: getIssuanceState(entry.state),
                    meta: (
                      <>
                        {(entry.issuanceCompletedAt ?? entry.createdAt) && (
                          <span>
                            {entry.issuanceCompletedAt ? "Completed" : "Requested"}{" "}
                            <TimeAgo
                              rfc3339={entry.issuanceCompletedAt ?? entry.createdAt}
                            />
                          </span>
                        )}
                        {entry.expiresAt && (
                          <span>
                            Expires <TimeAgo rfc3339={entry.expiresAt} />
                          </span>
                        )}
                      </>
                    ),
                  }))}
                />
              ),
            },
          ]
        : [
            {
              label: "History",
              group: "Issuance",
              value: <MutedValue>No issuance yet</MutedValue>,
            },
          ]),
    ],
  };
};
