import * as CoreP from "@/apis/corev1/corev1";
import InfoItem from "@/components/InfoItem";
import { useUpdateResource } from "@/pages/utils/resource";
import { Select } from "@mantine/core";
import { match } from "ts-pattern";

import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
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
                item.spec!.state =
                  CoreP.Authenticator_Spec_State[v as "ACTIVE"];
                mutationUpdate.mutate(item);
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
  const mutationUpdate = useUpdateResource();
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

export const MainInfo = (props: {
  item: CoreP.Authenticator;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const fido =
    item.status?.info?.type.oneofKind === "fido"
      ? item.status.info.type.fido
      : null;

  return {
    items: [
      {
        label: "User",
        value: <ResourceListLabel itemRef={item.status!.userRef} />,
      },
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
                  item.spec!.state =
                    CoreP.Authenticator_Spec_State[v as "ACTIVE"];
                  mutationUpdate.mutate(item);
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

      ...(fido?.aaguid
        ? [
            {
              label: "AAGUID",
              value: (
                <span className="text-sm font-mono text-slate-700">
                  {fido.aaguid}
                </span>
              ),
            },
          ]
        : []),

      ...(fido
        ? [
            {
              label: "FIDO flags",
              value: (
                <div className="flex flex-wrap gap-1">
                  {fido.isPasskey && <Label>Passkey</Label>}
                  {fido.isHardware && <Label>Hardware</Label>}
                  {fido.isAttestationVerified && (
                    <Label>
                      <span className="text-emerald-400">
                        Attestation verified
                      </span>
                    </Label>
                  )}
                </div>
              ),
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
                  {item.status!.successfulAuthentications > 0 && (
                    <span className="text-slate-400 font-medium ml-1.5">
                      ({item.status!.successfulAuthentications} successful)
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),
    ],
  };
};
