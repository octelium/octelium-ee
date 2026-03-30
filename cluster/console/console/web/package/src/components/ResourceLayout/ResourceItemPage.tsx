import { Navigate, Outlet } from "react-router-dom";
import PageWrap from "../PageWrap";

import { useContextResource } from "./utils";

import { Resource } from "@/utils/pb";

import { Tabs } from "@mantine/core";
import { useLocation, useNavigate } from "react-router-dom";
import { match } from "ts-pattern";

const ResourceMainBar = (props: { resource: Resource }) => {
  const navigate = useNavigate();
  const { resource } = props;

  const loc = useLocation();

  return (
    <div className="w-full">
      <Tabs
        defaultValue="main"
        value={match(loc.pathname.split("/").reverse().at(0))
          .with("edit", (v) => v)
          .with("actions", (v) => v)
          .with("accesslogs", (v) => v)
          .with("authenticationlogs", (v) => v)
          .with("auditlogs", (v) => v)
          .otherwise(() => "main")}
      >
        <Tabs.List className="mb-2">
          <Tabs.Tab
            value="main"
            onClick={() => {
              navigate(".");
            }}
          >
            Main
          </Tabs.Tab>

          <Tabs.Tab
            value="edit"
            onClick={() => {
              navigate("./edit");
            }}
          >
            Configure
          </Tabs.Tab>

          {/**
           {hasAccessLog(resource) && (
            <Tabs.Tab
              value="accesslogs"
              onClick={() => {
                navigate("./accesslogs");
              }}
            >
              Access Logs
            </Tabs.Tab>
          )}

          {hasAuthenticationLog(resource) && (
            <Tabs.Tab
              value="authenticationlogs"
              onClick={() => {
                navigate("./authenticationlogs");
              }}
            >
              Authentication Logs
            </Tabs.Tab>
          )}

          {hasAuditLog(resource) && (
            <Tabs.Tab
              value="auditlogs"
              onClick={() => {
                navigate("./auditlogs");
              }}
            >
              Audit Logs
            </Tabs.Tab>
          )}
           
           **/}

          <Tabs.Tab
            value="actions"
            onClick={() => {
              navigate("./actions");
            }}
          >
            Actions
          </Tabs.Tab>
        </Tabs.List>
      </Tabs>
    </div>
  );
};

const ResourceItemPage = () => {
  const ctx = useContextResource();

  if (ctx?.isError) {
    return <Navigate to={`/`} />;
  }

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx?.data && (
        <div className="w-full font-bold">
          <div className="mb-16">
            <ResourceMainBar resource={ctx.data} />
          </div>

          <div className="ml-8">
            <Outlet />
          </div>
        </div>
      )}
    </PageWrap>
  );
};

export default ResourceItemPage;
