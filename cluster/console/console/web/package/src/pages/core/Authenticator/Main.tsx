import * as CoreP from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import { useUpdateResource } from "@/pages/utils/resource";
import { Select } from "@mantine/core";
import { match } from "ts-pattern";

import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "@/lib/utils";
import Label from "@/components/Label";
import {
  HeroStat,
  HistoryList,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  TonePill,
} from "@/components/ResourceLayout/overview";
import { RefLabel } from "@/components/ResourceLayout/overviewItems";
import {
  Activity,
  BadgeCheck,
  Ban,
  Clock3,
  Cpu,
  Fingerprint,
  KeyRound,
  Laptop,
  Smartphone,
  UserRound,
} from "lucide-react";

export const getType = (item: CoreP.Authenticator) => {
  return match(item.status?.type)
    .with(CoreP.Authenticator_Status_Type.TPM, () => "TPM")
    .with(CoreP.Authenticator_Status_Type.FIDO, () => "FIDO")
    .with(CoreP.Authenticator_Status_Type.TOTP, () => "TOTP")
    .otherwise(() => "");
};

export const ItemInfo = (props: { item: CoreP.Authenticator }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">{getType(item)}</InfoItem>

      {item.status!.info && item.status!.info.type.oneofKind === `fido` && (
        <>
          <InfoItem title="AAGUID">
            {item.status!.info.type.fido.aaguid}
          </InfoItem>
          {item.status!.info.type.fido.isAttestationVerified && (
            <InfoItem title="Attestation Verified">
              <span className={twMerge(`text-green-600`)}>{`Yes`}</span>
            </InfoItem>
          )}
          {item.status!.info.type.fido.isHardware && (
            <InfoItem title="Hardware">{`Yes`}</InfoItem>
          )}
          {item.status!.info.type.fido.isPasskey && (
            <InfoItem title="Passkey">{`Yes`}</InfoItem>
          )}
        </>
      )}
      <InfoItem title="Registered">
        <span
          className={twMerge(
            !item.status!.isRegistered ? `text-red-500` : undefined,
          )}
        >
          {!item.status!.isRegistered ? `No` : `Yes`}
        </span>
      </InfoItem>
      {item.spec!.displayName.length > 0 && (
        <InfoItem title="Display Name">{item.spec!.displayName}</InfoItem>
      )}
      {item.status!.description.length > 0 && (
        <InfoItem title="Description">{item.status!.description}</InfoItem>
      )}

      {item.status!.totalAuthenticationAttempts > 0 && (
        <InfoItem title="Total Authentication Attempts">
          {item.status!.totalAuthenticationAttempts}
        </InfoItem>
      )}

      {item.status!.successfulAuthentications > 0 && (
        <InfoItem title="Successful Authentications">
          {item.status!.successfulAuthentications}
        </InfoItem>
      )}

      <InfoItem title="State">
        <EditItemWrap
          mutation={mutationUpdate}
          showComponent={
            <span
              className={twMerge(
                match(item.spec!.state)
                  .with(
                    CoreP.Authenticator_Spec_State.REJECTED,
                    () => "text-red-600",
                  )
                  .with(
                    CoreP.Authenticator_Spec_State.PENDING,
                    () => "text-yellow-600",
                  )
                  .otherwise(() => undefined),
              )}
            >
              {match(item.spec!.state)
                .with(CoreP.Authenticator_Spec_State.ACTIVE, () => "Active")
                .with(CoreP.Authenticator_Spec_State.REJECTED, () => "Rejected")
                .with(CoreP.Authenticator_Spec_State.PENDING, () => "Pending")
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Active",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.ACTIVE
                    ],
                },
                {
                  label: "Pending",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.PENDING
                    ],
                },
                {
                  label: "Rejected",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.REJECTED
                    ],
                },
              ]}
              value={CoreP.Authenticator_Spec_State[item.spec!.state]}
              onChange={(v) => {
                if (!v) {
                  return;
                }
                const next = CoreP.Authenticator.clone(item);
                next.spec!.state =
                  CoreP.Authenticator_Spec_State[v as "ACTIVE"];
                mutationUpdate.mutate(next);
              }}
            />
          }
        />
      </InfoItem>
    </>
  );
};

