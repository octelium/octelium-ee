import * as CoreP from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import { useUpdateResource } from "@/pages/utils/resource";
import { Select } from "@mantine/core";
import { match } from "ts-pattern";

import AccessLogViewer from "@/components/AccessLogViewer";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getResourceRef } from "@/utils/pb";
import { FaAndroid, FaApple, FaChrome, FaLinux, FaWindows } from "react-icons/fa";
import {
  Activity,
  Ban,
  Check,
  Clock3,
  Cpu,
  Gauge,
  Laptop,
  Link2,
  Lock,
  Minus,
  Radar,
  ShieldAlert,
  ShieldCheck,
  Unlink,
  UserRound,
  X,
} from "lucide-react";
import { twMerge } from "@/lib/utils";
import {
  ActionConfirmModal,
  HeroStat,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  OverviewTone,
  ResourceActionButton,
  toDate,
  TonePill,
} from "@/components/ResourceLayout/overview";
import {
  authorizationItems,
  RefLabel,
} from "@/components/ResourceLayout/overviewItems";
import { useNow } from "@/components/TimeAgo";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { invalidateResource, invalidateResourceList } from "@/utils/pb";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { notifications } from "@mantine/notifications";

export const getOSIcon = (item: CoreP.Device) => {
  return match(item.status!.osType)
    .with(CoreP.Device_Status_OSType.LINUX, () => <FaLinux />)
    .with(CoreP.Device_Status_OSType.ANDROID, () => <FaAndroid />)
    .with(CoreP.Device_Status_OSType.IOS, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.MAC, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.WINDOWS, () => <FaWindows />)
    .with(CoreP.Device_Status_OSType.CHROMEOS, () => <FaChrome />)
    .otherwise(() => undefined);
};

export const getOSStr = (item: CoreP.Device) => {
  return getOSTypeStr(item.status!.osType);
};

export const getOSTypeStr = (osType: CoreP.Device_Status_OSType) => {
  return match(osType)
    .with(CoreP.Device_Status_OSType.LINUX, () => "Linux")
    .with(CoreP.Device_Status_OSType.ANDROID, () => "Android")
    .with(CoreP.Device_Status_OSType.IOS, () => "IOS")
    .with(CoreP.Device_Status_OSType.MAC, () => "Mac OS")
    .with(CoreP.Device_Status_OSType.WINDOWS, () => "Windows")
    .with(CoreP.Device_Status_OSType.CHROMEOS, () => "ChromeOS")
    .with(CoreP.Device_Status_OSType.OS_TYPE_UNKNOWN, () => "Unknown OS")
    .otherwise(() => "");
};

const enumLabel = (values: any, value: number | undefined) => {
  if (value === undefined) return "Unknown";
  return String(values[value] ?? "Unknown")
    .replace(/^(RISK_LEVEL|SIGNAL_STATE|STATE|ACCEPTANCE_METHOD)_/, "")
    .toLowerCase()
    .replaceAll("_", " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
};

export const ItemInfo = (props: { item: CoreP.Device }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="OS">
        <Label>
          <span className="flex items-center">
            <span className="mr-1">{getOSIcon(item)}</span>
            <span>{getOSStr(item)}</span>
          </span>
        </Label>
      </InfoItem>
      {item.status!.id && <InfoItem title="ID">{item.status!.id}</InfoItem>}
      {item.status!.hostname && (
        <InfoItem title="Hostname">{item.status!.hostname}</InfoItem>
      )}
      {item.status!.serialNumber && (
        <InfoItem title="Serial Number">{item.status!.serialNumber}</InfoItem>
      )}
      <InfoItem title="State">
        <EditItemWrap
          mutation={mutationUpdate}
          showComponent={
            <span
              className={twMerge(
                match(item.spec!.state)
                  .with(CoreP.Device_Spec_State.REJECTED, () => "text-red-600")
                  .with(
                    CoreP.Device_Spec_State.PENDING,
                    () => "text-yellow-600",
                  )
                  .otherwise(() => undefined),
              )}
            >
              {match(item.spec!.state)
                .with(CoreP.Device_Spec_State.ACTIVE, () => "Active")
                .with(CoreP.Device_Spec_State.REJECTED, () => "Rejected")
                .with(CoreP.Device_Spec_State.PENDING, () => "Pending")
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Active",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE],
                },
                {
                  label: "Pending",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING],
                },
                {
                  label: "Rejected",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED],
                },
              ]}
              value={CoreP.Device_Spec_State[item.spec!.state]}
              onChange={(v) => {
                if (!v) {
                  return;
                }
                const next = CoreP.Device.clone(item);
                next.spec!.state = CoreP.Device_Spec_State[v as "ACTIVE"];
                mutationUpdate.mutate(next);
              }}
            />
          }
        />
      </InfoItem>
      {item.status!.macAddresses && item.status!.macAddresses.length > 0 && (
        <InfoItem title="MAC Addresses">
          {item.status!.macAddresses.map((x) => (
            <Label key={x}>{x}</Label>
          ))}
        </InfoItem>
      )}
    </>
  );
};

