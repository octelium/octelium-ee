import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import {
  AlertTriangle,
  Boxes,
  Cpu,
  History,
  LayoutTemplate,
  Loader2,
  MapPinned,
  Package,
  Play,
  Share2,
  Square,
  UserRound,
} from "lucide-react";
import {
  HeroStat,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  TonePill,
} from "@/components/ResourceLayout/overview";
import {
  getFailureLabel,
  getImageSource,
  getLimitItems,
  getSharedPortModeLabel,
  getSpaceTypeLabel,
  getStoppingReasonLabel,
  getWorkspaceMeta,
  isWorkspaceStarting,
  regionRefOf,
  sessionRefOf,
  spaceRefOf,
  templateRefOf,
  userRefOf,
} from "../utils";

const WorkspaceHero = (props: { item: CordiumP.Workspace }) => {
  const { item } = props;
  const status = item.status;
  const meta = getWorkspaceMeta(item);
  const starting = isWorkspaceStarting(status?.state);
  const failure = status?.run?.failure;
  const running = status?.state === CordiumP.Workspace_Status_State.RUNNING;

  return (
    <OverviewHero
      tone={meta.tone}
      icon={
        starting ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : failure ? (
          <AlertTriangle size={18} strokeWidth={2.2} />
        ) : running ? (
          <Play size={18} strokeWidth={2.2} />
        ) : (
          <Square size={16} strokeWidth={2.2} />
        )
      }
      title={
        <>
          {meta.label}
          {status?.currentStateSetAt && (
            <span className="font-normal text-slate-500">
              {" "}
              · since <TimeAgo rfc3339={status.currentStateSetAt} />
            </span>
          )}
        </>
      }
      detail={[
        `${getSpaceTypeLabel(status?.spaceType)} Workspace`,
        status?.userRef?.name && `of ${status.userRef.name}`,
        item.spec?.isEphemeral ? "ephemeral storage" : "persistent storage",
        status?.hostname,
      ]
        .filter(Boolean)
        .join(" · ")}
      meta={
        <>
          {status?.isBuild && <TonePill tone="info">Build Workspace</TonePill>}
          {item.spec?.runtime?.autoStop && <TonePill>Auto-stop</TonePill>}
          {getLimitItems(status?.limit).map((limit) => (
            <TonePill key={limit.label}>
              {limit.label} {limit.value}
            </TonePill>
          ))}
        </>
      }
      aside={
        status?.lastActivityAt && (
          <HeroStat
            label="Last activity"
            value={<TimeAgo rfc3339={status.lastActivityAt} />}
          />
        )
      }
    >
      {failure && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title={`${getFailureLabel(failure)} failed.`}>
          {failure.message || "The last run of the Workspace failed."}
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: CordiumP.Workspace;
}): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec;
  const status = item.status;
  const meta = getWorkspaceMeta(item);
  const userRef = userRefOf(status?.userRef);
  const spaceRef = spaceRefOf(status?.spaceRef);
  const templateRef = templateRefOf(status?.templateRef);
  const regionRef = regionRefOf(status?.regionRef);
  const lastRegionRef = regionRefOf(status?.lastRegionRef);
  const sessionRef = sessionRefOf(status?.sessionRef);
  const run = status?.run;
  const image = getImageSource(spec?.image);

  return {
    status: { label: meta.label, tone: meta.tone },
    header: <WorkspaceHero item={item} />,
    groupOrder: [
      { title: "Current run", icon: Play },
      { title: "Lifecycle", icon: History },
      { title: "Compute", icon: Cpu },
      { title: "Image", icon: Package },
      { title: "Sharing", icon: Share2 },
    ],
    items: [
      {
        label: "User",
        primary: true,
        icon: UserRound,
        value: userRef ? (
          <ResourceListLabel itemRef={userRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Space",
        primary: true,
        icon: Boxes,
        value: spaceRef ? (
          <ResourceListLabel itemRef={spaceRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Template",
        primary: true,
        icon: LayoutTemplate,
        value: templateRef ? (
          <ResourceListLabel itemRef={templateRef} />
        ) : (
          <MutedValue>None</MutedValue>
        ),
      },
      {
        label: "Region",
        primary: true,
        icon: MapPinned,
        value: regionRef ? (
          <ResourceListLabel itemRef={regionRef} />
        ) : lastRegionRef ? (
          <ResourceListLabel itemRef={lastRegionRef} />
        ) : (
          <MutedValue>Not scheduled</MutedValue>
        ),
      },
      ...(status?.hostname
        ? [
            {
              label: "Hostname",
              group: "Current run",
              value: <CopyText value={status.hostname} />,
            },
          ]
        : []),
      ...(sessionRef
        ? [
            {
              label: "Session",
              group: "Current run",
              value: <ResourceListLabel itemRef={sessionRef} />,
              hint: "Created by the Cluster for this run",
            },
          ]
        : []),

      ...(run?.id
        ? [
            {
              label: "Run ID",
              group: "Current run",
              value: <CopyText value={run.id} />,
            },
          ]
        : []),
      ...(run?.initializedAt
        ? [
            {
              label: "Initialized",
              group: "Current run",
              value: <TimeAgo rfc3339={run.initializedAt} />,
            },
          ]
        : []),
      ...(run?.stoppedAt
        ? [
            {
              label: "Stopped",
              group: "Current run",
              value: <TimeAgo rfc3339={run.stoppedAt} />,
            },
          ]
        : []),

      ...(status?.currentStateSetAt
        ? [
            {
              label: "State since",
              group: "Lifecycle",
              value: <TimeAgo rfc3339={status.currentStateSetAt} />,
            },
          ]
        : []),
      ...(status?.lastActivityAt
        ? [
            {
              label: "Last activity",
              group: "Lifecycle",
              value: <TimeAgo rfc3339={status.lastActivityAt} />,
            },
          ]
        : []),
      ...(status?.lastRunningAt
        ? [
            {
              label: "Last running",
              group: "Lifecycle",
              value: <TimeAgo rfc3339={status.lastRunningAt} />,
            },
          ]
        : []),
      ...(status?.lastStoppedAt
        ? [
            {
              label: "Last stopped",
              group: "Lifecycle",
              value: <TimeAgo rfc3339={status.lastStoppedAt} />,
            },
          ]
        : []),
      {
        label: "Successful runs",
        group: "Lifecycle",
        value: (
          <span className="font-semibold text-slate-700">
            {status?.successfulRuns ?? 0}
          </span>
        ),
      },
      ...((status?.lastRuns.length ?? 0) > 0
        ? [
            {
              label: "Previous runs",
              group: "Lifecycle",
              value: (
                <span className="font-semibold text-slate-700">
                  {status!.lastRuns.length}
                  <span className="ml-1.5 text-xs font-normal text-slate-500">
                    {
                      status!.lastRuns.filter((lastRun) => !!lastRun.failure)
                        .length
                    }{" "}
                    failed
                  </span>
                </span>
              ),
            },
          ]
        : []),
      ...(status?.stoppingReason
        ? [
            {
              label: "Stopping reason",
              group: "Lifecycle",
              value: (
                <Label>{getStoppingReasonLabel(status.stoppingReason)}</Label>
              ),
            },
          ]
        : []),
      ...(status?.lastStoppingReason
        ? [
            {
              label: "Last stopping reason",
              group: "Lifecycle",
              value: (
                <Label outlined>
                  {getStoppingReasonLabel(status.lastStoppingReason)}
                </Label>
              ),
            },
          ]
        : []),

      ...getLimitItems(status?.limit).map((limit) => ({
        label: limit.label,
        group: "Compute",
        value: (
          <span className="font-semibold text-slate-700">{limit.value}</span>
        ),
      })),

      ...(image
        ? [
            {
              label: "Image source",
              group: "Image",
              value: <Label>{image.label}</Label>,
            },
          ]
        : []),
      ...(image?.value
        ? [
            {
              label: "Image",
              group: "Image",
              span: "full" as const,
              value: <CopyText value={image.value} />,
            },
          ]
        : []),
      ...(spec?.repository?.url
        ? [
            {
              label: "Repository",
              group: "Image",
              span: "full" as const,
              value: <CopyText value={spec.repository.url} />,
            },
          ]
        : []),
      ...((spec?.additionalRepositories.length ?? 0) > 0
        ? [
            {
              label: "Additional repositories",
              group: "Image",
              value: (
                <span className="font-semibold text-slate-700">
                  {spec!.additionalRepositories.length}
                </span>
              ),
            },
          ]
        : []),

      ...((spec?.applications.length ?? 0) > 0
        ? [
            {
              label: "Applications",
              group: "Sharing",
              value: (
                <div className="flex flex-wrap gap-1.5">
                  {spec!.applications.map((app) => (
                    <Label key={app.name} size="sm" outlined>
                      {app.displayName || app.name}
                      <span className="font-normal text-slate-500">
                        :{app.port}
                      </span>
                    </Label>
                  ))}
                </div>
              ),
            },
          ]
        : []),
      ...((status?.sharedPorts.length ?? 0) > 0
        ? [
            {
              label: "Shared ports",
              group: "Sharing",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1.5">
                  {status!.sharedPorts.map((port) => (
                    <ResourceListLabel
                      key={port.applicationName}
                      label={port.applicationName}
                    >
                      {getSharedPortModeLabel(port.mode)}
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),
    ],
  };
};