export default (props: { item: CoreP.Authenticator }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

const FIDOType = CoreP.Authenticator_Status_Info_FIDO_Type;

const typeIcon = (item: CoreP.Authenticator) =>
  match(item.status?.type)
    .with(CoreP.Authenticator_Status_Type.FIDO, () => KeyRound)
    .with(CoreP.Authenticator_Status_Type.TOTP, () => Smartphone)
    .with(CoreP.Authenticator_Status_Type.TPM, () => Cpu)
    .otherwise(() => Fingerprint);

const typeDescription = (item: CoreP.Authenticator) => {
  const info = item.status?.info?.type;
  if (info?.oneofKind === "fido") {
    if (info.fido.isPasskey) return "Passkey";
    if (info.fido.type === FIDOType.ROAMING) return "Roaming security key";
    if (info.fido.type === FIDOType.PLATFORM) return "Platform authenticator";
    return "FIDO authenticator";
  }
  if (info?.oneofKind === "totp") return "Authenticator app (TOTP)";
  if (info?.oneofKind === "tpm") return "Trusted Platform Module";
  return `${getType(item) || "Unknown"} authenticator`;
};

const TypeGlyph = (props: { item: CoreP.Authenticator }) =>
  match(props.item.status?.type)
    .with(CoreP.Authenticator_Status_Type.FIDO, () => (
      <KeyRound size={18} strokeWidth={2.2} />
    ))
    .with(CoreP.Authenticator_Status_Type.TOTP, () => (
      <Smartphone size={18} strokeWidth={2.2} />
    ))
    .with(CoreP.Authenticator_Status_Type.TPM, () => (
      <Cpu size={18} strokeWidth={2.2} />
    ))
    .otherwise(() => <Fingerprint size={18} strokeWidth={2.2} />);

const AuthenticatorHero = (props: { item: CoreP.Authenticator }) => {
  const { item } = props;
  const status = item.status!;
  const fido =
    status.info?.type.oneofKind === "fido" ? status.info.type.fido : undefined;
  const total = status.totalAuthenticationAttempts;
  const rate = total > 0 ? (status.successfulAuthentications / total) * 100 : undefined;
  const state = item.spec?.state;
  const tone =
    state === CoreP.Authenticator_Spec_State.REJECTED
      ? "danger"
      : !status.isRegistered || state === CoreP.Authenticator_Spec_State.PENDING
        ? "warning"
        : "success";

  return (
    <OverviewHero
      tone={tone}
      icon={<TypeGlyph item={item} />}
      title={item.spec?.displayName || status.description || typeDescription(item)}
      detail={[
        typeDescription(item),
        status.userRef?.name && `registered by ${status.userRef.name}`,
      ]
        .filter(Boolean)
        .join(" · ")}
      meta={
        fido && (
          <>
            {fido.isHardware && <TonePill icon={Cpu}>Hardware</TonePill>}
            {fido.isSoftware && <TonePill>Software</TonePill>}
            {fido.backupEligible && <TonePill>Backup eligible</TonePill>}
            {fido.isAttestationVerified && (
              <TonePill tone="success" icon={BadgeCheck}>
                Attestation verified
              </TonePill>
            )}
          </>
        )
      }
      aside={
        rate !== undefined && (
          <HeroStat
            label="Success rate"
            tone={rate >= 90 ? "success" : rate >= 60 ? "warning" : "danger"}
            value={`${Math.round(rate)}%`}
          />
        )
      }
    >
      {!status.isRegistered && (
        <OverviewNotice tone="warning" icon={Clock3} title="Registration not completed.">
          The Authenticator cannot be used until its registration finishes.
        </OverviewNotice>
      )}
      {state === CoreP.Authenticator_Spec_State.PENDING && (
        <OverviewNotice tone="warning" icon={Clock3} title="Pending approval.">
          Set the Authenticator to Active to allow it to authenticate.
        </OverviewNotice>
      )}
      {state === CoreP.Authenticator_Spec_State.REJECTED && (
        <OverviewNotice tone="danger" icon={Ban} title="Rejected.">
          The Authenticator cannot be used to authenticate.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

const detailItems = (item: CoreP.Authenticator, group: string) => {
  const info = item.status?.info?.type;
  if (info?.oneofKind === "fido") {
    const fido = info.fido;
    return [
      {
        label: "Class",
        group,
        value:
          fido.type === FIDOType.PLATFORM
            ? "Platform"
            : fido.type === FIDOType.ROAMING
              ? "Roaming"
              : "Unknown",
      },
      {
        label: "Signature counter",
        group,
        value: (
          <span className="tabular-nums">{fido.signCount.toLocaleString()}</span>
        ),
      },
      ...(fido.aaguid
        ? [
            {
              label: "AAGUID",
              hint: "Identifies the authenticator model",
              group,
              span: "full" as const,
              value: <CopyText value={fido.aaguid} />,
            },
          ]
        : []),
    ];
  }
  if (info?.oneofKind === "totp") {
    const totp = info.totp;
    return [
      {
        label: "Algorithm",
        group,
        value:
          CoreP.Authenticator_Status_Info_TOTP_Algorithm[totp.algorithm]?.replace(
            "ALGORITHM_",
            "",
          ) || "Unknown",
      },
      { label: "Digits", group, value: totp.digits },
      { label: "Period", group, value: `${totp.periodSeconds} seconds` },
      ...(totp.lastAcceptedAt
        ? [
            {
              label: "Last accepted code",
              group,
              value: <TimeAgo rfc3339={totp.lastAcceptedAt} />,
            },
          ]
        : []),
    ];
  }
  if (info?.oneofKind === "tpm") {
    return [
      {
        label: "Attestation",
        group,
        value: "TPM attestation material configured",
      },
    ];
  }
  return [];
};

export const MainInfo = (props: {
  item: CoreP.Authenticator;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status!;
  const Icon = typeIcon(item);
  const attempts = [...status.lastAuthenticationAttempts].sort(
    (a, b) =>
      Number(b.createdAt?.seconds ?? 0) - Number(a.createdAt?.seconds ?? 0),
  );

  return {
    header: <AuthenticatorHero item={item} />,
    groupOrder: [
      { title: "Authenticator", icon: Icon },
      { title: "Activity", icon: Activity },
    ],
    items: [
      {
        label: "User",
        primary: true,
        icon: UserRound,
        value: <RefLabel itemRef={status.userRef} />,
      },
      {
        label: "Device",
        primary: true,
        icon: Laptop,
        value: status.deviceRef?.name || status.deviceRef?.uid ? (
          <RefLabel itemRef={status.deviceRef} />
        ) : (
          <MutedValue>Not bound to a Device</MutedValue>
        ),
      },
      {
        label: "Type",
        primary: true,
        icon: Icon,
        value: <Label>{getType(item) || "Unknown"}</Label>,
      },
      {
        label: "Authentications",
        primary: true,
        icon: Activity,
        value: (
          <span className="font-semibold tabular-nums">
            {status.successfulAuthentications.toLocaleString()}
            {status.failedAuthentications > 0 && (
              <span className="ml-1.5 text-xs font-normal text-red-600">
                {status.failedAuthentications.toLocaleString()} failed
              </span>
            )}
          </span>
        ),
      },

      {
        label: "Kind",
        group: "Authenticator",
        value: typeDescription(item),
      },
      {
        label: "Registered",
        group: "Authenticator",
        value: status.isRegistered ? (
          <TonePill tone="success" icon={BadgeCheck}>
            Yes
          </TonePill>
        ) : (
          <TonePill tone="warning">Not completed</TonePill>
        ),
      },
      ...detailItems(item, "Authenticator"),
      ...(status.description && item.spec?.displayName
        ? [
            {
              label: "Description",
              group: "Authenticator",
              span: "full" as const,
              value: status.description,
            },
          ]
        : []),

      {
        label: "Attempts",
        group: "Activity",
        value: (
          <span className="flex flex-wrap items-baseline gap-x-3">
            <span className="font-semibold tabular-nums">
              {status.totalAuthenticationAttempts.toLocaleString()}
            </span>
            <span className="text-xs text-emerald-700">
              {status.successfulAuthentications.toLocaleString()} successful
            </span>
            <span className="text-xs text-red-700">
              {status.failedAuthentications.toLocaleString()} failed
            </span>
          </span>
        ),
      },
      ...(status.authenticationAttempt
        ? [
            {
              label: "Current attempt",
              group: "Activity",
              value: (
                <span className="flex flex-wrap items-center gap-1.5">
                  <RefLabel itemRef={status.authenticationAttempt.sessionRef} />
                  {status.authenticationAttempt.createdAt && (
                    <span className="text-xs text-slate-500">
                      started{" "}
                      <TimeAgo rfc3339={status.authenticationAttempt.createdAt} />
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),
      ...(attempts.length > 0
        ? [
            {
              label: "Recent attempts",
              group: "Activity",
              layout: "stacked" as const,
              value: (
                <HistoryList
                  entries={attempts.slice(0, 5).map((attempt, index) => ({
                    key: `${attempt.createdAt?.seconds ?? 0}-${index}`,
                    tone: attempt.completedAt ? "success" : "neutral",
                    label: attempt.completedAt ? "Completed" : "Not completed",
                    meta: (
                      <>
                        {attempt.sessionRef?.name && (
                          <span>{attempt.sessionRef.name}</span>
                        )}
                        {attempt.createdAt && (
                          <TimeAgo rfc3339={attempt.createdAt} />
                        )}
                      </>
                    ),
                  }))}
                />
              ),
            },
          ]
        : []),
    ],
  };
};