export const AccessLog = (props: { item: CoreP.Device }) => {
  return <AccessLogViewer deviceRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreP.Device }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

const Signal = CoreP.Device_Status_Posture_SignalState;
const Risk = CoreP.Device_Status_Posture_RiskLevel;
const Binding_State = CoreP.Device_Status_Binding_State;
const Binding_Validity = CoreP.Device_Status_Binding_Validity;

const riskTone = (risk?: CoreP.Device_Status_Posture_RiskLevel): OverviewTone =>
  risk === Risk.CRITICAL || risk === Risk.HIGH
    ? "danger"
    : risk === Risk.MEDIUM
      ? "warning"
      : risk === Risk.LOW
        ? "success"
        : "neutral";

const POSTURE_CHECKS: [
  keyof CoreP.Device_Status_Posture,
  string,
][] = [
  ["diskEncryption", "Disk encryption"],
  ["firewall", "Firewall"],
  ["screenLock", "Screen lock"],
  ["secureBoot", "Secure boot"],
  ["osUpToDate", "OS up to date"],
  ["threatFree", "No active threats"],
  ["compliant", "Compliant"],
  ["agentHealthy", "Agent healthy"],
  ["enrolled", "Enrolled"],
  ["notContained", "Not contained"],
  ["mobileIntegrity", "Mobile integrity"],
];

const getChecks = (posture?: CoreP.Device_Status_Posture) => {
  if (!posture) return [];
  return [
    ...POSTURE_CHECKS.map(([key, label]) => ({
      label,
      state: posture[key] as CoreP.Device_Status_Posture_SignalState,
    })),
    ...Object.entries(posture.signals).map(([label, state]) => ({
      label,
      state,
    })),
  ].filter((check) => check.state !== Signal.SIGNAL_STATE_UNKNOWN);
};

const CheckPill = (props: {
  label: string;
  state: CoreP.Device_Status_Posture_SignalState;
}) => (
  <TonePill
    tone={
      props.state === Signal.PASS
        ? "success"
        : props.state === Signal.FAIL
          ? "danger"
          : "neutral"
    }
    icon={
      props.state === Signal.PASS ? Check : props.state === Signal.FAIL ? X : Minus
    }
  >
    {props.label}
  </TonePill>
);

const bindingProblem = (binding?: CoreP.Device_Status_Binding) => {
  if (!binding?.uid) return undefined;
  if (binding.state === Binding_State.AMBIGUOUS)
    return "The Device matches more than one managed device.";
  if (binding.state === Binding_State.CONFLICT)
    return "The managed device is already bound to another Device.";
  if (binding.validity === Binding_Validity.SUSPENDED)
    return "The device manager suspended the binding.";
  if (binding.validity === Binding_Validity.LOST)
    return "The managed device is no longer reported by the device manager.";
  return undefined;
};

