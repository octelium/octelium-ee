import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Building2,
  GitBranch,
  KeyRound,
  KeySquare,
  Layers,
  ShieldOff,
  UserRound,
  UsersRound,
} from "lucide-react";
import { useCordiumCreated, useCordiumTotals } from "./queries";

const Collaboration = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.normal);
  const created = useCordiumCreated(periodMinutes, QUERY_PRIORITY.low);

  const cordium = totals.data?.cordium;
  const fresh = created.data?.cordium;
  const spaces = cordium?.space;
  const memberships = cordium?.membership;
  const gitProviders = cordium?.gitProvider;
  const workspaces = cordium?.workspace;

  return (
    <Panel
      icon={Layers}
      title="Spaces & collaboration"
      description={`Who owns the Spaces, who collaborates in them and what they share · deltas cover the last ${rangeLabel}`}
      to="/cordium/spaces"
      toLabel="All Spaces"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Spaces"
            value={n(spaces?.totalNumber)}
            icon={Layers}
            to="/cordium/spaces"
          />
          <MiniStat
            label="Organization"
            value={n(spaces?.totalOrganizationSpace)}
            icon={Building2}
            to="/cordium/spaces?type=ORGANIZATION"
          />
          <MiniStat
            label="Memberships"
            value={n(memberships?.totalNumber)}
            icon={UsersRound}
            to="/cordium/memberships"
          />
          <MiniStat
            label="Members"
            value={n(memberships?.totalUser)}
            icon={UserRound}
            to="/cordium/memberships"
          />
          <MiniStat
            label="SSH disabled"
            value={n(spaces?.totalSSHDisabled)}
            icon={ShieldOff}
            to="/cordium/spaces?isSSHDisabled=true"
          />
          <MiniStat
            label="Git providers"
            value={n(gitProviders?.totalNumber)}
            icon={GitBranch}
            to="/cordium/gitproviders"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Breakdown
            title="Membership roles"
            note={`${compact(n(memberships?.totalSpace))} Spaces`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Owner",
                  value: n(memberships?.totalRoleOwner),
                  color: seriesColor(0),
                },
                {
                  label: "Admin",
                  value: n(memberships?.totalRoleAdmin),
                  color: seriesColor(1),
                },
                {
                  label: "User",
                  value: n(memberships?.totalRoleUser),
                  color: seriesColor(2),
                },
              ]}
              total={n(memberships?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Shared ports"
            note={`${compact(n(workspaces?.totalShared))} Workspaces`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Members",
                  value: n(workspaces?.totalSharedMembers),
                  color: seriesColor(3),
                },
                {
                  label: "Everyone",
                  value: n(workspaces?.totalSharedAll),
                  color: seriesColor(4),
                },
              ]}
              total={
                n(workspaces?.totalSharedMembers) +
                n(workspaces?.totalSharedAll)
              }
            />
          </Breakdown>

          <Breakdown title="Git providers">
            <CompositionBar
              segments={[
                {
                  label: "GitHub",
                  value: n(gitProviders?.totalGithub),
                  color: seriesColor(0),
                },
                {
                  label: "GitLab",
                  value: n(gitProviders?.totalGitlab),
                  color: seriesColor(1),
                },
                {
                  label: "OAuth2",
                  value: n(gitProviders?.totalOAuth2),
                  color: seriesColor(2),
                },
              ]}
              total={n(gitProviders?.totalNumber)}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Spaces · ${rangeLabel}`}
            value={n(fresh?.space?.totalNumber)}
            prev={n(fresh?.space?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Layers}
            to="/cordium/spaces"
          />
          <DeltaStat
            label="Memberships"
            value={n(fresh?.membership?.totalNumber)}
            prev={n(fresh?.membership?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={UsersRound}
            to="/cordium/memberships"
          />
          <DeltaStat
            label="Git providers"
            value={n(fresh?.gitProvider?.totalNumber)}
            prev={n(fresh?.gitProvider?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={GitBranch}
            to="/cordium/gitproviders"
          />
          <DeltaStat
            label="Secrets"
            value={n(fresh?.secret?.totalNumber)}
            prev={n(fresh?.secret?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={KeyRound}
            to="/cordium/secrets"
          />
          <DeltaStat
            label="User secrets"
            value={n(fresh?.userSecret?.totalNumber)}
            prev={n(fresh?.userSecret?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={KeySquare}
            to="/cordium/usersecrets"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Collaboration;
