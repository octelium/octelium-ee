import { Metadata } from "@/apis/metav1/metav1";
import {
  ResourceInfoComponent,
  ResourceInfoMainItem,
  ResourceMainInfo,
} from "@/pages/utils/types";
import { Resource } from "@/utils/pb";
import { EyeOff, ShieldAlert, Tag } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import DeleteResource from "../DeleteResource";
import PageWrap from "../PageWrap";
import { ResourceListLabel } from "../ResourceList";
import ResourceYAML from "../ResourceYAML";
import ResourceSchema from "../ResourceSchema";
import TimeAgo from "../TimeAgo";
import CloneResource from "./CloneResource";
import ResourceStatus from "./ResourceStatus";
import { useContextResource } from "./utils";

const METADATA_GROUP = "Metadata";

const isEmptyValue = (value: React.ReactNode) =>
  value === null ||
  value === undefined ||
  value === false ||
  value === "" ||
  (Array.isArray(value) && value.length === 0);

const InfoCell = ({
  label,
  hint,
  children,
  className,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "@container flex min-w-0 flex-col gap-1 px-4 py-3 @sm:flex-row @sm:gap-4",
      className,
    )}
  >
    <div className="flex shrink-0 flex-col @sm:w-24 @lg:w-32">
      <span className="text-xs font-normal leading-5 text-slate-500">
        {label}
      </span>
      {hint && (
        <span className="text-xs font-normal leading-4 text-slate-500">
          {hint}
        </span>
      )}
    </div>
    <div className="min-w-0 flex-1 text-sm font-normal leading-6 text-slate-800">
      {children}
    </div>
  </div>
);

const SectionHeading = ({ title }: { title: string }) => (
  <div className="flex items-center gap-3 border-y border-slate-200 bg-slate-50/70 px-4 py-2">
    <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
      {title}
    </span>
    <div className="h-px flex-1 bg-slate-200" />
  </div>
);

const InfoGrid = ({ items }: { items: ResourceInfoMainItem[] }) => {
  let column = 0;
  let row = 0;

  const cells = items.map((item) => {
    const full = item.span === "full";
    if (full && column === 1) {
      row += 1;
      column = 0;
    }
    const cell = { item, full, isLeft: !full && column === 0, row };
    if (full || column === 1) {
      row += 1;
      column = 0;
    } else {
      column = 1;
    }
    return cell;
  });

  return (
    <div className="grid grid-cols-1 @2xl:grid-cols-2">
      {cells.map(({ item, full, isLeft, row: cellRow }, index) => (
        <InfoCell
          key={item.label}
          label={item.label}
          hint={item.hint}
          className={twMerge(
            index > 0 && "border-t border-slate-100",
            full && "@2xl:col-span-2",
            isLeft && "@2xl:border-r @2xl:border-slate-100",
            index > 0 && cellRow === 0 && "@2xl:border-t-0",
          )}
        >
          {item.value}
        </InfoCell>
      ))}
    </div>
  );
};

const PrimaryTiles = ({ items }: { items: ResourceInfoMainItem[] }) => (
  <div className="flex flex-wrap gap-3 border-b border-slate-200 bg-slate-50/40 px-4 py-4">
    {items.map((item) => (
      <div
        key={item.label}
        className="flex min-w-[168px] max-w-[320px] flex-1 flex-col gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
      >
        <span className="text-xs font-semibold uppercase tracking-[0.05em] text-slate-500">
          {item.label}
        </span>
        <div className="min-w-0 text-sm font-medium leading-6 text-slate-900">
          {item.value}
        </div>
      </div>
    ))}
  </div>
);

const KeyValueChips = ({
  entries,
  emptyLabel,
}: {
  entries: [string, string][];
  emptyLabel: string;
}) => {
  const [expanded, setExpanded] = React.useState(false);
  const limit = 8;
  const visible = expanded ? entries : entries.slice(0, limit);
  const hidden = entries.length - visible.length;

  if (entries.length === 0)
    return <span className="text-slate-500">{emptyLabel}</span>;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {visible.map(([key, value]) => (
        <span
          key={key}
          className="inline-flex max-w-full items-center gap-1 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs leading-none"
        >
          <span className="shrink-0 font-semibold text-slate-600">{key}</span>
          {value !== "" && (
            <>
              <span aria-hidden="true" className="h-3 w-px bg-slate-200" />
              <span className="truncate font-normal text-slate-700">
                {value}
              </span>
            </>
          )}
        </span>
      ))}
      {hidden > 0 && (
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="rounded-md px-1.5 py-1 text-xs font-normal text-slate-500 underline underline-offset-2 hover:text-slate-800"
        >
          +{hidden} more
        </button>
      )}
    </div>
  );
};

