import { ResourceInfoMainItem, ResourceMainInfo } from "@/pages/utils/types";
import {
  getRefNameQueryArgStr,
  hasAccessLog,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
} from "@/utils/pb";
import {
  EyeOff,
  Library,
  ShieldAlert,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Tag,
} from "lucide-react";
import { Link, Navigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import DeleteResource from "../DeleteResource";
import PageWrap from "../PageWrap";
import ResourceYAML from "../ResourceYAML";
import TimeAgo from "../TimeAgo";
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
      "flex flex-col gap-1 px-5 py-4 bg-white",
      span === "full" ? "col-span-2" : "col-span-1",
    )}
  >
    <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400 leading-none">
      {label}
    </span>
    <div className="text-sm font-semibold text-slate-700 leading-snug">
      {children}
    </div>
  </div>
);

const ResourceVisibilityButtons = ({ item }: { item: Resource }) => {
  const qryNameArg = getRefNameQueryArgStr(item);

  const buttons = [
    {
      show: hasAccessLog(item),
      to: `/visibility/accesslogs?${qryNameArg}`,
      icon: ShieldEllipsis,
      label: "Access logs",
    },
    {
      show: hasAuthenticationLog(item),
      to: `/visibility/authenticationlogs?${qryNameArg}`,
      icon: ShieldUser,
      label: "Auth logs",
    },
    {
      show: hasAuditLog(item),
      to: `/visibility/auditlogs?${qryNameArg}`,
      icon: Library,
      label: "Audit logs",
    },
    {
      show: hasSSHSessionLog(item),
      to: `/visibility/ssh?${qryNameArg}`,
      icon: SquareTerminal,
      label: "SSH sessions",
    },
  ].filter((b) => b.show);

  if (buttons.length === 0) return null;

  return (
    <div className="flex items-center flex-wrap gap-1.5">
      {buttons.map(({ to, icon: Icon, label }) => (
        <Link
          key={to}
          to={to}
          className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[0.72rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150"
        >
          <Icon size={11} strokeWidth={2.5} />
          {label}
        </Link>
      ))}
    </div>
  );
};

const ResourceItemMainPage = (props: {
  mainComponent?: (props: { item: Resource }) => React.ReactNode;
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
}) => {
  const ctx = useContextResource();

  if (ctx?.isError) return <Navigate to="/" replace />;
  if (!ctx) return null;

  return (
    <PageWrap qry={ctx}>
      {ctx.data && (
        <ResourceMainContent
          resource={ctx.data}
          mainItemsGetter={props.mainItemsGetter}
        />
      )}
    </PageWrap>
  );
};

const ResourceMainContent = (props: {
  resource: Resource;
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
}) => {
  const { resource: item } = props;
  const md = item.metadata!;

  const sharedItems: ResourceInfoMainItem[] = [
    {
      label: "Name",
      value: (
        <span className="font-mono text-sm">
          <CopyText value={md.name} />
        </span>
      ),
    },
    {
      label: "UID",
      value: (
        <span className="font-mono text-xs text-slate-500 break-all">
          <CopyText value={md.uid} />
        </span>
      ),
    },
    ...(md.displayName
      ? [
          {
            label: "Display name",
            value: <span className="text-sm">{md.displayName}</span>,
          },
        ]
      : []),
    ...(md.createdAt
      ? [
          {
            label: "Created",
            value: (
              <span className="text-sm">
                <TimeAgo rfc3339={md.createdAt} />
              </span>
            ),
          },
        ]
      : []),
    ...(md.updatedAt
      ? [
          {
            label: "Updated",
            value: (
              <span className="text-sm">
                <TimeAgo rfc3339={md.updatedAt} />
              </span>
            ),
          },
        ]
      : []),
    ...(md.description
      ? [
          {
            label: "Description",
            value: (
              <span className="text-sm text-slate-500">{md.description}</span>
            ),
            span: "full" as const,
          },
        ]
      : []),
  ];

  const specificItems: ResourceInfoMainItem[] =
    props.mainItemsGetter?.({ item })?.items ?? [];

  return (
    <div className="w-full flex flex-col gap-6">
      {md.picURL?.length > 0 && (
        <img
          src={md.picURL}
          className="w-14 h-14 rounded-full border border-slate-200 shadow-sm"
          alt={md.displayName || md.name}
        />
      )}

      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-[0_1px_4px_rgba(15,23,42,0.06)]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-slate-100 bg-slate-50/60">
          <div className="flex items-center gap-2">
            <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-600">
              {item.kind}
            </span>
            {(md.isSystem || md.isUserHidden) && (
              <div className="flex items-center gap-1.5">
                {md.isSystem && (
                  <span className="inline-flex items-center gap-1 px-1.5 py-px text-[0.62rem] font-bold rounded border border-blue-200 text-blue-600 bg-blue-50">
                    <ShieldAlert size={9} strokeWidth={2.5} />
                    System
                  </span>
                )}
                {md.isUserHidden && (
                  <span className="inline-flex items-center gap-1 px-1.5 py-px text-[0.62rem] font-bold rounded border border-slate-200 text-slate-500 bg-slate-50">
                    <EyeOff size={9} strokeWidth={2.5} />
                    Hidden
                  </span>
                )}
              </div>
            )}
          </div>
          <div className="flex items-center">
            <ResourceYAML item={item} size="xs" />
            <div className="ml-2">
              <DeleteResource
                item={item}
                btnSize={`compact-xs`}
                btnVariant="outline"
              />
            </div>
          </div>
        </div>

        {/* Shared metadata grid */}
        <div className="grid grid-cols-2 gap-px bg-slate-100">
          {sharedItems.map((x, i) => (
            <InfoCell key={i} label={x.label} span={x.span}>
              {x.value}
            </InfoCell>
          ))}
        </div>

        {/* Resource-specific items */}
        {specificItems.length > 0 && (
          <>
            <div className="flex items-center gap-3 px-5 py-2 border-y border-slate-100 bg-slate-50/60">
              <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                {item.kind} details
              </span>
              <div className="flex-1 h-px bg-slate-200" />
            </div>
            <div className="grid grid-cols-2 gap-px bg-slate-100">
              {specificItems.map((x, i) => (
                <InfoCell key={i} label={x.label} span={x.span}>
                  {x.value}
                </InfoCell>
              ))}
            </div>
          </>
        )}

        {/* Tags */}
        {md.tags && md.tags.length > 0 && (
          <div className="flex flex-col gap-2 px-5 py-4 border-t border-slate-100">
            <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Tags
            </span>
            <div className="flex flex-wrap gap-1.5">
              {md.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 h-[22px] px-2 rounded text-[0.7rem] font-bold bg-slate-800 text-slate-100 border border-slate-700 shadow-[0_1px_2px_rgba(15,23,42,0.08)]"
                >
                  <Tag size={9} strokeWidth={2.5} />
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Visibility footer */}
        {(hasAccessLog(item) ||
          hasAuthenticationLog(item) ||
          hasAuditLog(item) ||
          hasSSHSessionLog(item)) && (
          <div className="px-5 py-3 border-t border-slate-100 bg-slate-50/60">
            <ResourceVisibilityButtons item={item} />
          </div>
        )}
      </div>
    </div>
  );
};

export default ResourceItemMainPage;
