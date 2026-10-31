import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "@/lib/utils";
import {
  AlertTriangle,
  Boxes,
  Cpu,
  GitBranch,
  Hammer,
  Loader2,
  Package,
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
  getBuildStateMeta,
  getFailureLabel,
  getImageSource,
  getLatestBuild,
  getLimitItems,
  getTemplateMeta,
  gitProviderRefOf,
  spaceRefOf,
  userRefOf,
} from "../utils";

const MAX_BUILDS = 10;

const BuildRow = (props: {
  build: CordiumP.Template_Status_BuildInfo_Build;
}) => {
  const { build } = props;
  const meta = build.isCanceled
    ? { label: "Canceled", className: "text-amber-600" }
    : getBuildStateMeta(build.state);

  return (
    <div className="flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className={twMerge("text-xs font-semibold", meta.className)}>
          {meta.label}
        </span>
        <span className="truncate text-xs font-semibold text-slate-700">
          {build.id}
        </span>
        {build.tags.map((tag) => (
          <Label key={tag} size="sm" outlined>
            {tag}
          </Label>
        ))}
        <span className="ml-auto inline-flex items-center gap-1 text-micro font-normal text-slate-500">
          {build.startedAt && <TimeAgo rfc3339={build.startedAt} />}
          {build.doneAt && (
            <>
              <span>→</span>
              <TimeAgo rfc3339={build.doneAt} />
            </>
          )}
        </span>
      </div>
      {build.failure && (
        <span className="text-micro font-semibold text-red-600">
          {getFailureLabel(build.failure)}
          {build.failure.message && (
            <span className="ml-1.5 font-normal">{build.failure.message}</span>
          )}
        </span>
      )}
    </div>
  );
};

const TemplateHero = (props: { item: CordiumP.Template }) => {
  const { item } = props;
  const meta = getTemplateMeta(item);
  const latest = getLatestBuild(item);
  const buildInfo = item.status?.buildInfo;
  const running = !!buildInfo?.currentRunningBuildID;
  const image = getImageSource(item.spec?.image);

  return (
    <OverviewHero
      tone={running ? "info" : meta.tone}
      icon={
        running ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : latest?.failure ? (
          <AlertTriangle size={18} strokeWidth={2.2} />
        ) : (
          <Hammer size={18} strokeWidth={2.2} />
        )
      }
      title={
        running
          ? "Building"
          : latest
            ? `Latest build ${meta.label.toLowerCase()}`
            : "Never built"
      }
      detail={[
        image && `${image.label} image`,
        image?.value,
        buildInfo?.currentReadyBuildID &&
          `new Workspaces use ${buildInfo.currentReadyBuildID}`,
      ]
        .filter(Boolean)
        .join(" · ")}
      meta={getLimitItems(item.spec?.limit).map((limit) => (
        <TonePill key={limit.label}>
          {limit.label} {limit.value}
        </TonePill>
      ))}
      aside={
        latest?.doneAt && (
          <HeroStat
            label="Last build"
            value={<TimeAgo rfc3339={latest.doneAt} />}
          />
        )
      }
    >
      {latest?.failure && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title={`${getFailureLabel(latest.failure)} failed.`}>
          {latest.failure.message ||
            "The latest build failed; new Workspaces use the last ready build."}
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: CordiumP.Template;
}): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec;
  const status = item.status;
  const buildInfo = status?.buildInfo;
  const meta = getTemplateMeta(item);
  const latest = getLatestBuild(item);
  const userRef = userRefOf(status?.userRef);
  const spaceRef = spaceRefOf(status?.spaceRef);
  const gitProviderRef = gitProviderRefOf(status?.gitProviderRef);
  const image = getImageSource(spec?.image);
  const builds = buildInfo?.builds ?? [];

  return {
    status: { label: meta.label, tone: meta.tone },
    header: <TemplateHero item={item} />,
    groupOrder: [
      { title: "Builds", icon: Hammer },
      { title: "Image", icon: Package },
      { title: "Compute", icon: Cpu },
    ],
    items: [
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
        label: "Created by",
        primary: true,
        icon: UserRound,
        value: userRef ? (
          <ResourceListLabel itemRef={userRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Builds",
        primary: true,
        icon: Hammer,
        value: (
          <span className="font-semibold tabular-nums">{builds.length}</span>
        ),
      },
      {
        label: "Git provider",
        primary: true,
        icon: GitBranch,
        value: gitProviderRef ? (
          <ResourceListLabel itemRef={gitProviderRef} />
        ) : (
          <MutedValue>None</MutedValue>
        ),
      },
      ...(buildInfo?.currentReadyBuildID
        ? [
            {
              label: "Current ready build",
              group: "Builds",
              value: <CopyText value={buildInfo.currentReadyBuildID} />,
              hint: "Used by the new Workspaces of this Template.",
            },
          ]
        : []),
      ...(buildInfo?.currentRunningBuildID
        ? [
            {
              label: "Current running build",
              group: "Builds",
              value: <CopyText value={buildInfo.currentRunningBuildID} />,
            },
          ]
        : []),
      ...(latest?.doneAt
        ? [
            {
              label: "Last build done",
              group: "Builds",
              value: <TimeAgo rfc3339={latest.doneAt} />,
            },
          ]
        : []),
      ...(builds.length > 0
        ? [
            {
              label:
                builds.length > MAX_BUILDS
                  ? `Recent builds (latest ${MAX_BUILDS})`
                  : "Recent builds",
              group: "Builds",
              span: "full" as const,
              value: (
                <div className="flex flex-col gap-1.5">
                  {builds.slice(0, MAX_BUILDS).map((build) => (
                    <BuildRow key={build.id} build={build} />
                  ))}
                </div>
              ),
            },
          ]
        : []),

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
      ...(spec?.gitProvider
        ? [
            {
              label: "Git provider name",
              group: "Image",
              value: <CopyText value={spec.gitProvider} />,
            },
          ]
        : []),

      ...getLimitItems(spec?.limit).map((limit) => ({
        label: limit.label,
        group: "Compute",
        value: (
          <span className="font-semibold text-slate-700">{limit.value}</span>
        ),
      })),
    ],
  };
};