const buildMetadataItems = (md: Metadata): ResourceInfoMainItem[] => {
  const labels = Object.entries(md.labels ?? {});
  const annotations = Object.entries(md.annotations ?? {});

  return [
    {
      label: "UID",
      value: (
        <span className="break-all font-mono text-micro font-semibold text-slate-600">
          <CopyText value={md.uid} />
        </span>
      ),
      group: METADATA_GROUP,
    },
    ...(md.createdAt
      ? [
          {
            label: "Created",
            value: <TimeAgo rfc3339={md.createdAt} />,
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(md.updatedAt
      ? [
          {
            label: "Updated",
            value: <TimeAgo rfc3339={md.updatedAt} />,
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(md.actorRef?.kind
      ? [
          {
            label: "Last change",
            value: (
              <span className="flex flex-wrap items-center gap-1.5">
                <ResourceListLabel itemRef={md.actorRef} />
                {md.actorOperation && (
                  <span className="text-xs text-slate-500">
                    {md.actorOperation.toLowerCase()}
                  </span>
                )}
              </span>
            ),
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(md.resourceVersion
      ? [
          {
            label: "Resource version",
            value: (
              <span className="break-all font-mono text-micro font-semibold text-slate-600">
                <CopyText value={md.resourceVersion} />
              </span>
            ),
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(md.description
      ? [
          {
            label: "Description",
            value: <span className="text-slate-700">{md.description}</span>,
            span: "full" as const,
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(md.tags?.length
      ? [
          {
            label: "Tags",
            value: (
              <div className="flex flex-wrap gap-1.5">
                {md.tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-medium leading-none text-slate-700"
                  >
                    <Tag size={10} strokeWidth={2.25} />
                    {tag}
                  </span>
                ))}
              </div>
            ),
            span: "full" as const,
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(labels.length
      ? [
          {
            label: "Labels",
            hint: "Used by selectors and Policies",
            value: <KeyValueChips entries={labels} emptyLabel="None" />,
            span: "full" as const,
            group: METADATA_GROUP,
          },
        ]
      : []),
    ...(annotations.length
      ? [
          {
            label: "Annotations",
            value: <KeyValueChips entries={annotations} emptyLabel="None" />,
            span: "full" as const,
            group: METADATA_GROUP,
          },
        ]
      : []),
  ];
};

export const ResourceNotFound = (props: { parentPath: string }) => {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-[55vh] w-full flex-col items-center justify-center gap-3 text-center">
      <p className="text-lg font-bold text-slate-800">
        This resource does not exist.
      </p>
      <p className="text-sm font-normal text-slate-600" role="status">
        It may have been deleted, renamed, or you may no longer have access.
      </p>
      <button
        type="button"
        className="text-sm font-semibold text-slate-700 underline underline-offset-4 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
        onClick={() => navigate(props.parentPath, { replace: true })}
      >
        Return to the resource list
      </button>
    </div>
  );
};

export const ResourceLoadError = (props: {
  error: unknown;
  retry: () => void;
}) => (
  <div className="flex min-h-[55vh] w-full flex-col items-center justify-center gap-3 text-center">
    <p className="text-lg font-bold text-slate-800">
      This resource could not be loaded.
    </p>
    <p className="max-w-xl text-sm font-normal text-red-600" role="alert">
      {props.error instanceof Error
        ? props.error.message
        : "An unexpected error occurred."}
    </p>
    <button
      type="button"
      className="text-sm font-semibold text-slate-700 underline underline-offset-4 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
      onClick={props.retry}
    >
      Try again
    </button>
  </div>
);

export const ResourceOverviewSkeleton = () => (
  <div className="w-full animate-pulse overflow-hidden rounded-xl border border-slate-200 bg-white">
    <div className="flex items-center gap-3 border-b border-slate-200 bg-slate-50/60 px-4 py-4">
      <div className="h-10 w-10 shrink-0 rounded-lg bg-slate-200" />
      <div className="flex flex-1 flex-col gap-2">
        <div className="h-4 w-48 rounded bg-slate-200" />
        <div className="h-3 w-32 rounded bg-slate-100" />
      </div>
      <div className="h-7 w-24 rounded-md bg-slate-100" />
    </div>
    <div className="grid grid-cols-1 sm:grid-cols-2">
      {Array.from({ length: 6 }).map((_, index) => (
        <div
          key={index}
          className="flex gap-4 border-t border-slate-100 px-4 py-3"
        >
          <div className="h-3 w-24 shrink-0 rounded bg-slate-100" />
          <div className="h-3 w-full max-w-56 rounded bg-slate-100" />
        </div>
      ))}
    </div>
  </div>
);

const ResourceItemMainPage = (props: {
  infoComponent?: ResourceInfoComponent;
  mainAction?: React.ComponentType<{ item: Resource }>;
  unDeletable?: boolean;
  cloneable?: boolean;
}) => {
  const ctx = useContextResource();
  const location = useLocation();

  const parentPath =
    location.pathname.replace(/\/+$/, "").split("/").slice(0, -1).join("/") ||
    "/";

  if (!ctx) return null;
  if (ctx.isError) {
    return (ctx.error as { code?: string })?.code === "NOT_FOUND" ? (
      <ResourceNotFound parentPath={parentPath} />
    ) : (
      <ResourceLoadError error={ctx.error} retry={() => ctx.refetch()} />
    );
  }

  return (
    <PageWrap qry={ctx} skeleton={<ResourceOverviewSkeleton />}>
      {ctx.data && (
        <ResourceMainContent
          resource={ctx.data}
          infoComponent={props.infoComponent}
          mainAction={props.mainAction}
          unDeletable={props.unDeletable}
          cloneable={props.cloneable}
        />
      )}
    </PageWrap>
  );
};

const ResourceMainContent = (props: {
  resource: Resource;
  infoComponent?: ResourceInfoComponent;
  mainAction?: React.ComponentType<{ item: Resource }>;
  unDeletable?: boolean;
  cloneable?: boolean;
}) => {
  const render = (info: ResourceMainInfo) => (
    <ResourceOverview
      resource={props.resource}
      info={info}
      mainAction={props.mainAction}
      unDeletable={props.unDeletable}
      cloneable={props.cloneable}
    />
  );

  const Info = props.infoComponent;

  return Info ? (
    <React.Suspense fallback={<ResourceOverviewSkeleton />}>
      <Info item={props.resource}>{render}</Info>
    </React.Suspense>
  ) : (
    render({})
  );
};

const ResourceOverview = (props: {
  resource: Resource;
  info: ResourceMainInfo;
  mainAction?: React.ComponentType<{ item: Resource }>;
  unDeletable?: boolean;
  cloneable?: boolean;
}) => {
  const { resource: item, info } = props;
  const md = item.metadata!;

  const specificItems = (info.items ?? []).filter(
    (entry) => !isEmptyValue(entry.value),
  );
  const primaryItems = specificItems.filter((entry) => entry.primary);
  const detailItems = specificItems.filter((entry) => !entry.primary);
  const defaultGroup = `${item.kind} details`;

  const groups: { title: string; items: ResourceInfoMainItem[] }[] = [];
  const groupIndex = new Map<string, number>();

  const push = (title: string, entry: ResourceInfoMainItem) => {
    let index = groupIndex.get(title);
    if (index === undefined) {
      index = groups.length;
      groupIndex.set(title, index);
      groups.push({ title, items: [] });
    }
    groups[index].items.push(entry);
  };

  for (const entry of detailItems) push(entry.group ?? defaultGroup, entry);
  for (const entry of buildMetadataItems(md)) push(METADATA_GROUP, entry);

  const order = info.groupOrder ?? [];
  groups.sort((a, b) => {
    if (a.title === METADATA_GROUP) return 1;
    if (b.title === METADATA_GROUP) return -1;
    const ai = order.indexOf(a.title);
    const bi = order.indexOf(b.title);
    if (ai === -1 && bi === -1) return 0;
    if (ai === -1) return 1;
    if (bi === -1) return -1;
    return ai - bi;
  });

  return (
    <div className="@container flex w-full flex-col gap-4">
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
        <header className="flex flex-col gap-4 border-b border-slate-200 bg-slate-50/60 px-4 py-4 @3xl:flex-row @3xl:items-start @3xl:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            {md.picURL && md.picURL.length > 0 && (
              <img
                src={md.picURL}
                className="h-11 w-11 shrink-0 rounded-lg border border-slate-200 object-cover shadow-sm"
                alt=""
                loading="lazy"
              />
            )}

            <div className="flex min-w-0 flex-col gap-1.5">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-xs font-medium text-slate-600">
                  {item.kind}
                </span>
                <h1 className="min-w-0 break-all text-base font-bold leading-6 text-slate-900">
                  {md.name}
                </h1>
                <CopyText value={md.name} hide />
                <ResourceStatus item={item} status={info.status} />
                {md.isSystem && (
                  <span className="inline-flex items-center gap-1 rounded-md border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-xs font-medium text-blue-700">
                    <ShieldAlert size={11} strokeWidth={2.25} />
                    System
                  </span>
                )}
                {md.isUserHidden && (
                  <span className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-600">
                    <EyeOff size={11} strokeWidth={2.25} />
                    Hidden
                  </span>
                )}
              </div>

              {md.displayName && (
                <p className="truncate text-sm font-normal text-slate-600">
                  {md.displayName}
                </p>
              )}
            </div>
          </div>

          <div
            className="flex shrink-0 flex-wrap items-center gap-1.5"
            role="toolbar"
            aria-label={`Actions for ${md.name}`}
          >
            {info.actions}
            {props.mainAction && (
              <React.Suspense fallback={null}>
                <props.mainAction item={item} />
              </React.Suspense>
            )}
            <ResourceYAML item={item} size="xs" />
            <ResourceSchema item={item} />
            {props.cloneable && <CloneResource item={item} />}
            {!props.unDeletable && (
              <DeleteResource
                item={item}
                btnSize="compact-xs"
                btnVariant="outline"
                btnColor="red.7"
                btnLabel="Delete"
              />
            )}
          </div>
        </header>

        {primaryItems.length > 0 && <PrimaryTiles items={primaryItems} />}

        {groups.map((group) => (
          <React.Fragment key={group.title}>
            <SectionHeading title={group.title} />
            <InfoGrid items={group.items} />
          </React.Fragment>
        ))}
      </section>
    </div>
  );
};

export default ResourceItemMainPage;
