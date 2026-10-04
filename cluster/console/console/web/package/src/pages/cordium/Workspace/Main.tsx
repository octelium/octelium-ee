import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { AlertTriangle } from "lucide-react";
import { twMerge } from "tailwind-merge";
import {
  getFailureLabel,
  getImageSource,
  getLimitItems,
  getSharedPortModeLabel,
  getSpaceTypeLabel,
  getStoppingReasonLabel,
  getWorkspaceMeta,
  regionRefOf,
  sessionRefOf,
  spaceRefOf,
  templateRefOf,
  userRefOf,
} from "../utils";

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
    groupOrder: [
      "Workspace details",
      "Current run",
      "Lifecycle",
      "Compute",
      "Image",
      "Sharing",
    ],
    items: [
      {
        label: "State",
        primary: true,
        value: (
          <span className={twMerge("font-semibold", meta.className)}>
            {meta.label}
          </span>
        ),
      },
      {
        label: "Space type",
        primary: true,
        value: <Label>{getSpaceTypeLabel(status?.spaceType)}</Label>,
      },
      ...(userRef
        ? [{ label: "User", value: <ResourceListLabel itemRef={userRef} /> }]
        : []),
      ...(spaceRef
        ? [{ label: "Space", value: <ResourceListLabel itemRef={spaceRef} /> }]
        : []),
      ...(templateRef
        ? [
            {
              label: "Template",
              value: <ResourceListLabel itemRef={templateRef} />,
            },
          ]
        : []),
      ...(regionRef
        ? [
            {
              label: "Region",
              value: <ResourceListLabel itemRef={regionRef} />,
            },
          ]
        : []),
      ...(sessionRef
        ? [
            {
              label: "Session",
              value: <ResourceListLabel itemRef={sessionRef} />,
              hint: "The dedicated Session that the Cluster created for the current run of the Workspace.",
            },
          ]
        : []),
      ...(status?.hostname
        ? [
            {
              label: "Hostname",
              value: <CopyText value={status.hostname} />,
            },
          ]
        : []),
      {
        label: "Mode",
        value: (
          <div className="flex flex-wrap gap-1.5">
            <Label size="sm" outlined>
              {spec?.isEphemeral ? "Ephemeral" : "Persistent"}
            </Label>
            {status?.isBuild && (
              <Label size="sm" tone="info">
                Build
              </Label>
            )}
            {spec?.runtime?.autoStop && (
              <Label size="sm" outlined>
                Auto-stop
              </Label>
            )}
          </div>
        ),
        hint: "Ephemeral Workspaces lose their storage once stopped.",
      },

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
      ...(run?.failure
        ? [
            {
              label: "Failure",
              group: "Current run",
              span: "full" as const,
              value: (
                <div className="flex items-start gap-2 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
                  <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                  <span>
                    {getFailureLabel(run.failure)}
                    {run.failure.message && (
                      <span className="ml-1.5 font-normal">
                        {run.failure.message}
                      </span>
                    )}
                  </span>
                </div>
              ),
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
      ...(lastRegionRef && !regionRef
        ? [
            {
              label: "Last Region",
              group: "Lifecycle",
              value: <ResourceListLabel itemRef={lastRegionRef} />,
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
