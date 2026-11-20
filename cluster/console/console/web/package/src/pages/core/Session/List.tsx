import { Session_Spec_State, Session_Status_Type } from "@/apis/corev1/corev1";

import { Session } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import TimeAgo from "@/components/TimeAgo";
import StatusBadge from "@/components/StatusBadge";
import { MdTimer } from "react-icons/md";
import { match } from "ts-pattern";

import { GetSessionSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import PieChart from "@/components/Charts/PieChart";
import { getCountryName } from "@/components/GeoMap/geo";
import { CountryFlag } from "@/components/SelectCountry/CountryFlag";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { BsFillTerminalFill } from "react-icons/bs";
import { IoBrowsers } from "react-icons/io5";
import {
  CircleCheck,
  CircleX,
  Clock3,
  Globe2,
  Laptop,
  Monitor,
  Terminal,
  Users,
  Wifi,
} from "lucide-react";
import { Link, useSearchParams } from "react-router";
import { twMerge } from "@/lib/utils";
import { workspaceRefOf } from "@/pages/cordium/utils";
import { getCordiumSessionInfo } from "./utils";

export const getType = (svc: Session) => {
  return match(svc.status?.type)
    .with(Session_Status_Type.CLIENT, () => (
      <>
        <BsFillTerminalFill size={14} />
        <span className="ml-2">Client</span>
      </>
    ))
    .with(Session_Status_Type.CLIENTLESS, () => (
      <>
        {svc.status?.isBrowser ? (
          <>
            <IoBrowsers size={14} />
            <span className="ml-2">Browser</span>
          </>
        ) : (
          <span>Clientless</span>
        )}
      </>
    ))

    .otherwise(() => <></>);
};

const getState = (svc: Session) => {
  return match(svc.spec?.state)
    .with(Session_Spec_State.ACTIVE, () => (
      <StatusBadge label="Active" tone="success" compact />
    ))
    .with(Session_Spec_State.REJECTED, () => (
      <StatusBadge label="Rejected" tone="danger" compact />
    ))
    .with(Session_Spec_State.PENDING, () => (
      <StatusBadge label="Pending" tone="warning" compact />
    ))
    .otherwise(() => "");
};

export const getSessionLocation = (item: Session) => {
  const geoip = item.status?.authentication?.info?.geoip;
  const code = geoip?.country?.code;
  if (!code && !geoip?.city?.name) return undefined;
  return {
    code,
    label: [
      geoip?.city?.name,
      code ? getCountryName(code, geoip?.country?.name) : geoip?.country?.name,
    ]
      .filter(Boolean)
      .join(", "),
  };
};

export const LabelComponent = (props: { item: Session }) => {
  const { item } = props;
  const workspaceRef = workspaceRefOf(getCordiumSessionInfo(item)?.workspaceRef);
  const location = getSessionLocation(item);

  return (
    <ResourceListLabelWrap>
      {item.spec?.expiresAt && (
        <ResourceListLabel>
          <span className="mr-1">
            <MdTimer size={14} />
          </span>
          <span>
            Expires <TimeAgo rfc3339={item.spec.expiresAt} />
          </span>
        </ResourceListLabel>
      )}
      {item.status?.authentication &&
        item.status.authentication.setAt && (
          <ResourceListLabel label="Last Authentication">
            <TimeAgo rfc3339={item.status.authentication.setAt} />
          </ResourceListLabel>
        )}
      <ResourceListLabel>{getType(item)}</ResourceListLabel>
      {getState(item)}
      {item.status?.isConnected && (
        <ResourceListLabel>
          <div className="flex items-center">
            <div className="flex relative w-[10px] h-[10px]">
              <div className="bg-green-500 rounded-full w-[10px] h-[10px]"></div>
            </div>
            <span className="ml-1">Connected</span>
          </div>
        </ResourceListLabel>
      )}
      {location && (
        <ResourceListLabel label="Location">
          <span className="inline-flex min-w-0 items-center gap-1.5">
            <CountryFlag
              code={location.code}
              className="h-2.5 w-3.5 shrink-0 rounded-[1px]"
            />
            <span className="truncate">{location.label}</span>
          </span>
        </ResourceListLabel>
      )}
      {(item.status?.userRef?.name || item.status?.userRef?.uid) && (
        <ResourceListLabel itemRef={item.status.userRef} />
      )}
      {(item.status?.deviceRef?.name || item.status?.deviceRef?.uid) && (
        <ResourceListLabel itemRef={item.status!.deviceRef}></ResourceListLabel>
      )}
      {workspaceRef && <ResourceListLabel itemRef={workspaceRef} />}
    </ResourceListLabelWrap>
  );
};

const COUNTRY_CHIPS = 10;

const SessionCountries = (props: { resp: GetSessionSummaryResponse }) => {
  const [searchParams] = useSearchParams();
  const active = searchParams.get("countryCode")?.toUpperCase();
  const countries = Object.entries(props.resp.totalByCountry ?? {})
    .map(([code, count]) => ({ code: code.toUpperCase(), count: Number(count) }))
    .filter((item) => item.count > 0)
    .sort((a, b) => b.count - a.count || a.code.localeCompare(b.code));

  if (countries.length === 0) return null;

  const shown = countries.slice(0, COUNTRY_CHIPS);
  if (active && !shown.some((item) => item.code === active)) {
    const match = countries.find((item) => item.code === active);
    if (match) shown.push(match);
  }

  return (
    <nav
      aria-label="Sessions by country"
      className="mt-3 flex flex-wrap items-center gap-1.5"
    >
      <span className="mr-1 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
        By country
      </span>
      {shown.map((item) => {
        const isActive = item.code === active;
        const name = getCountryName(item.code);
        return (
          <Link
            key={item.code}
            to={toURLWithQry(`/core/sessions`, { countryCode: item.code })}
            title={`${name}: ${item.count.toLocaleString()} Sessions`}
            aria-current={isActive ? "true" : undefined}
            className={twMerge(
              "inline-flex items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600 outline-none transition-colors duration-150",
              "hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400",
              isActive && "border-slate-400 bg-slate-100 text-slate-900",
            )}
          >
            <CountryFlag
              code={item.code}
              className="h-2.5 w-3.5 shrink-0 rounded-[1px]"
            />
            <span>{name}</span>
            <span className="tabular-nums text-slate-500">
              {item.count.toLocaleString()}
            </span>
          </Link>
        );
      })}
      {countries.length > shown.length && (
        <span className="text-micro font-normal text-slate-500">
          +{countries.length - shown.length} more
        </span>
      )}
    </nav>
  );
};

const DoSummary = (props: { resp: GetSessionSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/core/sessions">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalClient}
          to={toURLWithQry(`/core/sessions`, {
            type: Session_Status_Type[Session_Status_Type.CLIENT],
          })}
          active={
            searchParams.get(`type`) ===
            Session_Status_Type[Session_Status_Type.CLIENT]
          }
          icon={Terminal}
        >
          Clients
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalClientless}
          to={toURLWithQry(`/core/sessions`, {
            type: Session_Status_Type[Session_Status_Type.CLIENTLESS],
          })}
          active={
            searchParams.get(`type`) ===
            Session_Status_Type[Session_Status_Type.CLIENTLESS]
          }
          icon={Globe2}
        >
          Clientless
        </SummaryItemCount>

        <SummaryItemCount
          count={resp.totalClientlessBrowser}
          to={toURLWithQry(`/core/sessions`, {
            isBrowser: "true",
          })}
          active={searchParams.get(`isBrowser`) === "true"}
          icon={Monitor}
        >
          Browsers
        </SummaryItemCount>

        <SummaryItemCount
          count={resp.totalActive}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.ACTIVE],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.ACTIVE]
          }
          icon={CircleCheck}
        >
          Active
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRejected}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.REJECTED],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.REJECTED]
          }
          icon={CircleX}
        >
          Rejected
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPending}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.PENDING],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.PENDING]
          }
          icon={Clock3}
        >
          Pending
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalConnected}
          to={toURLWithQry(`/core/sessions`, {
            isConnected: "true",
          })}
          active={searchParams.get(`isConnected`) === "true"}
          icon={Wifi}
        >
          Connected
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={Users}>
          Users
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalDevice} icon={Laptop}>
          Devices
        </SummaryItemCount>
      </SummaryItemCountWrap>
      <SessionCountries resp={resp} />
    </div>
  );
};

export const Summary = (props: {
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Session"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getSessionSummary(
        {},
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const d = qry.data;

  return (
    <div>
      {d.totalNumber > 0 && (
        <div>
          <DoSummary resp={qry.data} />
          {props.pieMain && (
            <div className="w-full grid grid-cols-2 gap-1">
              <PieChart
                data={[
                  { name: "Client", value: d.totalClient },
                  { name: "Clientless", value: d.totalClientless },
                ]}
              />
              <PieChart
                data={[
                  { name: "Active", value: d.totalActive },
                  { name: "Rejected", value: d.totalRejected },
                  { name: "Pending", value: d.totalPending },
                ]}
              />
            </div>
          )}

          {props.pieMain && (
            <div className="w-full">
              <PieChart
                data={[
                  { name: "Browser", value: d.totalClientlessBrowser },
                  { name: "OAuth2", value: d.totalClientlessOAuth2 },
                  { name: "SDK", value: d.totalClientlessSDK },
                ]}
              />
            </div>
          )}
        </div>
      )}
      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
