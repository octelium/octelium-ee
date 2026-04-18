import { Timestamp } from "@/apis/google/protobuf/timestamp";
import AccessLogDataPoint from "@/components/AccessLogViewer/AccessLogDataPoint";
import AccessLogHealthWidget from "@/components/AccessLogViewer/AccessLogWidget";
import AccessLogTopList from "@/components/AccessLogViewer/TopList";
import AuthenticationLogDataPoint from "@/components/AuthenticationLogViewer/AuthenticationLogDataPoint";
import AuthenticationLogHealthWidget from "@/components/AuthenticationLogViewer/AuthenticationLogWidget";
import AuthenticationLogTopList from "@/components/AuthenticationLogViewer/TopList";
import AccessLogSummary from "@/components/LogSummary/AccessLogSummary";
import AuditLogSummary from "@/components/LogSummary/AuditLogSummary";
import AuthenticationLogSummary from "@/components/LogSummary/AuthenticationLogSummary";
import ComponentLogSummary from "@/components/LogSummary/ComponentLogSummary";
import TimestampPicker from "@/components/TimestampPicker";
import { Group, SegmentedControl } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { BarChart3, ChevronRight, Layers } from "lucide-react";
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

export const Item = (props: {
  title: string;
  link?: string;
  children?: React.ReactNode;
  className?: string;
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className={twMerge(
        "bg-white border border-slate-200 rounded-xl overflow-hidden",
        props.className,
      )}
    >
      <div className="flex items-center justify-between px-[18px] py-[13px] border-b border-slate-100">
        <span className="text-[0.78rem] font-bold tracking-[0.05em] uppercase text-slate-800">
          {props.title}
        </span>
        {props.link && (
          <Link
            to={props.link}
            className="flex items-center gap-1 text-[0.72rem] font-semibold text-slate-500 hover:text-slate-900 transition-colors duration-150"
          >
            View all
            <ChevronRight size={11} strokeWidth={2.5} />
          </Link>
        )}
      </div>
      <div className="px-[18px] py-[14px]">{props.children}</div>
    </motion.div>
  );
};

const TimeRangeFilter = (props: {
  from: Timestamp | undefined;
  to: Timestamp | undefined;
  onFromChange: (v: Timestamp | undefined) => void;
  onToChange: (v: Timestamp | undefined) => void;
}) => (
  <Group grow mb="md">
    <TimestampPicker
      label="From"
      disableExcludePast
      description="Start of range"
      value={props.from}
      onChange={props.onFromChange}
    />
    <TimestampPicker
      label="To"
      disableExcludePast
      description="End of range"
      value={props.to}
      onChange={props.onToChange}
    />
  </Group>
);

const AccessLog = () => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  const [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  return (
    <Item title="Access Logs" link="/visibility/accesslogs">
      <TimeRangeFilter
        from={from}
        to={to}
        onFromChange={setFrom}
        onToChange={setTo}
      />
      <AccessLogSummary from={from} to={to} />
      <div className="my-6">
        <AccessLogDataPoint from={from} to={to} />
      </div>
      <AccessLogTopList from={from} to={to} />
    </Item>
  );
};

const AuthenticationLog = () => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  const [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  return (
    <Item title="Authentication Logs" link="/visibility/authenticationlogs">
      <TimeRangeFilter
        from={from}
        to={to}
        onFromChange={setFrom}
        onToChange={setTo}
      />
      <AuthenticationLogSummary from={from} to={to} />
      <div className="my-6">
        <AuthenticationLogDataPoint from={from} to={to} />
      </div>
      <AuthenticationLogTopList from={from} to={to} />
    </Item>
  );
};

const AuditLog = () => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  const [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  return (
    <Item title="Audit Logs" link="/visibility/auditlogs">
      <TimeRangeFilter
        from={from}
        to={to}
        onFromChange={setFrom}
        onToChange={setTo}
      />
      <AuditLogSummary from={from} to={to} />
    </Item>
  );
};

const ComponentLog = () => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);
  const [to, setTo] = React.useState<Timestamp | undefined>(undefined);
  return (
    <Item title="Component Logs" link="/visibility/componentlogs">
      <TimeRangeFilter
        from={from}
        to={to}
        onFromChange={setFrom}
        onToChange={setTo}
      />
      <ComponentLogSummary from={from} to={to} />
    </Item>
  );
};

