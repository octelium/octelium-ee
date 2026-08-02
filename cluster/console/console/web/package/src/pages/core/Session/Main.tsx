import * as CoreC from "@/apis/corev1/corev1";
import * as CoreP from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import TimestampPicker from "@/components/TimestampPicker";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getResourceRef } from "@/utils/pb";
import { Select } from "@mantine/core";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getType } from "./List";
import { SessionOperationalDetails } from "./Info";

export const ItemInfo = (props: { item: CoreC.Session }) => {
  let { item } = props;
  const mutationUpdate = useUpdateResource();
  const updateItem = (update: (item: CoreC.Session) => void) => {
    const clone = CoreC.Session.clone(item);
    update(clone);
    mutationUpdate.mutate(clone);
  };

  return (
    <>
      <InfoItem title="Type">
        <Label>{getType(item)}</Label>
      </InfoItem>
      {item.status!.isConnected && (
        <InfoItem title="Connected">
          <div className="flex items-center">
            <div className="flex relative w-[10px] h-[10px]">
              <div className="bg-green-500 rounded-full w-[10px] h-[10px] animate-ping absolute"></div>
              <div className="bg-green-500 rounded-full w-[10px] h-[10px]"></div>
            </div>
            <span className="ml-1">Yes</span>
          </div>
        </InfoItem>
      )}

      {item.spec?.expiresAt && (
        <InfoItem title="Expiration">
          <div className="w-full flex items-center">
            <EditItemWrap
              showComponent={
                <span className="flex-1 mr-2">
                  <TimeAgo rfc3339={item.spec.expiresAt} />
                </span>
              }
              editComponent={
                <TimestampPicker
                  label="Expires at"
                  description="Set the expiration time for the Session"
                  isFuture
                  value={item.spec.expiresAt}
                  onChange={(v) => {
                    updateItem((clone) => {
                      clone.spec!.expiresAt = v;
                    });
                  }}
                />
              }
            />
          </div>
        </InfoItem>
      )}
      {item.status?.authentication &&
        item.status.authentication.setAt && (
          <InfoItem title="Last Authentication">
            <TimeAgo rfc3339={item.status.authentication.setAt} />
          </InfoItem>
        )}

      {item.status?.authentication &&
        item.status.authentication.info &&
        item.status.authentication.info.downstream && (
          <>
            {item.status.authentication.info.downstream.userAgent && (
              <InfoItem title="User Agent">
                {item.status.authentication.info.downstream.userAgent}
              </InfoItem>
            )}

            {item.status.authentication.info.downstream.ipAddress && (
              <InfoItem title="IP Address">
                {item.status.authentication.info.downstream.ipAddress}
              </InfoItem>
            )}

            {item.status.authentication.info.downstream.clientVersion && (
              <InfoItem title="Client Version">
                {item.status.authentication.info.downstream.clientVersion}
              </InfoItem>
            )}
          </>
        )}

      <InfoItem title="State">
        <EditItemWrap
          showComponent={
            <span
              className={twMerge(
                match(item.spec!.state)
                  .with(CoreP.Session_Spec_State.REJECTED, () => "text-red-600")
                  .with(
                    CoreP.Session_Spec_State.PENDING,
                    () => "text-yellow-600",
                  )
                  .otherwise(() => undefined),
              )}
            >
              {match(item.spec!.state)
                .with(CoreP.Session_Spec_State.ACTIVE, () => "Active")
                .with(CoreP.Session_Spec_State.REJECTED, () => "Rejected")
                .with(CoreP.Session_Spec_State.PENDING, () => "Pending")
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Active",
                  value:
                    CoreP.Session_Spec_State[CoreP.Session_Spec_State.ACTIVE],
                },
                {
                  label: "Pending",
                  value:
                    CoreP.Session_Spec_State[CoreP.Session_Spec_State.PENDING],
                },
                {
                  label: "Rejected",
                  value:
                    CoreP.Session_Spec_State[CoreP.Session_Spec_State.REJECTED],
                },
              ]}
              value={CoreP.Session_Spec_State[item.spec!.state]}
              onChange={(v) => {
                if (!v) {
                  return;
                }
                updateItem((clone) => {
                  clone.spec!.state = CoreP.Session_Spec_State[v as "ACTIVE"];
                });
              }}
            />
          }
        />
      </InfoItem>
    </>
  );
};

