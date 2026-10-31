import { Space_Status_Type } from "@/apis/cordiumv1/cordiumv1";
import * as CoreC from "@/apis/corev1/corev1";
import * as CoreP from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import {
  formatSeconds,
  HeroStat,
  Meter,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  remainingPercent,
  toDate,
  TONE_TEXT,
  TonePill,
} from "@/components/ResourceLayout/overview";
import {
  authorizationItems,
  RefLabel,
} from "@/components/ResourceLayout/overviewItems";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo, { useNow } from "@/components/TimeAgo";
import TimestampPicker from "@/components/TimestampPicker";
import { twMerge } from "@/lib/utils";
import {
  getSpaceTypeLabel,
  spaceRefOf,
  templateRefOf,
  workspaceRefOf,
} from "@/pages/cordium/utils";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getResourceRef } from "@/utils/pb";
import {
  Ban,
  Boxes,
  Clock3,
  Fingerprint,
  Laptop,
  Lock,
  MapPin,
  MonitorSmartphone,
  Network,
  Server,
  ShieldCheck,
  ShieldQuestion,
  UserRound,
} from "lucide-react";
import {
  authenticationItems,
  ClientIcon,
  clientItems,
  connectionItems,
  describeClient,
  formatLocation,
  getAALMeta,
  getAuthenticationSource,
  getAuthenticatorAction,
  getExpiry,
  getMethodLabel,
  getSignIn,
  getTransportLabel,
  LocationValue,
  scopeItems,
  servedItems,
} from "./Info";
import { getCordiumSessionInfo } from "./utils";

export const AccessLog = (props: { item: CoreC.Session }) => {
  return <AccessLogViewer sessionRef={getResourceRef(props.item)} />;
};

