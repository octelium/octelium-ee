import { Panel } from "@/components/Dashboard/components";
import { useAuthTop } from "@/components/Dashboard/queries";
import { useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Fingerprint, KeyRound, ShieldUser, User } from "lucide-react";
import Leaderboard, { fromResources } from "./Leaderboard";

const IdentityLeaders = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const users = useAuthTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const providers = useAuthTop("identityProvider", periodMinutes);
  const credentials = useAuthTop("credential", periodMinutes);

  return (
    <Panel
      icon={ShieldUser}
      title="Identity leaders"
      description={`Who authenticated over the last ${rangeLabel}, and what they authenticated with`}
      to="/visibility/authenticationlogs"
      toLabel="Authentication logs"
    >
      <div className="grid grid-cols-1 gap-3 xl:grid-cols-3">
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
          unit="logins"
          emptyLabel={`Nobody authenticated in the last ${rangeLabel}.`}
        />

        <Leaderboard
          title="Identity providers"
          icon={Fingerprint}
          items={fromResources(
            (providers.data?.items ?? []).map((item: any) => ({
              resource: item.identityProvider,
              count: item.count,
            })),
          )}
          totalCount={n(providers.data?.totalCount)}
          totalOther={n(providers.data?.totalOther)}
          to="/core/identityproviders"
          isLoading={providers.isLoading}
          unit="logins"
          emptyLabel={`No IdentityProvider was used in the last ${rangeLabel}.`}
        />

        <Leaderboard
          title="Credentials"
          icon={KeyRound}
          items={fromResources(
            (credentials.data?.items ?? []).map((item: any) => ({
              resource: item.credential,
              count: item.count,
            })),
          )}
          totalCount={n(credentials.data?.totalCount)}
          totalOther={n(credentials.data?.totalOther)}
          to="/core/credentials"
          isLoading={credentials.isLoading}
          unit="logins"
          emptyLabel={`No Credential was used in the last ${rangeLabel}.`}
        />
      </div>
    </Panel>
  );
};

export default IdentityLeaders;