export const AccessLog = (props: { item: CoreC.Session }) => {
  return <AccessLogViewer sessionRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Session }) => {
  let { item } = props;
  return (
    <div className="w-full">
      <div className="w-full mb-4">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

const SessionStateControl = (props: { item: CoreC.Session }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <EditItemWrap
      label="state"
      showComponent={
        <span
          className={twMerge(
            "text-[0.75rem] font-semibold",
            match(item.spec!.state)
              .with(
                CoreP.Session_Spec_State.ACTIVE,
                () => "text-emerald-600",
              )
              .with(
                CoreP.Session_Spec_State.REJECTED,
                () => "text-red-500",
              )
              .with(
                CoreP.Session_Spec_State.PENDING,
                () => "text-amber-500",
              )
              .otherwise(() => "text-slate-600"),
          )}
        >
          {match(item.spec!.state)
            .with(CoreP.Session_Spec_State.ACTIVE, () => "Active")
            .with(CoreP.Session_Spec_State.REJECTED, () => "Rejected")
            .with(CoreP.Session_Spec_State.PENDING, () => "Pending")
            .otherwise(() => "")}
        </span>
      }
      editComponent={
        <Select
          size="xs"
          data={[
            {
              label: "Active",
              value: CoreP.Session_Spec_State[CoreP.Session_Spec_State.ACTIVE],
            },
            {
              label: "Pending",
              value: CoreP.Session_Spec_State[CoreP.Session_Spec_State.PENDING],
            },
            {
              label: "Rejected",
              value: CoreP.Session_Spec_State[CoreP.Session_Spec_State.REJECTED],
            },
          ]}
          value={CoreP.Session_Spec_State[item.spec!.state]}
          onChange={(v) => {
            if (!v) return;
            const clone = CoreP.Session.clone(item);
            clone.spec!.state = CoreP.Session_Spec_State[v as "ACTIVE"];
            mutationUpdate.mutate(clone);
          }}
        />
      }
    />
  );
};

const SessionExpirationControl = (props: { item: CoreC.Session }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <EditItemWrap
      label="expiration"
      showComponent={
        <span className="text-[0.75rem] font-semibold text-slate-600">
          {item.spec?.expiresAt ? (
            <TimeAgo rfc3339={item.spec.expiresAt} />
          ) : (
            "Not set"
          )}
        </span>
      }
      editComponent={
        <TimestampPicker
          label="Expires at"
          description="Set the expiration time for the Session"
          isFuture
          value={item.spec?.expiresAt}
          onChange={(v) => {
            const clone = CoreP.Session.clone(item);
            clone.spec!.expiresAt = v;
            mutationUpdate.mutate(clone);
          }}
        />
      }
    />
  );
};

export const MainInfo = (props: { item: CoreC.Session }): ResourceMainInfo => {
  const { item } = props;

  return {
    items: [
      ...(item.status?.userRef?.name || item.status?.userRef?.uid
        ? [
            {
              label: "User",
              value: <ResourceListLabel itemRef={item.status.userRef} />,
            },
          ]
        : []),
      ...(item.status?.deviceRef?.name || item.status?.deviceRef?.uid
        ? [
            {
              label: "Device",
              value: <ResourceListLabel itemRef={item.status!.deviceRef} />,
            },
          ]
        : []),
      {
        label: "Type",
        value: <Label>{getType(item)}</Label>,
      },

      {
        label: "State",
        value: <SessionStateControl item={item} />,
      },

      ...(item.status?.isConnected
        ? [
            {
              label: "Connected",
              value: (
                <span className="flex items-center gap-1.5">
                  <span className="relative flex w-2.5 h-2.5 shrink-0">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                    <span className="relative inline-flex rounded-full w-2.5 h-2.5 bg-emerald-500" />
                  </span>
                  <span className="text-[0.75rem] font-semibold text-emerald-600">
                    Live
                  </span>
                </span>
              ),
            },
          ]
        : []),

      {
        label: "Expires",
        value: <SessionExpirationControl item={item} />,
      },

      {
        label: "Session investigation",
        value: <SessionOperationalDetails item={item} />,
        span: "full",
      },
    ],
  };
};