const ResetBinding = (props: { item: CoreP.Device }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const binding = props.item.status?.binding;
  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientEnterprise().resetDeviceBinding({
          deviceRef: getResourceRef(props.item),
          bindingUID: binding?.uid ?? "",
        })
      ).response,
    onSuccess: () => {
      close();
      invalidateResource(props.item);
      invalidateResourceList(props.item);
      notifications.show({ color: "green", message: "Device binding reset" });
    },
    onError,
  });

  if (!binding?.uid) return null;

  return (
    <>
      <ResourceActionButton icon={Unlink} onClick={open}>
        Reset binding
      </ResourceActionButton>
      <ActionConfirmModal
        opened={opened}
        onClose={close}
        icon={Unlink}
        title="Reset device binding"
        subtitle={props.item.metadata?.name}
        confirmLabel="Reset binding"
        loading={mutation.isPending}
        onConfirm={() => mutation.mutate()}
      >
        <p>
          The Cluster clears the binding
          {binding.ownerRef?.name ? ` to ${binding.ownerRef.name}` : ""} and the
          Device posture, then selects a device manager again using the
          configured order.
        </p>
        <p className="text-xs text-slate-500">
          Policies that depend on the posture may deny this Device until it is
          verified again.
        </p>
      </ActionConfirmModal>
    </>
  );
};

