import { GitProvider } from "@/apis/cordiumv1/cordiumv1";
import { GetGitProviderSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientVisibilityCordium } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { GitBranch, KeyRound, Layers, UserRound } from "lucide-react";
import {
  getGitProviderConfig,
  getGitProviderTypeLabel,
  spaceRefOf,
  userRefOf,
} from "../utils";

export const LabelComponent = (props: { item: GitProvider }) => {
  const { item } = props;
  const config = getGitProviderConfig(item);
  const userRef = userRefOf(item.status?.userRef);
  const spaceRef = spaceRefOf(item.status?.spaceRef);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">
        {getGitProviderTypeLabel(item)}
      </ResourceListLabel>
      {config?.clientID && (
        <ResourceListLabel label="Client ID">
          {config.clientID}
        </ResourceListLabel>
      )}
      {(config?.scopes.length ?? 0) > 0 && (
        <ResourceListLabel label="Scopes">
          {config!.scopes.length}
        </ResourceListLabel>
      )}
      {spaceRef && <ResourceListLabel itemRef={spaceRef} />}
      {userRef && <ResourceListLabel label="User" itemRef={userRef} />}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetGitProviderSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/gitproviders">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalGithub}
          to="/cordium/gitproviders?type=GITHUB"
          icon={GitBranch}
        >
          GitHub
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalGitlab}
          to="/cordium/gitproviders?type=GITLAB"
          icon={GitBranch}
        >
          GitLab
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOAuth2}
          to="/cordium/gitproviders?type=OAUTH2"
          icon={KeyRound}
        >
          OAuth2
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalSpace} icon={Layers}>
          Spaces
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={UserRound}>
          Users
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "GitProvider"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCordium().getGitProviderSummary({});
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return (
    <div>
      {qry.data.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {qry.data.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