const LogsTab = () => (
  <motion.div
    key="logs"
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: 6 }}
    transition={{ duration: 0.2, ease: "easeOut" }}
    className="flex flex-col gap-4"
  >
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <Item title="Access health">
        <AccessLogHealthWidget />
      </Item>
      <Item title="Authentication health">
        <AuthenticationLogHealthWidget />
      </Item>
    </div>

    <div className="flex flex-col gap-4">
      <AccessLog />
      <AuthenticationLog />
      <AuditLog />
      <ComponentLog />
    </div>
  </motion.div>
);

const ResourcesTab = () => (
  <motion.div
    key="resources"
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: 6 }}
    transition={{ duration: 0.2, ease: "easeOut" }}
    className="grid grid-cols-2 gap-4"
  >
    <Item title="Users" link="/core/users">
      <UserSummary pieMain showNoItems />
    </Item>
    <Item title="Sessions" link="/core/sessions">
      <SessionSummary pieMain showNoItems />
    </Item>
    <Item title="Devices" link="/core/devices">
      <DeviceSummary pieMain showNoItems />
    </Item>
    <Item title="Services" link="/core/services">
      <ServiceSummary pieMain showNoItems />
    </Item>
    <Item title="Namespaces" link="/core/namespaces">
      <NamespaceSummary />
    </Item>
    <Item title="Credentials" link="/core/credentials">
      <CredentialSummary pieMain showNoItems />
    </Item>
    <Item title="Identity Providers" link="/core/identityproviders">
      <IdentityProviderSummary pieMain showNoItems />
    </Item>
    <Item title="Policies" link="/core/policies">
      <PolicySummary showNoItems />
    </Item>
    <Item title="Authenticators" link="/core/authenticators">
      <AuthenticatorSummary pieMain showNoItems />
    </Item>
    <Item title="Groups" link="/core/groups">
      <GroupSummary showNoItems />
    </Item>
    <Item title="Secrets" link="/core/secrets">
      <SecretSummary showNoItems />
    </Item>
    <Item title="Gateways" link="/core/gateways">
      <GatewaySummary />
    </Item>
    <Item title="Regions" link="/core/regions">
      <RegionSummary />
    </Item>
  </motion.div>
);

type TabValue = "logs" | "resources";

export default () => {
  const [tab, setTab] = React.useState<TabValue>("logs");

  return (
    <div className="w-full flex flex-col gap-5 py-4">
      <div className="flex items-center justify-between">
        <SegmentedControl
          value={tab}
          onChange={(v) => setTab(v as TabValue)}
          fullWidth
          data={[
            {
              value: "logs",
              label: (
                <span className="flex items-center justify-center gap-2 py-0.5">
                  <BarChart3 size={14} strokeWidth={2.5} />
                  <span>Activity & Logs</span>
                </span>
              ),
            },
            {
              value: "resources",
              label: (
                <span className="flex items-center justify-center gap-2 py-0.5">
                  <Layers size={14} strokeWidth={2.5} />
                  <span>Resources</span>
                </span>
              ),
            },
          ]}
          styles={{
            root: {
              backgroundColor: "#f1f5f9",
              border: "1px solid #e2e8f0",
              borderRadius: "10px",
              padding: "3px",
              height: "44px",
            },
            label: {
              fontSize: "0.82rem",
              fontWeight: 700,
              color: "#64748b",
              transition: "color 150ms",
              "&[data-active]": {
                color: "#ffffff",
              },
            },
            indicator: {
              backgroundColor: "#fff",
              borderRadius: "7px",
              boxShadow: "0 1px 4px rgba(15,23,42,0.18)",
            },
          }}
        />

        <span className="text-[0.68rem] font-semibold text-slate-400">
          {tab === "logs"
            ? "Activity trends and security health"
            : "Resource inventory and status"}
        </span>
      </div>

      <AnimatePresence mode="wait">
        {tab === "logs" ? (
          <LogsTab key="logs" />
        ) : (
          <ResourcesTab key="resources" />
        )}
      </AnimatePresence>
    </div>
  );
};