const SessionExpirationControl = (props: { item: CoreC.Session }) => {
  const { item } = props;
  const now = useNow();
  const mutationUpdate = useUpdateResource();
  const expiry = getExpiry(item, now);

  const show = (
    <span
      className={twMerge(
        "text-sm font-bold tabular-nums",
        TONE_TEXT[expiry.tone],
      )}
    >
      {item.spec?.expiresAt ? (
        <TimeAgo rfc3339={item.spec.expiresAt} />
      ) : (
        "Not set"
      )}
    </span>
  );

  if (item.metadata?.isSystem) return show;

  return (
    <EditItemWrap
      mutation={mutationUpdate}
      label="expiration"
      showComponent={show}
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

const LiveDot = () => (
  <span className="relative ml-2 inline-flex h-2 w-2 align-middle">
    <span className="absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75 motion-safe:animate-ping" />
    <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
  </span>
);

const SessionHero = (props: { item: CoreC.Session }) => {
  const { item } = props;
  const now = useNow();
  const status = item.status;
  const state = item.spec?.state;
  const isClient = status?.type === CoreP.Session_Status_Type.CLIENT;
  const connection = isClient && status?.isConnected ? status.connection : undefined;
  const expiry = getExpiry(item, now);
  const action = getAuthenticatorAction(item);
  const signIn = getSignIn(item);
  const source = getAuthenticationSource(signIn);
  const info = status?.authentication?.info;
  const location = formatLocation(info?.geoip);
  const lifetime = remainingPercent(
    toDate(item.metadata?.createdAt),
    expiry.expiresAt,
    now,
  );

  const tone =
    state === CoreP.Session_Spec_State.REJECTED ||
    status?.isLocked ||
    expiry.isExpired
      ? "danger"
      : state === CoreP.Session_Spec_State.PENDING
        ? "warning"
        : connection || !isClient
          ? "success"
          : "neutral";

  const detail = isClient
    ? [
        describeClient(item),
        connection?.startedAt && (
          <>
            connected <TimeAgo rfc3339={connection.startedAt} />
          </>
        ),
        connection?.lastSeenAt && (
          <>
            last seen <TimeAgo rfc3339={connection.lastSeenAt} />
          </>
        ),
      ]
    : [
        source && (
          <>
            Signed in via {source.name || source.uid}
            {signIn?.setAt && (
              <>
                {" "}
                <TimeAgo rfc3339={signIn.setAt} />
              </>
            )}
          </>
        ),
        info?.downstream?.ipAddress,
        location,
      ];

  return (
    <OverviewHero
      tone={tone}
      icon={<ClientIcon item={item} />}
      title={
        isClient ? (
          connection ? (
            <>
              Connected via {getTransportLabel(connection.type)}
              <LiveDot />
            </>
          ) : (
            "Client not connected"
          )
        ) : (
          describeClient(item)
        )
      }
      detail={detail.filter(Boolean).map((part, index) => (
        <span key={index}>
          {index > 0 && " · "}
          {part}
        </span>
      ))}
      meter={
        lifetime !== undefined &&
        expiry.left !== undefined &&
        !expiry.isExpired && (
          <Meter
            value={lifetime}
            label="Session lifetime left"
            tone={expiry.tone === "warning" ? "warning" : "success"}
            caption={`${formatSeconds(expiry.left)} of the Session lifetime left`}
          />
        )
      }
      aside={
        <HeroStat
          label="Expires"
          value={<SessionExpirationControl item={item} />}
        />
      }
    >
      {status?.isLocked && (
        <OverviewNotice tone="danger" icon={Lock} title="Locked by the Cluster.">
          Every request of this Session is rejected until it is unlocked.
        </OverviewNotice>
      )}
      {expiry.isExpired && (
        <OverviewNotice tone="danger" icon={Clock3} title="Expired.">
          The Session can no longer authenticate or access any Service.
        </OverviewNotice>
      )}
      {state === CoreP.Session_Spec_State.REJECTED && (
        <OverviewNotice tone="danger" icon={Ban} title="Rejected.">
          The Session cannot access any Service.
        </OverviewNotice>
      )}
      {state === CoreP.Session_Spec_State.PENDING && (
        <OverviewNotice tone="warning" icon={Clock3} title="Pending activation.">
          The Session cannot access any Service until it is set to Active.
        </OverviewNotice>
      )}
      {action && (
        <OverviewNotice tone={action.tone} icon={ShieldQuestion} title={action.title}>
          {action.detail}
          {status?.requiredAuthenticatorRef &&
            (status.requiredAuthenticatorRef.name ||
              status.requiredAuthenticatorRef.uid) && (
              <span className="ml-1 inline-flex align-middle">
                <RefLabel itemRef={status.requiredAuthenticatorRef} />
              </span>
            )}
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: { item: CoreC.Session }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const info = status?.authentication?.info;
  const aal = getAALMeta(info?.aal);
  const cordiumInfo = getCordiumSessionInfo(item);
  const workspaceRef = workspaceRefOf(cordiumInfo?.workspaceRef);
  const spaceRef = spaceRefOf(cordiumInfo?.spaceRef);
  const templateRef = templateRefOf(cordiumInfo?.templateRef);

  return {
    header: <SessionHero item={item} />,
    groupOrder: [
      { title: "Authentication", icon: Fingerprint },
      { title: "Client & location", icon: MonitorSmartphone },
      { title: "Connection", icon: Network },
      { title: "Served Services", icon: Server },
      { title: "Authorization", icon: ShieldCheck },
      { title: "Cordium", icon: Boxes },
    ],
    items: [
      {
        label: "User",
        primary: true,
        icon: UserRound,
        value: status?.userRef?.name || status?.userRef?.uid ? (
          <ResourceListLabel itemRef={status.userRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Device",
        primary: true,
        icon: Laptop,
        value: status?.deviceRef?.name || status?.deviceRef?.uid ? (
          <ResourceListLabel itemRef={status.deviceRef} />
        ) : (
          <MutedValue>No Device</MutedValue>
        ),
      },
      {
        label: "Authentication",
        primary: true,
        icon: Fingerprint,
        value: (
          <span className="flex flex-wrap items-center gap-1.5">
            <span className="font-semibold">{getMethodLabel(info?.type)}</span>
            {aal && <TonePill tone={aal.tone}>{aal.label}</TonePill>}
          </span>
        ),
      },
      {
        label: "Location",
        primary: true,
        icon: MapPin,
        value: <LocationValue info={info} />,
      },

      ...authenticationItems(item, "Authentication"),
      ...clientItems(item, "Client & location"),
      ...connectionItems(item, "Connection"),
      ...servedItems(item, "Served Services"),
      ...authorizationItems(item.spec?.authorization),
      ...scopeItems(item, "Authorization"),

      ...(workspaceRef
        ? [
            {
              label: "Workspace",
              group: "Cordium",
              value: <ResourceListLabel itemRef={workspaceRef} />,
            },
          ]
        : []),
      ...(spaceRef
        ? [
            {
              label: "Space",
              group: "Cordium",
              value: <ResourceListLabel itemRef={spaceRef} />,
            },
          ]
        : []),
      ...(templateRef
        ? [
            {
              label: "Template",
              group: "Cordium",
              value: <ResourceListLabel itemRef={templateRef} />,
            },
          ]
        : []),
      ...(cordiumInfo &&
      cordiumInfo.spaceType !== Space_Status_Type.SPACE_TYPE_UNSET
        ? [
            {
              label: "Space type",
              group: "Cordium",
              value: <Label>{getSpaceTypeLabel(cordiumInfo.spaceType)}</Label>,
            },
          ]
        : []),
    ],
  };
};
