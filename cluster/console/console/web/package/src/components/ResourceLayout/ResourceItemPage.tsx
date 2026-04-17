import { Resource } from "@/utils/pb";
import { Tabs } from "@mantine/core";
import { LayoutDashboard, Settings, Zap } from "lucide-react";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { match } from "ts-pattern";
import PageWrap from "../PageWrap";
import { useContextResource } from "./utils";

const TABS = [
  { value: "main", label: "Overview", icon: LayoutDashboard, path: "" },
  { value: "edit", label: "Configure", icon: Settings, path: "edit" },
  { value: "actions", label: "Actions", icon: Zap, path: "actions" },
] as const;

type TabValue = (typeof TABS)[number]["value"];

const getActiveTab = (pathname: string): TabValue => {
  const segments = pathname.split("/").filter(Boolean);
  const last = segments.at(-1) ?? "";
  return match(last)
    .with("edit", () => "edit" as const)
    .with("actions", () => "actions" as const)
    .otherwise(() => "main" as const);
};

const ResourceMainBar = (props: { resource: Resource }) => {
  const navigate = useNavigate();
  const loc = useLocation();
  const activeTab = getActiveTab(loc.pathname);

  return (
    <Tabs value={activeTab}>
      <Tabs.List>
        {TABS.map(({ value, label, icon: Icon, path }) => (
          <Tabs.Tab
            key={value}
            value={value}
            leftSection={<Icon size={13} strokeWidth={2.5} />}
            onClick={() => navigate(path)}
          >
            {label}
          </Tabs.Tab>
        ))}
      </Tabs.List>
    </Tabs>
  );
};

const ResourceItemPage = () => {
  const ctx = useContextResource();

  if (ctx?.isError) {
    return <Navigate to="/" replace />;
  }

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
