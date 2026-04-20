import { ResourceInfoMainItem, ResourceMainInfo } from "@/pages/utils/types";
import {
  hasAccessLog,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
} from "@/utils/pb";
import { EyeOff, ShieldAlert, Tag } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import ResourceYAML from "../ResourceYAML";
import TimeAgo from "../TimeAgo";

import { ResourceVisibilityButtons } from "./ResourceInfo";

const CompactInfoCell = ({
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
      "flex flex-col gap-0.5",
      span === "full" ? "col-span-2" : "col-span-1",
    )}
  >
    <span className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400 leading-none">
      {label}
    </span>
    <div className="text-[0.78rem] font-semibold text-slate-700 leading-snug">
      {children}
    </div>
  </div>
);

const SectionDivider = ({ label }: { label: string }) => (
  <div className="col-span-2 flex items-center gap-2 pt-1">
    <span className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400 shrink-0">
      {label}
    </span>
    <div className="flex-1 h-px bg-slate-200" />
  </div>
);

export const ResourceListItemDetails = (props: {
  item: Resource;
  expanded: boolean;
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
}) => {
  const { item, expanded } = props;
  const md = item.metadata!;

  const sharedItems: ResourceInfoMainItem[] = [
    {
      label: "Name",
      value: (
        <span className="font-mono text-[0.72rem]">
          <CopyText value={md.name} />
        </span>
      ),
    },
    {
      label: "UID",
      value: (
        <span className="font-mono text-[0.65rem] text-slate-500 break-all">
          <CopyText value={md.uid} />
        </span>
      ),
    },
    ...(md.displayName
      ? [{ label: "Display name", value: <span>{md.displayName}</span> }]
      : []),
    ...(md.createdAt
      ? [{ label: "Created", value: <TimeAgo rfc3339={md.createdAt} /> }]
      : []),
    ...(md.updatedAt
      ? [{ label: "Updated", value: <TimeAgo rfc3339={md.updatedAt} /> }]
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

  const specificItems: ResourceInfoMainItem[] =
    props.mainItemsGetter?.({ item })?.items ?? [];

  const hasVisibility =
    hasAccessLog(item) ||
    hasAuthenticationLog(item) ||
    hasAuditLog(item) ||
    hasSSHSessionLog(item);

  return (
    <div className="border-t border-slate-100 px-5 py-4 flex flex-col gap-4">
      {(md.isSystem || md.isUserHidden) && (
        <div className="flex items-center gap-1.5">
          {md.isSystem && (
            <span className="inline-flex items-center gap-1 px-1.5 py-px text-[0.6rem] font-bold rounded border border-blue-200 text-blue-600 bg-blue-50">
              <ShieldAlert size={8} strokeWidth={2.5} />
              System
            </span>
          )}
          {md.isUserHidden && (
            <span className="inline-flex items-center gap-1 px-1.5 py-px text-[0.6rem] font-bold rounded border border-slate-200 text-slate-500 bg-white">
              <EyeOff size={8} strokeWidth={2.5} />
              Hidden
            </span>
          )}
        </div>
      )}

      <div className="grid grid-cols-2 gap-x-6 gap-y-3">
        {sharedItems.map((x, i) => (
          <CompactInfoCell key={i} label={x.label} span={x.span}>
            {x.value}
          </CompactInfoCell>
        ))}

        {specificItems.length > 0 && (
          <SectionDivider label={`${item.kind} details`} />
        )}

        {specificItems.map((x, i) => (
          <CompactInfoCell key={i} label={x.label} span={x.span}>
            {x.value}
          </CompactInfoCell>
        ))}
      </div>

      {md.tags && md.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 pt-1 border-t border-slate-100">
          {md.tags.map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-1 h-[20px] px-1.5 rounded text-[0.65rem] font-bold bg-slate-800 text-slate-100 border border-slate-700"
            >
              <Tag size={8} strokeWidth={2.5} />
              {tag}
            </span>
          ))}
        </div>
      )}

      <div className="flex items-center justify-between pt-1 border-t border-slate-100">
        {hasVisibility && <ResourceVisibilityButtons item={item} compact />}
        <div className="ml-auto">
          <ResourceYAML item={item} size="xs" />
        </div>
      </div>
    </div>
  );
};

export default ResourceListItemDetails;
