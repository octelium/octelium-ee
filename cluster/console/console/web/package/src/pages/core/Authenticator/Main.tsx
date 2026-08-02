import * as CoreP from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import { useUpdateResource } from "@/pages/utils/resource";
import { Select } from "@mantine/core";
import { match } from "ts-pattern";

import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "tailwind-merge";

export const getType = (item: CoreP.Authenticator) => {
  return match(item.status!.type)
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

const AuthenticatorDetails = (props: { item: CoreP.Authenticator }) => {
  const info = props.item.status?.info?.type;
  if (!info?.oneofKind) return null;

  if (info.oneofKind === "fido") {
    const fido = info.fido;
    return (
      <div className="flex flex-wrap gap-1">
        {fido.aaguid && (
          <ResourceListLabel label="AAGUID">
            <CopyText value={fido.aaguid} />
          </ResourceListLabel>
        )}
        <ResourceListLabel label="Authenticator class">
          {fido.type === CoreP.Authenticator_Status_Info_FIDO_Type.PLATFORM
            ? "Platform"
            : fido.type === CoreP.Authenticator_Status_Info_FIDO_Type.ROAMING
              ? "Roaming"
              : "Unknown"}
        </ResourceListLabel>
        {fido.isPasskey && <ResourceListLabel>Passkey</ResourceListLabel>}
        {fido.isHardware && <ResourceListLabel>Hardware</ResourceListLabel>}
        {fido.isSoftware && <ResourceListLabel>Software</ResourceListLabel>}
        {fido.backupEligible && (
          <ResourceListLabel>Backup eligible</ResourceListLabel>
        )}
        {fido.isAttestationVerified && (
          <ResourceListLabel>
            <span className="text-emerald-600">Attestation verified</span>
          </ResourceListLabel>
        )}
        <ResourceListLabel label="Signature counter">
          {fido.signCount.toLocaleString()}
        </ResourceListLabel>
      </div>
    );
  }

  if (info.oneofKind === "totp") {
    const totp = info.totp;
    return (
      <div className="flex flex-wrap gap-1">
        <ResourceListLabel label="Algorithm">
          {CoreP.Authenticator_Status_Info_TOTP_Algorithm[
            totp.algorithm
          ]?.replace("ALGORITHM_", "") || "Unknown"}
        </ResourceListLabel>
        <ResourceListLabel label="Digits">{totp.digits}</ResourceListLabel>
        <ResourceListLabel label="Period">
          {totp.periodSeconds} seconds
        </ResourceListLabel>
        {totp.lastAcceptedAt && (
          <ResourceListLabel label="Last accepted">
            <TimeAgo rfc3339={totp.lastAcceptedAt} />
          </ResourceListLabel>
        )}
      </div>
    );
  }

  return (
    <ResourceListLabel>TPM attestation material configured</ResourceListLabel>
  );
};

export const MainInfo = (props: {
  item: CoreP.Authenticator;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "User",
        value: <ResourceListLabel itemRef={item.status!.userRef} />,
      },
      ...(item.status?.deviceRef?.name || item.status?.deviceRef?.uid
        ? [
            {
              label: "Device",
              value: <ResourceListLabel itemRef={item.status.deviceRef} />,
            },
          ]
        : []),
      {
        label: "Type",
        value: (
          <span className="text-sm font-semibold text-slate-700">
            {getType(item)}
          </span>
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
                      CoreP.Authenticator_Spec_State.ACTIVE,
                      () => "text-emerald-600",
                    )
                    .with(
                      CoreP.Authenticator_Spec_State.REJECTED,
                      () => "text-red-500",
                    )
                    .with(
                      CoreP.Authenticator_Spec_State.PENDING,
                      () => "text-amber-500",
                    )
                    .otherwise(() => "text-slate-600"),
                )}
              >
                {match(item.spec!.state)
                  .with(CoreP.Authenticator_Spec_State.ACTIVE, () => "Active")
                  .with(
                    CoreP.Authenticator_Spec_State.REJECTED,
                    () => "Rejected",
                  )
                  .with(CoreP.Authenticator_Spec_State.PENDING, () => "Pending")
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
                  if (!v) return;
                  const next = CoreP.Authenticator.clone(item);
                  next.spec!.state =
                    CoreP.Authenticator_Spec_State[v as "ACTIVE"];
                  mutationUpdate.mutate(next);
                }}
              />
            }
          />
        ),
      },

      {
        label: "Registered",
        value: (
          <span
            className={twMerge(
              "text-sm font-semibold",
              item.status!.isRegistered ? "text-emerald-600" : "text-red-500",
            )}
          >
            {item.status!.isRegistered ? "Yes" : "No"}
          </span>
        ),
      },

      ...(item.spec?.displayName
        ? [
            {
              label: "Display name",
              value: item.spec.displayName,
            },
          ]
        : []),

      ...(item.status?.description
        ? [
            {
              label: "Description",
              value: item.status.description,
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status!.totalAuthenticationAttempts > 0
        ? [
            {
              label: "Auth attempts",
              value: (
                <span className="text-sm font-semibold text-slate-700 tabular-nums">
                  {item.status!.totalAuthenticationAttempts}
                  <span className="ml-1.5 font-medium text-emerald-600">
                    {item.status!.successfulAuthentications} successful
                  </span>
                  <span className="ml-1.5 font-medium text-red-500">
                    {item.status!.failedAuthentications} failed
                  </span>
                </span>
              ),
            },
          ]
        : []),

      {
        label: "Authenticator details",
        value: <AuthenticatorDetails item={item} />,
        span: "full",
      },

      ...(item.status?.authenticationAttempt
        ? [
            {
              label: "Current attempt",
              value: (
                <div className="flex flex-wrap gap-1">
                  {(item.status.authenticationAttempt.sessionRef?.name ||
                    item.status.authenticationAttempt.sessionRef?.uid) && (
                    <ResourceListLabel
                      itemRef={item.status.authenticationAttempt.sessionRef}
                    />
                  )}
                  {item.status.authenticationAttempt.createdAt && (
                    <ResourceListLabel label="Started">
                      <TimeAgo
                        rfc3339={item.status.authenticationAttempt.createdAt}
                      />
                    </ResourceListLabel>
                  )}
                  {item.status.authenticationAttempt.completedAt && (
                    <ResourceListLabel label="Completed">
                      <TimeAgo
                        rfc3339={item.status.authenticationAttempt.completedAt}
                      />
                    </ResourceListLabel>
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
