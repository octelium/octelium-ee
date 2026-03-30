import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import AccessLogDataPoint from "@/components/AccessLogViewer/AccessLogDataPoint";
import AccessLogTopList from "@/components/AccessLogViewer/TopList";
import AuthenticationLogDataPoint from "@/components/AuthenticationLogViewer/AuthenticationLogDataPoint";
import AuthenticationLogTopList from "@/components/AuthenticationLogViewer/TopList";
import AccessLogSummary from "@/components/LogSummary/AccessLogSummary";
import AuditLogSummary from "@/components/LogSummary/AuditLogSummary";
import AuthenticationLogSummary from "@/components/LogSummary/AuthenticationLogSummary";
import ComponentLogSummary from "@/components/LogSummary/ComponentLogSummary";
import TimestampPicker from "@/components/TimestampPicker";
import { Group } from "@mantine/core";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { Summary as AuthenticatorSummary } from "../../core/Authenticator/List";
import { Summary as CredentialSummary } from "../../core/Credential/List";
import { Summary as DeviceSummary } from "../../core/Device/List";
import { Summary as GatewaySummary } from "../../core/Gateway/List";
import { Summary as GroupSummary } from "../../core/Group/List";
import { Summary as IdentityProviderSummary } from "../../core/IdentityProvider/List";
import { Summary as NamespaceSummary } from "../../core/Namespace/List";
import { Summary as PolicySummary } from "../../core/Policy/List";
import { Summary as RegionSummary } from "../../core/Region/List";
import { Summary as SecretSummary } from "../../core/Secret/List";
import { Summary as ServiceSummary } from "../../core/Service/List";
import { Summary as SessionSummary } from "../../core/Session/List";
import { Summary as UserSummary } from "../../core/User/List";

const Item = (props: {
  title: string;
  link?: string;
  children?: React.ReactNode;
}) => {
  return (
    <div
      className={twMerge(
        "b-1 border-gray-200 rounded-lg shadow-lg bg-white p-4",
        "shadow shadow-slate-200 w-full",
        "border-[1px] border-slate-300",
      )}
    >
      <div className="font-bold text-2xl mb-2 flex items-center">
        <span> {props.title}</span>
        {props.link && (
          <Link
            className="ml-3 font-bold text-lg text-shadow-2xs text-gray-500 hover:text-black transition-all duration-500"
            to={props.link}
          >
            More
          </Link>
        )}
      </div>
      {props.children}
    </div>
  );
};

export const VisibilityCore = () => {
  return (
    <div className="grid grid-cols-2 gap-4 mt-8">
      <Item title="Users">
        <UserSummary pieMain showNoItems />
      </Item>

      <Item title="Sessions">
        <SessionSummary pieMain showNoItems />
      </Item>

      <Item title="Devices">
        <DeviceSummary pieMain showNoItems />
      </Item>

      <Item title="Services">
        <ServiceSummary pieMain showNoItems />
      </Item>

      <Item title="Namespaces">
        <NamespaceSummary />
      </Item>

      <Item title="Credentials">
        <CredentialSummary pieMain showNoItems />
      </Item>

      <Item title="Identity Providers">
        <IdentityProviderSummary pieMain showNoItems />
      </Item>

      <Item title="Policies">
        <PolicySummary showNoItems />
      </Item>

      <Item title="Authenticators">
        <AuthenticatorSummary pieMain showNoItems />
      </Item>

      <Item title="Groups">
        <GroupSummary showNoItems />
      </Item>

      <Item title="Secrets">
        <SecretSummary showNoItems />
      </Item>
      <Item title="Gateways">
        <GatewaySummary />
      </Item>
      <Item title="Regions">
        <RegionSummary />
      </Item>
    </div>
  );
};

export default () => {
  return (
    <div className="w-full my-4">
      <VisibilityCore />

      <div className="my-4">
        <AccessLog />
      </div>

      <div className="my-4">
        <AuthenticationLog />
      </div>

      <div className="my-4">
        <AuditLog />
      </div>

      <div className="my-4">
        <ComponentLog />
      </div>
    </div>
  );
};

const AccessLog = () => {
  let [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  let [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  let [interval, setInterval] = React.useState<Duration | undefined>(undefined);

  return (
    <Item title="Access Logs" link={"/visibility/accesslogs"}>
      <Group grow>
        <TimestampPicker
          label="From"
          disableExcludePast
          description="Set the start timestamp for the chart"
          value={from}
          onChange={(v) => {
            setFrom(v);
          }}
        />
        <TimestampPicker
          label="To"
          description="Set the end timestamp for the chart"
          disableExcludePast
          value={to}
          onChange={(v) => {
            setTo(v);
          }}
        />
      </Group>

      <div>
        <AccessLogSummary from={from} to={to} />
        <div className="my-8">
          <AccessLogDataPoint from={from} to={to} />
        </div>
        <AccessLogTopList from={from} to={to} />
      </div>
    </Item>
  );
};

const AuthenticationLog = () => {
  let [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  let [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  let [interval, setInterval] = React.useState<Duration | undefined>(undefined);

  return (
    <Item title="Authentication Logs" link={"/visibility/authenticationlogs"}>
      <Group grow>
        <TimestampPicker
          label="From"
          disableExcludePast
          description="Set the start timestamp for the chart"
          value={from}
          onChange={(v) => {
            setFrom(v);
          }}
        />
        <TimestampPicker
          label="To"
          description="Set the end timestamp for the chart"
          disableExcludePast
          value={to}
          onChange={(v) => {
            setTo(v);
          }}
        />
      </Group>

      <div>
        <AuthenticationLogSummary from={from} to={to} />
        <div className="my-8">
          <AuthenticationLogDataPoint from={from} to={to} />
        </div>
        <AuthenticationLogTopList from={from} to={to} />
      </div>
    </Item>
  );
};

const ComponentLog = () => {
  let [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  let [to, setTo] = React.useState<Timestamp | undefined>(undefined);

  return (
    <Item title="Component Logs" link={"/visibility/componentlogs"}>
      <Group grow>
        <TimestampPicker
          label="From"
          disableExcludePast
          description="Set the start timestamp for the chart"
          value={from}
          onChange={(v) => {
            setFrom(v);
          }}
        />
        <TimestampPicker
          label="To"
          description="Set the end timestamp for the chart"
          disableExcludePast
          value={to}
          onChange={(v) => {
            setTo(v);
          }}
        />
      </Group>

      <div>
        <ComponentLogSummary from={from} to={to} />
      </div>
    </Item>
  );
};

const AuditLog = () => {
  let [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  let [to, setTo] = React.useState<Timestamp | undefined>(undefined);

  return (
    <Item title="Audit Logs" link={"/visibility/auditlogs"}>
      <Group grow>
        <TimestampPicker
          label="From"
          disableExcludePast
          description="Set the start timestamp for the chart"
          value={from}
          onChange={(v) => {
            setFrom(v);
          }}
        />
        <TimestampPicker
          label="To"
          description="Set the end timestamp for the chart"
          disableExcludePast
          value={to}
          onChange={(v) => {
            setTo(v);
          }}
        />
      </Group>

      <div>
        <AuditLogSummary from={from} to={to} />
      </div>
    </Item>
  );
};
