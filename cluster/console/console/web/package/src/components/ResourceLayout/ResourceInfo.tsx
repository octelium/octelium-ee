import CopyText from "@/components/CopyText";
import TimeAgo from "@/components/TimeAgo";
import {
  getRefNameQueryArgStr,
  getResourcePath,
  hasAccessLog,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
} from "@/utils/pb";
import {
  ExternalLink,
  EyeOff,
  Library,
  ShieldAlert,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Tag,
} from "lucide-react";
import { Link } from "react-router-dom";
import ResourceYAML from "../ResourceYAML";

const Field = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <div className="flex items-baseline gap-2 min-w-0">
    <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400 shrink-0 w-[80px]">
      {label}
    </span>
    <div className="text-[0.75rem] font-semibold text-slate-700 leading-snug min-w-0 flex-1">
      {children}
    </div>
  </div>
);

const ResourceInfo = (props: { resource: Resource }) => {
  const { resource: item } = props;
  const md = item.metadata!;

  return (
    <div className="flex flex-col gap-3 w-full">
      {md.picURL?.length > 0 && (
        <img
          src={md.picURL}
          className="w-12 h-12 rounded-full border border-slate-200 shadow-sm"
          alt={md.displayName || md.name}
        />
      )}

      <div className="flex flex-col gap-2">
        <Field label="Name">
          <span className="flex items-center gap-1.5 font-mono text-[0.72rem]">
            <CopyText value={md.name} />
          </span>
        </Field>

        {md.displayName && (
          <Field label="Display name">
            <span className="text-slate-600">{md.displayName}</span>
          </Field>
        )}

        <Field label="UID">
          <span className="font-mono text-[0.68rem] text-slate-500 break-all">
            <CopyText value={md.uid} />
          </span>
        </Field>

        {md.description && (
          <Field label="Description">
            <span className="text-slate-500 text-[0.72rem] leading-relaxed">
              {md.description}
            </span>
          </Field>
        )}

        <Field label="Created">
          <TimeAgo rfc3339={md.createdAt} />
        </Field>

        {md.updatedAt && (
          <Field label="Updated">
            <TimeAgo rfc3339={md.updatedAt} />
          </Field>
        )}
      </div>

      {md.tags && md.tags.length > 0 && (
        <div className="flex flex-col gap-1">
          <span className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
            Tags
          </span>
          <div className="flex flex-wrap gap-1">
            {md.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 h-[20px] px-1.5 rounded text-[0.65rem] font-bold bg-slate-800 text-slate-100 border border-slate-700"
              >
                <Tag size={9} strokeWidth={2.5} />
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

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

      <div className="flex items-center justify-between pt-2 border-t border-slate-100">
        <ResourceVisibilityButtons item={item} compact />
        <div className="flex items-center gap-1.5 ml-auto">
          <ResourceYAML item={item} size="xs" />
          <Link
            to={getResourcePath(item)}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[0.68rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150"
          >
            <ExternalLink size={10} strokeWidth={2.5} />
            Details
          </Link>
        </div>
      </div>
    </div>
  );
};

export default ResourceInfo;

export const ResourceVisibilityButtons = (props: {
  item: Resource;
  compact?: boolean;
}) => {
  const { item, compact } = props;
  const qryNameArg = getRefNameQueryArgStr(item);

  const btnClass =
    "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[0.65rem] font-bold border transition-colors duration-150 text-slate-500 border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50";

  const buttons = [
    {
      show: hasAccessLog(item),
      to: `/visibility/accesslogs?${qryNameArg}`,
      icon: ShieldEllipsis,
      label: compact ? "Access" : "Access Logs",
    },
    {
      show: hasAuthenticationLog(item),
      to: `/visibility/authenticationlogs?${qryNameArg}`,
      icon: ShieldUser,
      label: compact ? "Auth" : "Authentication Logs",
    },
    {
      show: hasAuditLog(item),
      to: `/visibility/auditlogs?${qryNameArg}`,
      icon: Library,
      label: compact ? "Audit" : "Audit Logs",
    },
    {
      show: hasSSHSessionLog(item),
      to: `/visibility/ssh?${qryNameArg}`,
      icon: SquareTerminal,
      label: compact ? "SSH" : "SSH Sessions",
    },
  ].filter((b) => b.show);

  if (buttons.length === 0) return null;

  return (
    <div className="flex items-center flex-wrap gap-1">
      {buttons.map(({ to, icon: Icon, label }) => (
        <Link key={to} to={to} className={btnClass}>
          <Icon size={10} strokeWidth={2.5} />
          {label}
        </Link>
      ))}
    </div>
  );
};
