import * as CoreP from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
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
import { FaAndroid, FaApple, FaLinux, FaWindows } from "react-icons/fa";
import { Activity, Shield } from "lucide-react";
import { twMerge } from "tailwind-merge";

export const getOSIcon = (item: CoreP.Device) => {
  return match(item.status!.osType)
    .with(CoreP.Device_Status_OSType.LINUX, () => <FaLinux />)
    .with(CoreP.Device_Status_OSType.ANDROID, () => <FaAndroid />)
    .with(CoreP.Device_Status_OSType.IOS, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.MAC, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.WINDOWS, () => <FaWindows />)
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

export const MainInfo = (props: { item: CoreP.Device }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "User",
        value: <ResourceListLabel itemRef={item.status!.userRef} />,
      },
      {
        label: "OS",
        value: (
          <Label>
            <span className="flex items-center gap-1">
              {getOSIcon(item)}
              <span>{getOSStr(item)}</span>
            </span>
          </Label>
        ),
      },

      {
        label: "State",
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="state"
            showComponent={
              <span
                className={twMerge(
                  "text-sm font-semibold",
                  match(item.spec!.state)
                    .with(
                      CoreP.Device_Spec_State.ACTIVE,
                      () => "text-emerald-600",
                    )
                    .with(
                      CoreP.Device_Spec_State.REJECTED,
                      () => "text-red-500",
                    )
                    .with(
                      CoreP.Device_Spec_State.PENDING,
                      () => "text-amber-500",
                    )
                    .otherwise(() => "text-slate-600"),
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
                size="sm"
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
                  if (!v) return;
                  const next = CoreP.Device.clone(item);
                  next.spec!.state = CoreP.Device_Spec_State[v as "ACTIVE"];
                  mutationUpdate.mutate(next);
                }}
              />
            }
          />
        ),
      },

      ...(item.status!.hostname
        ? [
            {
              label: "Hostname",
              value: <CopyText value={item.status!.hostname} />,
            },
          ]
        : []),

      ...(item.status!.id
        ? [
            {
              label: "Device ID",
              value: <CopyText value={item.status!.id} />,
            },
          ]
        : []),

      ...(item.status!.serialNumber
        ? [
            {
              label: "Serial number",
              value: <CopyText value={item.status!.serialNumber} />,
            },
          ]
        : []),

      ...(item.status!.macAddresses?.length > 0
        ? [
            {
              label: "MAC addresses",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.status!.macAddresses.map((x) => (
                    <ResourceListLabel key={x} label="MAC">
                      <CopyText value={x} />
                    </ResourceListLabel>
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status?.isLocked
        ? [
            {
              label: "Security state",
              value: <span className="font-semibold text-red-600">Locked</span>,
            },
          ]
        : []),

      ...(item.spec?.authorization?.policies.length
        ? [
            {
              label: "Policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.policies.map((policy) => (
                    <ResourceListLabel
                      key={policy}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Policy",
                        name: policy,
                      })}
                    />
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.spec?.authorization?.inlinePolicies.length
        ? [
            {
              label: "Inline policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.inlinePolicies.map(
                    (policy, index) => (
                      <ResourceListLabel
                        key={`${policy.name}-${index}`}
                        label="Inline policy"
                      >
                        <Shield size={12} strokeWidth={2.5} />
                        {policy.name || `Inline policy ${index + 1}`}
                      </ResourceListLabel>
                    ),
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status?.posture
        ? [
            {
              label: "Device posture",
              value: (
                <div className="flex flex-wrap gap-1">
                  <ResourceListLabel label="Risk">
                    <span
                      className={twMerge(
                        item.status.posture.riskLevel ===
                          CoreP.Device_Status_Posture_RiskLevel.CRITICAL ||
                          item.status.posture.riskLevel ===
                            CoreP.Device_Status_Posture_RiskLevel.HIGH
                          ? "text-red-600"
                          : item.status.posture.riskLevel ===
                              CoreP.Device_Status_Posture_RiskLevel.MEDIUM
                            ? "text-amber-600"
                            : "text-emerald-600",
                      )}
                    >
                      {enumLabel(
                        CoreP.Device_Status_Posture_RiskLevel,
                        item.status.posture.riskLevel,
                      )}
                    </span>
                  </ResourceListLabel>
                  {[
                    ["Disk encryption", item.status.posture.diskEncryption],
                    ["Compliance", item.status.posture.compliant],
                    ["Threat status", item.status.posture.threatFree],
                  ].map(([label, state]) => (
                    <ResourceListLabel key={String(label)} label={String(label)}>
                      <span
                        className={twMerge(
                          state === CoreP.Device_Status_Posture_SignalState.FAIL
                            ? "text-red-600"
                            : state ===
                                CoreP.Device_Status_Posture_SignalState.PASS
                              ? "text-emerald-600"
                              : "text-slate-500",
                        )}
                      >
                        {enumLabel(
                          CoreP.Device_Status_Posture_SignalState,
                          state as number,
                        )}
                      </span>
                    </ResourceListLabel>
                  ))}
                  {Object.entries(item.status.posture.signals).map(
                    ([name, state]) => (
                      <ResourceListLabel key={name} label={name}>
                        <span
                          className={twMerge(
                            state ===
                              CoreP.Device_Status_Posture_SignalState.FAIL
                              ? "text-red-600"
                              : state ===
                                  CoreP.Device_Status_Posture_SignalState.PASS
                                ? "text-emerald-600"
                                : "text-slate-500",
                          )}
                        >
                          {enumLabel(
                            CoreP.Device_Status_Posture_SignalState,
                            state,
                          )}
                        </span>
                      </ResourceListLabel>
                    ),
                  )}
                  {item.status.posture.lastSyncAt && (
                    <ResourceListLabel label="Last sync">
                      <TimeAgo rfc3339={item.status.posture.lastSyncAt} />
                    </ResourceListLabel>
                  )}
                  {item.status.posture.lastSeenAt && (
                    <ResourceListLabel label="Last seen">
                      <TimeAgo rfc3339={item.status.posture.lastSeenAt} />
                    </ResourceListLabel>
                  )}
                  {item.status.posture.expiresAt && (
                    <ResourceListLabel label="Expires">
                      <TimeAgo rfc3339={item.status.posture.expiresAt} />
                    </ResourceListLabel>
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status?.binding
        ? [
            {
              label: "Device Manager binding",
              value: (
                <div className="flex flex-wrap gap-1">
                  {(item.status.binding.ownerRef?.name ||
                    item.status.binding.ownerRef?.uid) && (
                    <ResourceListLabel
                      itemRef={item.status.binding.ownerRef}
                    />
                  )}
                  <ResourceListLabel label="State">
                    {enumLabel(
                      CoreP.Device_Status_Binding_State,
                      item.status.binding.state,
                    )}
                  </ResourceListLabel>
                  <ResourceListLabel label="Acceptance">
                    {enumLabel(
                      CoreP.Device_Status_Binding_AcceptanceMethod,
                      item.status.binding.acceptanceMethod,
                    )}
                  </ResourceListLabel>
                  {item.status.binding.externalID && (
                    <ResourceListLabel label="External ID">
                      <CopyText value={item.status.binding.externalID} />
                    </ResourceListLabel>
                  )}
                  {item.status.binding.acceptedAt && (
                    <ResourceListLabel label="Accepted">
                      <TimeAgo rfc3339={item.status.binding.acceptedAt} />
                    </ResourceListLabel>
                  )}
                  {item.status.binding.lastVerifiedAt && (
                    <ResourceListLabel label="Last verified">
                      <TimeAgo rfc3339={item.status.binding.lastVerifiedAt} />
                    </ResourceListLabel>
                  )}
                  {item.status.binding.expiresAt && (
                    <ResourceListLabel label="Binding expires">
                      <TimeAgo rfc3339={item.status.binding.expiresAt} />
                    </ResourceListLabel>
                  )}
                  {item.status.binding.verificationFailures > 0 && (
                    <ResourceListLabel label="Verification failures">
                      <span className="text-red-600">
                        {item.status.binding.verificationFailures}
                      </span>
                    </ResourceListLabel>
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status?.probeAttempt
        ? [
            {
              label: "Probe attempt",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.status.probeAttempt.uid && (
                    <ResourceListLabel label="Attempt ID">
                      <CopyText value={item.status.probeAttempt.uid} />
                    </ResourceListLabel>
                  )}
                  {item.status.probeAttempt.startedAt && (
                    <ResourceListLabel label="Started">
                      <TimeAgo rfc3339={item.status.probeAttempt.startedAt} />
                    </ResourceListLabel>
                  )}
                  <ResourceListLabel label="Probes">
                    <Activity size={12} strokeWidth={2.5} />
                    {item.status.probeAttempt.probes.length}
                  </ResourceListLabel>
                  <ResourceListLabel label="Results">
                    {item.status.probeAttempt.results.length}
                  </ResourceListLabel>
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