const DeviceHero = (props: { item: CoreP.Device }) => {
  const { item } = props;
  const now = useNow();
  const status = item.status;
  const posture = status?.posture;
  const checks = getChecks(posture);
  const failing = checks.filter((check) => check.state === Signal.FAIL);
  const risk = riskTone(posture?.riskLevel);
  const postureExpiresAt = toDate(posture?.expiresAt);
  const postureExpired =
    !!postureExpiresAt && postureExpiresAt.getTime() <= now;
  const problem = bindingProblem(status?.binding);
  const state = item.spec?.state;
  const tone: OverviewTone =
    state === CoreP.Device_Spec_State.REJECTED || status?.isLocked || risk === "danger"
      ? "danger"
      : state === CoreP.Device_Spec_State.PENDING ||
          risk === "warning" ||
          failing.length > 0 ||
          !!problem ||
          postureExpired
        ? "warning"
        : "success";
  const OSIcon = getOSIcon(item);

  return (
    <OverviewHero
      tone={tone}
      icon={
        OSIcon ? (
          <span className="text-lg">{OSIcon}</span>
        ) : (
          <Laptop size={18} strokeWidth={2.2} />
        )
      }
      title={status?.hostname || item.metadata?.name}
      detail={[
        getOSStr(item),
        status?.userRef?.name && `owned by ${status.userRef.name}`,
        status?.serialNumber && `serial ${status.serialNumber}`,
      ]
        .filter(Boolean)
        .join(" · ")}
      meta={
        checks.length > 0 && (
          <>
            <TonePill tone={failing.length ? "warning" : "success"} icon={ShieldCheck}>
              {checks.length - failing.length} of {checks.length} checks passing
            </TonePill>
            {failing.map((check) => (
              <CheckPill key={check.label} {...check} />
            ))}
          </>
        )
      }
      aside={
        posture && (
          <HeroStat
            label="Risk"
            tone={risk}
            value={enumLabel(Risk, posture.riskLevel)}
          />
        )
      }
    >
      {status?.isLocked && (
        <OverviewNotice tone="danger" icon={Lock} title="Locked by the Cluster.">
          Sessions on this Device cannot access any Service.
        </OverviewNotice>
      )}
      {state === CoreP.Device_Spec_State.PENDING && (
        <OverviewNotice tone="warning" icon={Clock3} title="Pending approval.">
          Set the Device to Active to let its Sessions access Services.
        </OverviewNotice>
      )}
      {state === CoreP.Device_Spec_State.REJECTED && (
        <OverviewNotice tone="danger" icon={Ban} title="Rejected.">
          Sessions on this Device cannot access any Service.
        </OverviewNotice>
      )}
      {problem && (
        <OverviewNotice tone="warning" icon={ShieldAlert} title="Binding needs attention.">
          {status?.binding?.message || problem}
        </OverviewNotice>
      )}
      {postureExpired && (
        <OverviewNotice tone="warning" icon={Clock3} title="Posture expired.">
          The device manager has not refreshed the posture of this Device.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: { item: CoreP.Device }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const posture = status?.posture;
  const binding = status?.binding;
  const checks = getChecks(posture);
  const probe = status?.probeAttempt;

  return {
    header: <DeviceHero item={item} />,
    actions: <ResetBinding item={item} />,
    groupOrder: [
      { title: "Posture", icon: ShieldCheck },
      { title: "Device manager", icon: Link2 },
      { title: "Hardware", icon: Cpu },
      { title: "Probe", icon: Radar },
      { title: "Authorization", icon: ShieldCheck },
    ],
    items: [
      {
        label: "User",
        primary: true,
        icon: UserRound,
        value: <RefLabel itemRef={status?.userRef} />,
      },
      {
        label: "Operating system",
        primary: true,
        icon: Laptop,
        value: (
          <span className="inline-flex items-center gap-1.5 font-semibold">
            {getOSIcon(item)}
            {getOSStr(item)}
          </span>
        ),
      },
      {
        label: "Risk",
        primary: true,
        icon: Gauge,
        value: posture ? (
          <TonePill tone={riskTone(posture.riskLevel)}>
            {enumLabel(Risk, posture.riskLevel)}
          </TonePill>
        ) : (
          <MutedValue>No posture reported</MutedValue>
        ),
      },
      {
        label: "Device manager",
        primary: true,
        icon: Link2,
        value: binding?.ownerRef?.name ? (
          <span className="font-semibold">{binding.ownerRef.name}</span>
        ) : (
          <MutedValue>Not bound</MutedValue>
        ),
      },

      ...(posture
        ? [
            {
              label: "Checks",
              group: "Posture",
              span: "full" as const,
              value:
                checks.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {checks.map((check) => (
                      <CheckPill key={check.label} {...check} />
                    ))}
                  </div>
                ) : (
                  <MutedValue>No checks reported</MutedValue>
                ),
            },
            ...(posture.lastSyncAt
              ? [
                  {
                    label: "Last synced",
                    group: "Posture",
                    value: <TimeAgo rfc3339={posture.lastSyncAt} />,
                  },
                ]
              : []),
            ...(posture.lastSeenAt
              ? [
                  {
                    label: "Last seen",
                    group: "Posture",
                    value: <TimeAgo rfc3339={posture.lastSeenAt} />,
                  },
                ]
              : []),
            ...(posture.expiresAt
              ? [
                  {
                    label: "Expires",
                    group: "Posture",
                    value: <TimeAgo rfc3339={posture.expiresAt} />,
                  },
                ]
              : []),
          ]
        : [
            {
              label: "Checks",
              group: "Posture",
              value: <MutedValue>No posture reported</MutedValue>,
            },
          ]),

      ...(binding?.uid
        ? [
            {
              label: "Manager",
              group: "Device manager",
              value: binding.ownerRef?.name ? (
                <span className="font-semibold">{binding.ownerRef.name}</span>
              ) : (
                <MutedValue>Unknown</MutedValue>
              ),
            },
            ...(binding.externalID
              ? [
                  {
                    label: "External ID",
                    group: "Device manager",
                    value: <CopyText value={binding.externalID} />,
                  },
                ]
              : []),
            {
              label: "Binding",
              group: "Device manager",
              value: (
                <div className="flex flex-wrap gap-1">
                  <TonePill
                    tone={
                      binding.state === Binding_State.ACCEPTED
                        ? "success"
                        : "warning"
                    }
                  >
                    {enumLabel(Binding_State, binding.state)}
                  </TonePill>
                  <TonePill
                    tone={
                      binding.validity === Binding_Validity.VALID
                        ? "success"
                        : binding.validity === Binding_Validity.VALIDITY_UNKNOWN
                          ? "neutral"
                          : "danger"
                    }
                  >
                    {enumLabel(Binding_Validity, binding.validity)}
                  </TonePill>
                </div>
              ),
            },
            ...(binding.acceptedAt
              ? [
                  {
                    label: "Accepted",
                    group: "Device manager",
                    value: <TimeAgo rfc3339={binding.acceptedAt} />,
                  },
                ]
              : []),
            ...(binding.lastVerifiedAt
              ? [
                  {
                    label: "Last verified",
                    group: "Device manager",
                    value: <TimeAgo rfc3339={binding.lastVerifiedAt} />,
                  },
                ]
              : []),
            ...(binding.nextVerificationAt
              ? [
                  {
                    label: "Next verification",
                    group: "Device manager",
                    value: <TimeAgo rfc3339={binding.nextVerificationAt} />,
                  },
                ]
              : []),
            ...(binding.reason || binding.message
              ? [
                  {
                    label: "Reason",
                    group: "Device manager",
                    span: "full" as const,
                    value: [binding.reason, binding.message]
                      .filter(Boolean)
                      .join(" · "),
                  },
                ]
              : []),
          ]
        : [
            {
              label: "Manager",
              group: "Device manager",
              value: <MutedValue>The Device is not bound to a device manager.</MutedValue>,
            },
          ]),

      ...(status?.hostname
        ? [
            {
              label: "Hostname",
              group: "Hardware",
              value: <CopyText value={status.hostname} />,
            },
          ]
        : []),
      ...(status?.id
        ? [
            {
              label: "Device ID",
              group: "Hardware",
              value: <CopyText value={status.id} />,
            },
          ]
        : []),
      ...(status?.serialNumber
        ? [
            {
              label: "Serial number",
              group: "Hardware",
              value: <CopyText value={status.serialNumber} />,
            },
          ]
        : []),
      ...(status?.macAddresses.length
        ? [
            {
              label: "MAC addresses",
              group: "Hardware",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {status.macAddresses.map((x) => (
                    <ResourceListLabel key={x} label="MAC">
                      <CopyText value={x} />
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(probe
        ? [
            {
              label: "State",
              group: "Probe",
              value: (
                <TonePill
                  tone={
                    probe.state ===
                    CoreP.Device_Status_ProbeAttempt_State.PROCESSED
                      ? "success"
                      : "info"
                  }
                  icon={Activity}
                >
                  {enumLabel(CoreP.Device_Status_ProbeAttempt_State, probe.state)}
                </TonePill>
              ),
            },
            {
              label: "Results",
              group: "Probe",
              value: (
                <span className="font-semibold tabular-nums">
                  {probe.results.length} of {probe.probes.length} probes
                </span>
              ),
            },
            ...(probe.startedAt
              ? [
                  {
                    label: "Started",
                    group: "Probe",
                    value: <TimeAgo rfc3339={probe.startedAt} />,
                  },
                ]
              : []),
            ...(probe.submittedAt
              ? [
                  {
                    label: "Submitted",
                    group: "Probe",
                    value: <TimeAgo rfc3339={probe.submittedAt} />,
                  },
                ]
              : []),
            ...(probe.sessionRef?.name || probe.sessionRef?.uid
              ? [
                  {
                    label: "Session",
                    group: "Probe",
                    value: <RefLabel itemRef={probe.sessionRef} />,
                  },
                ]
              : []),
          ]
        : []),

      ...authorizationItems(item.spec?.authorization),
    ],
  };
};
