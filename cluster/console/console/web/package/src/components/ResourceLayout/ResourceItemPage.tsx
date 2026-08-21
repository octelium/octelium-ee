import { getResourceComponentInfoFromResource } from "@/pages/utils/resourceRegistry";
import {
  hasAccessLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
} from "@/utils/pb";
import { SegmentedControl } from "@mantine/core";
import {
  ChartNoAxesCombined,
  Library,
  LayoutDashboard,
  Settings,
  ShieldEllipsis,
  ShieldUser,
  Terminal,
} from "lucide-react";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { match } from "ts-pattern";
import PageWrap from "../PageWrap";
import { useContextResource } from "./utils";

interface Tab {
  value: string;
  label: string;
  icon: React.FC<any>;
  path: string;
}

const BASE_TABS: Tab[] = [
  { value: "main", label: "Overview", icon: LayoutDashboard, path: "" },
];

const ACCESS_LOG_TAB: Tab = {
  value: "accesslogs",
  label: "Access Logs",
  icon: ShieldEllipsis,
  path: "accesslogs",
};

const AUTH_LOG_TAB: Tab = {
  value: "authenticationlogs",
  label: "Auth Logs",
  icon: ShieldUser,
  path: "authenticationlogs",
};

const METRICS_TAB: Tab = {
  value: "metrics",
  label: "Metrics",
  icon: ChartNoAxesCombined,
  path: "metrics",
};

const SSH_TAB: Tab = {
  value: "ssh",
  label: "SSH Recordings",
  icon: Terminal,
  path: "ssh",
};

const AUDIT_LOG_TAB: Tab = {
  value: "auditlogs",
  label: "Audit Logs",
  icon: Library,
  path: "auditlogs",
};

const getActiveTab = (pathname: string): string => {
  const segments = pathname.split("/").filter(Boolean);
  const last = segments.at(-1) ?? "";
  return match(last)
    .with("edit", () => "edit")
    .with("accesslogs", () => "accesslogs")
    .with("authenticationlogs", () => "authenticationlogs")
    .with("metrics", () => "metrics")
    .with("ssh", () => "ssh")
    .with("auditlogs", () => "auditlogs")
    .otherwise(() => "main");
};

const buildTabs = (resource: Resource): Tab[] => {
  const tabs = [...BASE_TABS];
  if (!getResourceComponentInfoFromResource(resource)?.unEditable) {
    tabs.push({
      value: "edit",
      label: "Configure",
      icon: Settings,
      path: "edit",
    });
  }
  if (
    resource.apiVersion === "core/v1" &&
    (resource.kind === "Service" || resource.kind === "Namespace")
  ) {
    tabs.push(METRICS_TAB);
  }
  if (
    resource.apiVersion === "core/v1" &&
    resource.kind === "Service" &&
    hasSSHSessionLog(resource)
  ) {
    tabs.push(SSH_TAB);
  }
  if (hasAccessLog(resource)) tabs.push(ACCESS_LOG_TAB);
  if (hasAuthenticationLog(resource)) tabs.push(AUTH_LOG_TAB);
  tabs.push(AUDIT_LOG_TAB);
  return tabs;
};

const ResourceMainBar = (props: { resource: Resource }) => {
  const navigate = useNavigate();
  const loc = useLocation();
  const activeTab = getActiveTab(loc.pathname);
  const tabs = buildTabs(props.resource);

  return (
    <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="max-w-full overflow-x-auto pb-1">
        <SegmentedControl
          value={activeTab}
          onChange={(v) => {
            const tab = tabs.find((t) => t.value === v);
            if (tab) {
              navigate(tab.path, {
                state: loc.state,
                preventScrollReset: true,
              });
            }
          }}
          data={tabs.map(({ value, label, icon: Icon }) => ({
            value,
            label: (
              <span className="flex items-center gap-1.5 px-1">
                <Icon size={13} strokeWidth={2.5} />
                {label}
              </span>
            ),
          }))}
        />
      </div>

      <div className="flex min-w-0 items-center gap-1.5 sm:justify-end">
        <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-500">
          {props.resource.kind}
        </span>
        <span className="text-slate-300 text-xs">·</span>
        <span className="max-w-[200px] truncate text-[0.68rem] font-semibold text-slate-400">
          {props.resource.metadata?.name}
        </span>
      </div>
    </div>
  );
};

const ResourceItemPage = () => {
  const ctx = useContextResource();

  if (ctx?.isError) return <Navigate to="/" replace />;
  if (!ctx) return null;

  return (
    <PageWrap qry={ctx}>
      {ctx.data && (
        <div className="w-full flex flex-col gap-6">
          <ResourceMainBar resource={ctx.data} />
          <Outlet />
        </div>
      )}
    </PageWrap>
  );
};

export default ResourceItemPage;
