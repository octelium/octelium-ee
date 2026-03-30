/*
import {
  GetServiceSummaryResponse,
  GetSessionSummaryResponse,
  GetUserSummaryResponse,
} from "@/apis/visibilityv1/core/vcorev1";
import { getClientVisibilityCore } from "@/utils/client";
import { Grid, Group } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import PieChart from "@/components/Charts/PieChart";

export const SummaryItemCount = (props: {
  children?: React.ReactNode;
  count?: number;
}) => {
  const { count, children } = props;
  if (!count || count < 1) {
    return <></>;
  }

  return (
    <div className="flex flex-col items-start justify-start p-2 font-bold m-4 border-l-[4px] border-l-slate-700">
      <span className="text-3xl text-slate-700 mr-1">{count}</span>
      <div className="text-lg text-slate-500">{children}</div>
    </div>
  );
};

export const SummaryItemCountWrap = (props: { children?: React.ReactNode }) => {
  return <Grid>{props.children}</Grid>;
};

const SummaryWrap = (props: { children?: React.ReactNode; href: string }) => {
  return (
    <Link
      to={props.href}
      className="w-full border-[1px] !cursor-pointer rounded-lg p-4 m-4 border-slate-300 font-bold bg-white shadow-sm flex hover:bg-transparent hover:border-slate-400 duration-500 transition-all"
    >
      <div className="w-full">{props.children}</div>
    </Link>
  );
};

const ServiceSummary = (props: { resp: GetServiceSummaryResponse }) => {
  const { resp } = props;
  const data = [
    {
      name: "HTTP",
      value: resp.totalHTTP,
    },
    {
      name: "TCP",
      value: resp.totalTCP,
    },
    {
      name: "SSH",
      value: resp.totalSSH,
    },
    {
      name: "Kubernetes",
      value: resp.totalKubernetes,
    },
    {
      name: "Web",
      value: resp.totalWeb,
    },
    {
      name: "UDP",
      value: resp.totalUDP,
    },
    {
      name: "gRPC",
      value: resp.totalGRPC,
    },
    {
      name: "PostgreSQL",
      value: resp.totalPostgres,
    },
    {
      name: "MySQL",
      value: resp.totalMysql,
    },
    {
      name: "DNS",
      value: resp.totalDNS,
    },
  ];

  return (
    <SummaryWrap href="/core/services">
      <div className="flex pb-8 text-3xl">
        <span className="text-slate-900 mr-4">{resp.totalNumber}</span>
        <span className="text-slate-600">Services</span>
      </div>
      <div className="w-full">
        <div>
          <PieChart data={data} />
        </div>

        <></>
      </div>

      <Grid>
        <SummaryItemCount count={resp.totalAnonymous}>
          Anonymous
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalPublic}>Public</SummaryItemCount>
      </Grid>
    </SummaryWrap>
  );
};

const SessionSummary = (props: { resp: GetSessionSummaryResponse }) => {
  const { resp } = props;
  return (
    <SummaryWrap href="/core/sessions">
      <div className="flex pb-8 text-3xl">
        <span className="text-slate-900 mr-4">{resp.totalNumber}</span>
        <span className="text-slate-600">Sessions</span>
      </div>

      <div>
        <PieChart
          data={[
            { value: resp.totalClient, name: "Client" },
            { value: resp.totalClientlessBrowser, name: "Browser" },
            { value: resp.totalClientlessOAuth2, name: "OAuth2" },
            { value: resp.totalClientlessSDK, name: "SDK" },
          ]}
        />
      </div>
      <Grid>
        <SummaryItemCount count={resp.totalClient}>
          Client-based
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalClientless}>
          Clientless
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalConnected}>
          Connected Clients
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Users</SummaryItemCount>
        <SummaryItemCount count={resp.totalDevice}>Devices</SummaryItemCount>
        <SummaryItemCount count={resp.totalClientlessBrowser}>
          Browser
        </SummaryItemCount>
      </Grid>
    </SummaryWrap>
  );
};

const UserSummary = (props: { resp: GetUserSummaryResponse }) => {
  const { resp } = props;
  const data = [
    {
      name: "Humans",
      value: resp.totalHuman,
      color: "oklch(39.1% 0.09 240.876)",
    },
    {
      name: "Workloads",
      value: resp.totalWorkload,
      color: "oklch(21% 0.034 264.665)",
    },
  ];

  return (
    <SummaryWrap href="/core/sessions">
      <div className="flex pb-8 text-3xl">
        <span className="text-slate-900 mr-4">{resp.totalNumber}</span>
        <span className="text-slate-600">Users</span>
      </div>

      <PieChart
        data={[
          { value: resp.totalHuman, name: "Human" },
          { value: resp.totalWorkload, name: "Workload" },
        ]}
      />

      <Grid>
        <SummaryItemCount count={resp.totalHuman}>Humans</SummaryItemCount>
        <SummaryItemCount count={resp.totalWorkload}>
          Workloads
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalDisabled}>
          Deactivated
        </SummaryItemCount>
      </Grid>
    </SummaryWrap>
  );
};

export default () => {
  const client = getClientVisibilityCore();
  const qryService = useQuery({
    queryKey: ["visibility", "core", "summary", "service"],
    queryFn: async () => {
      const { response } = await client.getServiceSummary({});

      return response;
    },
  });

  const qrySession = useQuery({
    queryKey: ["visibility", "core", "summary", "session"],
    queryFn: async () => {
      const { response } = await client.getSessionSummary({});

      return response;
    },
  });

  const qryUser = useQuery({
    queryKey: ["visibility", "core", "summary", "user"],
    queryFn: async () => {
      const { response } = await client.getUserSummary({});

      return response;
    },
  });

  return (
    <div className="w-full">
      {qryService.isSuccess && <ServiceSummary resp={qryService.data} />}
      {qrySession.isSuccess && <SessionSummary resp={qrySession.data} />}
      {qryUser.isSuccess && <UserSummary resp={qryUser.data} />}
    </div>
  );
};
*/

export default () => {
  return <></>;
};
