import { Workspace } from "@/apis/cordiumv1/cordiumv1";
import { GetWorkspaceSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
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
  CirclePause,
  CirclePlay,
  Hammer,
  Hourglass,
  Layers,
  LayoutTemplate,
  Share2,
  Sparkles,
  UserRound,
} from "lucide-react";
import {
  getFailureLabel,
  getWorkspaceMeta,
  regionRefOf,
  spaceRefOf,
  templateRefOf,
  userRefOf,
} from "../utils";

export const LabelComponent = (props: { item: Workspace }) => {
  const { item } = props;
  const status = item.status;
  const meta = getWorkspaceMeta(item);
  const userRef = userRefOf(status?.userRef);
  const spaceRef = spaceRefOf(status?.spaceRef);
  const templateRef = templateRefOf(status?.templateRef);
  const regionRef = regionRefOf(status?.regionRef);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      {status?.run?.failure && (
        <ResourceListLabel label="Failure">
          {getFailureLabel(status.run.failure)}
        </ResourceListLabel>
      )}
      {userRef && <ResourceListLabel label="User" itemRef={userRef} />}
      {spaceRef && <ResourceListLabel itemRef={spaceRef} />}
      {templateRef && <ResourceListLabel itemRef={templateRef} />}
      {regionRef && <ResourceListLabel itemRef={regionRef} />}
      {status?.isBuild && <ResourceListLabel>Build</ResourceListLabel>}
      {item.spec?.isEphemeral && (
        <ResourceListLabel>Ephemeral</ResourceListLabel>
      )}
      {item.spec?.runtime?.autoStop && (
        <ResourceListLabel>Auto-stop</ResourceListLabel>
      )}
      {(status?.sharedPorts.length ?? 0) > 0 && (
        <ResourceListLabel label="Shared ports">
          {status!.sharedPorts.length}
        </ResourceListLabel>
      )}
      {status?.lastActivityAt && (
        <ResourceListLabel label="Last activity">
          <TimeAgo rfc3339={status.lastActivityAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetWorkspaceSummaryResponse }) => {
  const starting =
    resp.totalInitializing +
    resp.totalPullingImage +
    resp.totalBuildingImage +
    resp.totalStartingRuntime +
    resp.totalPreparing;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/workspaces">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRunning}
          to="/cordium/workspaces?state=RUNNING"
          icon={CirclePlay}
        >
          Running
        </SummaryItemCount>
        <SummaryItemCount count={starting} icon={Hourglass}>
          Starting
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalStopping} icon={Hourglass}>
          Stopping
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalStopped}
          to="/cordium/workspaces?state=STOPPED"
          icon={CirclePause}
        >
          Stopped
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalFailed}
          to="/cordium/workspaces?isFailed=true"
          icon={AlertTriangle}
        >
          Failed
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalBuild}
          to="/cordium/workspaces?isBuild=true"
          icon={Hammer}
        >
          Builds
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalEphemeral}
          to="/cordium/workspaces?isEphemeral=true"
          icon={Sparkles}
        >
          Ephemeral
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalShared}
          to="/cordium/workspaces?isShared=true"
          icon={Share2}
        >
          Shared
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={UserRound}>
          Users
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalSpace} icon={Layers}>
          Spaces
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalTemplate} icon={LayoutTemplate}>
          Templates
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "Workspace"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCordium().getWorkspaceSummary({});
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
