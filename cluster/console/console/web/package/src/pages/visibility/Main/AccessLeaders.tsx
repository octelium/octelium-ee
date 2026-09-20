import { AccessLog_Entry_Common_Reason_Type } from "@/apis/corev1/corev1";
import { Panel } from "@/components/Dashboard/components";
import {
  useAccessDenyReasons,
  useAccessTop,
} from "@/components/Dashboard/queries";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  PanelTop,
  Shield,
  ShieldX,
  Terminal,
  Trophy,
  User,
} from "lucide-react";
import Leaderboard, { fromResources, LeaderboardItem } from "./Leaderboard";

const reasonLabel = (reason: number) =>
  (AccessLog_Entry_Common_Reason_Type[reason] ?? "Unknown")
    .toLowerCase()
    .replace(/_/g, " ")
    .replace(/^./, (character) => character.toUpperCase());

const AccessLeaders = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const users = useAccessTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const services = useAccessTop(
    "service",
    periodMinutes,
    QUERY_PRIORITY.normal,
  );
  const policies = useAccessTop("policy", periodMinutes);
  const sessions = useAccessTop("session", periodMinutes);
  const denials = useAccessDenyReasons(periodMinutes, QUERY_PRIORITY.low);

  const denyItems: LeaderboardItem[] = (denials.data?.items ?? []).map(
    (item) => ({
      id: `${item.reason}-${item.policyRef?.uid ?? ""}`,
      name: reasonLabel(item.reason),
      sub: item.policyRef?.name ? `via ${item.policyRef.name}` : undefined,
      count: n(item.count),
      accent: STATUS_COLORS.critical,
    }),
  );

  return (
    <Panel
      icon={Trophy}
      title="Access leaders"
      description={`Who and what generated the authorization traffic of the last ${rangeLabel}`}
      to="/visibility/accesslogs"
      toLabel="Access logs"
    >
      <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
        <Leaderboard
          title="Users"
          icon={User}
          items={fromResources(
            (users.data?.items ?? []).map((item: any) => ({
              resource: item.user,
              count: item.count,
            })),
          )}
          totalCount={n(users.data?.totalCount)}
          totalOther={n(users.data?.totalOther)}
          to="/core/users"
          isLoading={users.isLoading}
          unit="requests"
          emptyLabel={`No identity made a request in the last ${rangeLabel}.`}
        />

        <Leaderboard
          title="Services"
          icon={PanelTop}
          items={fromResources(
            (services.data?.items ?? []).map((item: any) => ({
              resource: item.service,
              count: item.count,
            })),
          )}
          totalCount={n(services.data?.totalCount)}
          totalOther={n(services.data?.totalOther)}
          to="/core/services"
          isLoading={services.isLoading}
          unit="requests"
          emptyLabel={`No Service was reached in the last ${rangeLabel}.`}
        />

        <Leaderboard
          title="Policies"
          icon={Shield}
          items={fromResources(
            (policies.data?.items ?? []).map((item: any) => ({
              resource: item.policy,
              count: item.count,
            })),
          )}
          totalCount={n(policies.data?.totalCount)}
          totalOther={n(policies.data?.totalOther)}
          to="/core/policies"
          isLoading={policies.isLoading}
          unit="decisions"
          emptyLabel={`No Policy matched a request in the last ${rangeLabel}.`}
        />

        <Leaderboard
          title="Sessions"
          icon={Terminal}
          items={fromResources(
            (sessions.data?.items ?? []).map((item: any) => ({
              resource: item.session,
              count: item.count,
            })),
          )}
          totalCount={n(sessions.data?.totalCount)}
          totalOther={n(sessions.data?.totalOther)}
          to="/core/sessions"
          isLoading={sessions.isLoading}
          unit="requests"
          emptyLabel={`No Session made a request in the last ${rangeLabel}.`}
        />

        <div className="xl:col-span-2">
          <Leaderboard
            title="Why requests were denied"
            icon={ShieldX}
            items={denyItems}
            totalCount={n(denials.data?.totalCount)}
            totalOther={n(denials.data?.totalOther)}
            to="/visibility/accesslogs?status=DENIED"
            isLoading={denials.isLoading}
            unit="denials"
            emptyLabel={`Nothing was denied in the last ${rangeLabel}.`}
          />
        </div>
      </div>
    </Panel>
  );
};

export default AccessLeaders;
