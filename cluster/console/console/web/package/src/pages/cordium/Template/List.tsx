import { Template } from "@/apis/cordiumv1/cordiumv1";
import { GetTemplateSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import TimeAgo from "@/components/TimeAgo";
import { getClientVisibilityCordium } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  CheckCircle2,
  CircleDashed,
  GitBranch,
  Hammer,
  Layers,
  UserRound,
} from "lucide-react";
import {
  getImageSource,
  getLatestBuild,
  getTemplateMeta,
  gitProviderRefOf,
  spaceRefOf,
  userRefOf,
} from "../utils";

export const LabelComponent = (props: { item: Template }) => {
  const { item } = props;
  const status = item.status;
  const meta = getTemplateMeta(item);
  const latest = getLatestBuild(item);
  const userRef = userRefOf(status?.userRef);
  const spaceRef = spaceRefOf(status?.spaceRef);
  const gitProviderRef = gitProviderRefOf(status?.gitProviderRef);
  const image = getImageSource(item.spec?.image);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      {image && (
        <ResourceListLabel label="Image">{image.label}</ResourceListLabel>
      )}
      {spaceRef && <ResourceListLabel itemRef={spaceRef} />}
      {userRef && <ResourceListLabel label="User" itemRef={userRef} />}
      {gitProviderRef && <ResourceListLabel itemRef={gitProviderRef} />}
      {(status?.buildInfo?.builds.length ?? 0) > 0 && (
        <ResourceListLabel label="Builds">
          {status!.buildInfo!.builds.length}
        </ResourceListLabel>
      )}
      {status?.buildInfo?.currentReadyBuildID && (
        <ResourceListLabel label="Ready build">
          {status.buildInfo.currentReadyBuildID}
        </ResourceListLabel>
      )}
      {latest?.startedAt && (
        <ResourceListLabel label="Last build">
          <TimeAgo rfc3339={latest.startedAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetTemplateSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/templates">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWithReadyBuild}
          to="/cordium/templates?hasReadyBuild=true"
          icon={CheckCircle2}
        >
          Ready
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWithRunningBuild}
          to="/cordium/templates?isBuilding=true"
          icon={Hammer}
        >
          Building
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalNeverBuilt} icon={CircleDashed}>
          Never built
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalBuild} icon={Hammer}>
          Builds
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalBuildFailed} icon={AlertTriangle}>
          Failed builds
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalSpace} icon={Layers}>
          Spaces
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={UserRound}>
          Users
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalGitProvider} icon={GitBranch}>
          Git providers
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "Template"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCordium().getTemplateSummary({});
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
