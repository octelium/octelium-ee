import CopyText from "@/components/CopyText";
import TimeAgo from "@/components/TimeAgo";
import { useAppSelector } from "@/utils/hooks";
import {
  ArrowUpRight,
  BadgeCheck,
  CircleAlert,
  LaptopMinimal,
  Mail,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import { Link } from "react-router-dom";

const initials = (value: string) =>
  value
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();

const Field = (props: {
  label: string;
  children: React.ReactNode;
  wide?: boolean;
}) => (
  <div className={props.wide ? "sm:col-span-2" : ""}>
    <div className="mb-1 text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {props.label}
    </div>
    <div className="min-w-0 text-sm font-semibold leading-5 text-slate-700">
      {props.children}
    </div>
  </div>
);

const Section = (props: {
  title: string;
  description: string;
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  children: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_4px_rgba(15,23,42,0.05)]">
      <header className="flex items-start gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 shadow-sm">
          <Icon size={16} strokeWidth={2} />
        </span>
        <div className="min-w-0">
          <h2 className="text-sm font-bold text-slate-800">{props.title}</h2>
          <p className="mt-0.5 text-xs font-medium leading-5 text-slate-500">
            {props.description}
          </p>
        </div>
      </header>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 px-4 py-4 sm:grid-cols-2 sm:px-5">
        {props.children}
      </div>
    </section>
  );
};

export default () => {
  const status = useAppSelector((state) => state.settings.status);
  const user = status?.user;
  const metadata = user?.metadata;
  const session = status?.session;
  const displayName = metadata?.displayName || metadata?.name || "User";

  if (!user || !metadata) {
    return (
      <main className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-2 py-2 sm:px-4 sm:py-4">
        <section className="flex min-h-[360px] flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white/70 px-6 text-center">
          <CircleAlert size={22} className="mb-3 text-slate-400" />
          <h1 className="text-base font-bold text-slate-700">
            Account information is not available
          </h1>
          <p className="mt-1 max-w-md text-sm font-medium text-slate-500">
            Your signed-in user information has not finished loading yet.
          </p>
        </section>
      </main>
    );
  }

  const sessionName = session?.metadata?.name;

  return (
    <main className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-2 py-2 sm:px-4 sm:py-4">
      <header className="flex flex-col gap-4 rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-[0_1px_4px_rgba(15,23,42,0.05)] sm:flex-row sm:items-center sm:justify-between sm:px-5">
        <div className="flex min-w-0 items-center gap-3.5">
          <div className="h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-slate-200 bg-slate-700 shadow-sm">
            {metadata.picURL ? (
              <img
                src={metadata.picURL}
                alt={displayName}
                className="h-full w-full object-cover"
              />
            ) : (
              <span className="flex h-full w-full items-center justify-center text-sm font-bold text-white">
                {initials(displayName)}
              </span>
            )}
          </div>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-lg font-bold text-slate-800">
                {displayName}
              </h1>
            </div>
            <div className="mt-1 flex min-w-0 items-center gap-1.5 text-sm font-medium text-slate-500">
              <Mail size={13} className="shrink-0" />
              <span className="truncate">{user.spec?.email || "No email set"}</span>
            </div>
          </div>
        </div>
        <Link
          to={`/core/users/${encodeURIComponent(metadata.name)}`}
          state={{ returnTo: "/settings" }}
          preventScrollReset
          className="inline-flex shrink-0 items-center justify-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-bold text-slate-700 shadow-sm transition-colors duration-500 hover:border-slate-400 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
        >
          <UserRound size={14} strokeWidth={2.25} />
          Open user
          <ArrowUpRight size={13} strokeWidth={2.5} />
        </Link>
      </header>

      <Section
        title="Account and session"
        description="Your read-only identity information and current console session."
        icon={ShieldCheck}
      >
        <Field label="Name">
          <CopyText value={metadata.name} />
        </Field>
        {metadata.displayName && (
          <Field label="Display name">{metadata.displayName}</Field>
        )}
        <Field label="Email">
          {user.spec?.email ? <CopyText value={user.spec.email} /> : "Not set"}
        </Field>
        <Field label="User ID" wide>
          <span className="break-all text-xs text-slate-500">
            <CopyText value={metadata.uid} />
          </span>
        </Field>
        {metadata.description && (
          <Field label="Description" wide>
            {metadata.description}
          </Field>
        )}
        {session && (
          <Field label="Current session" wide>
            {sessionName ? (
              <Link
                to={`/core/sessions/${encodeURIComponent(sessionName)}`}
                state={{ returnTo: "/settings" }}
                preventScrollReset
                className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-bold text-slate-700 shadow-sm transition-colors duration-500 hover:border-slate-400 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
              >
                <LaptopMinimal size={14} strokeWidth={2.25} />
                Open session
                <ArrowUpRight size={13} strokeWidth={2.5} />
              </Link>
            ) : (
              "No named session"
            )}
          </Field>
        )}
      </Section>

      {(metadata.createdAt || metadata.updatedAt || metadata.tags.length > 0) && (
        <Section
          title="Profile metadata"
          description="Additional metadata associated with your account."
          icon={BadgeCheck}
        >
          {metadata.createdAt && (
            <Field label="Created">
              <TimeAgo rfc3339={metadata.createdAt} />
            </Field>
          )}
          {metadata.updatedAt && (
            <Field label="Updated">
              <TimeAgo rfc3339={metadata.updatedAt} />
            </Field>
          )}
          {metadata.tags.length > 0 && (
            <Field label="Tags" wide>
              <div className="flex flex-wrap gap-1.5">
                {metadata.tags.map((tag) => (
                  <span
                    key={tag}
                    className="rounded-md border border-slate-200 bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </Field>
          )}
        </Section>
      )}
    </main>
  );
};
