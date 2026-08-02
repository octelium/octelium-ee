import { ResourceInfoMainItem, ResourceMainInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";
import { EyeOff, ShieldAlert, Tag } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import DeleteResource from "../DeleteResource";
import PageWrap from "../PageWrap";
import ResourceYAML from "../ResourceYAML";
import TimeAgo from "../TimeAgo";
import CloneResource from "./CloneResource";
import ResourceInfoItems from "./ResourceInfoItems";
import { useContextResource } from "./utils";

const InfoCell = ({
  label,
  children,
  span,
}: {
  label: string;
  children: React.ReactNode;
  span?: "full" | "half";
}) => (
  <div
    className={twMerge(
      "flex min-w-0 items-start gap-3 bg-white px-3 py-2.5 sm:px-4",
      span === "full" ? "sm:col-span-2" : "col-span-1",
    )}
  >
    <span className="w-24 shrink-0 pt-0.5 text-[0.61rem] font-bold uppercase leading-4 tracking-[0.06em] text-slate-500">
      {label}
    </span>
    <div className="min-w-0 flex-1 text-[0.76rem] font-semibold leading-5 text-slate-700">
      {children}
    </div>
  </div>
);

const ResourceNotFound = (props: { parentPath: string }) => {
  const navigate = useNavigate();
  const [seconds, setSeconds] = React.useState(5);

  React.useEffect(() => {
    const interval = window.setInterval(
      () => setSeconds((value) => Math.max(0, value - 1)),
      1000,
    );
    return () => window.clearInterval(interval);
  }, []);

  React.useEffect(() => {
    if (seconds === 0) navigate(props.parentPath, { replace: true });
  }, [navigate, props.parentPath, seconds]);

  return (
    <div className="flex min-h-[55vh] w-full flex-col items-center justify-center gap-3 text-center">
      <p className="text-lg font-bold text-slate-800">
        This resource does not exist.
      </p>
      <p className="text-sm font-semibold text-slate-500" role="status">
        Returning to the resource list in {seconds} second
        {seconds === 1 ? "" : "s"}.
      </p>
      <button
        type="button"
        className="text-sm font-bold text-slate-700 underline underline-offset-4 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
        onClick={() => navigate(props.parentPath, { replace: true })}
      >
        Return now
      </button>
    </div>
  );
};

const ResourceLoadError = (props: { error: unknown; retry: () => void }) => (
  <div className="flex min-h-[55vh] w-full flex-col items-center justify-center gap-3 text-center">
    <p className="text-lg font-bold text-slate-800">
      This resource could not be loaded.
    </p>
    <p className="max-w-xl text-sm font-semibold text-red-600" role="alert">
      {props.error instanceof Error
        ? props.error.message
        : "An unexpected error occurred."}
    </p>
    <button
      type="button"
      className="text-sm font-bold text-slate-700 underline underline-offset-4 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
      onClick={props.retry}
    >
      Try again
    </button>
  </div>
);

const ResourceItemMainPage = (props: {
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
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
    <PageWrap qry={ctx}>
      {ctx.data && (
        <ResourceMainContent
          resource={ctx.data}
          mainItemsGetter={props.mainItemsGetter}
          unDeletable={props.unDeletable}
          cloneable={props.cloneable}
        />
      )}
    </PageWrap>
  );
};

const ResourceMainContent = (props: {
  resource: Resource;
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
  unDeletable?: boolean;
  cloneable?: boolean;
}) => {
  const { resource: item } = props;
  const md = item.metadata!;

  const sharedItems: ResourceInfoMainItem[] = [
    {
      label: "UID",
      value: (
        <span className="break-all text-[0.72rem] text-slate-500">
          <CopyText value={md.uid} />
        </span>
      ),
    },
    ...(md.createdAt
      ? [
          {
            label: "Created",
            value: <TimeAgo rfc3339={md.createdAt} />,
          },
        ]
      : []),
    ...(md.updatedAt
      ? [
          {
            label: "Updated",
            value: <TimeAgo rfc3339={md.updatedAt} />,
          },
        ]
      : []),
    ...(md.description
      ? [
          {
            label: "Description",
            value: <span className="text-slate-500">{md.description}</span>,
            span: "full" as const,
          },
        ]
      : []),
  ];

  return (
    <div className="flex w-full flex-col gap-4">
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_4px_rgba(15,23,42,0.05)]">
        <header className="flex flex-col gap-4 border-b border-slate-200 bg-slate-50/60 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            {md.picURL?.length > 0 && (
              <img
                src={md.picURL}
                className="h-11 w-11 shrink-0 rounded-lg border border-slate-200 object-cover shadow-sm"
                alt={md.displayName || md.name}
                loading="lazy"
              />
            )}

            <div className="flex min-w-0 flex-col gap-1">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="truncate text-sm font-bold text-slate-800">
                  <CopyText value={md.name} />
                </span>
                <span className="rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[0.61rem] font-bold uppercase tracking-[0.05em] text-slate-500">
                  {item.kind}
                </span>
                {md.isSystem && (
                  <span className="inline-flex items-center gap-1 rounded-md border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-[0.61rem] font-bold text-blue-700">
                    <ShieldAlert size={10} strokeWidth={2.5} />
                    System
                  </span>
                )}
                {md.isUserHidden && (
                  <span className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-[0.61rem] font-bold text-slate-600">
                    <EyeOff size={10} strokeWidth={2.5} />
                    Hidden
                  </span>
                )}
              </div>
              {md.displayName && (
                <span className="truncate text-[0.76rem] font-medium text-slate-500">
                  {md.displayName}
                </span>
              )}
            </div>
          </div>

          <div
            className="flex shrink-0 flex-wrap items-center gap-1.5"
            role="toolbar"
            aria-label={`Actions for ${md.name}`}
          >
            <ResourceYAML item={item} size="xs" />
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

        <div className="grid grid-cols-1 gap-px bg-slate-200/70 sm:grid-cols-2">
          {sharedItems.map((x) => (
            <InfoCell key={x.label} label={x.label} span={x.span}>
              {x.value}
            </InfoCell>
          ))}
        </div>

        {props.mainItemsGetter && (
          <ResourceInfoItems getter={props.mainItemsGetter} item={item}>
            {(specificItems) =>
              specificItems.length > 0 ? (
                <>
                  <div className="flex items-center gap-3 border-y border-slate-200 bg-slate-50/70 px-4 py-2">
                    <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-500">
                      {item.kind} details
                    </span>
                    <div className="h-px flex-1 bg-slate-200" />
                  </div>
                  <div className="grid grid-cols-1 gap-px bg-slate-200/70 sm:grid-cols-2">
                    {specificItems.map((x) => (
                      <InfoCell key={x.label} label={x.label} span={x.span}>
                        {x.value}
                      </InfoCell>
                    ))}
                  </div>
                </>
              ) : null
            }
          </ResourceInfoItems>
        )}

        {md.tags && md.tags.length > 0 && (
          <div className="flex items-start gap-3 border-t border-slate-200 px-4 py-3">
            <span className="w-24 shrink-0 pt-1 text-[0.61rem] font-bold uppercase tracking-[0.06em] text-slate-500">
              Tags
            </span>
            <div className="flex flex-wrap gap-1.5">
              {md.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex min-h-6 items-center gap-1 rounded-md border border-slate-200 bg-slate-100 px-2 py-1 text-[0.68rem] font-bold text-slate-600"
                >
                  <Tag size={9} strokeWidth={2.5} />
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

      </section>
    </div>
  );
};

export default ResourceItemMainPage;
