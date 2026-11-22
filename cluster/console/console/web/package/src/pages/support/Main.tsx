import {
  ListSupportRequestOptions_Filter,
  SupportInfo,
  SupportInfo_State,
  SupportRequest,
} from "@/apis/enterprisev1/enterprisev1";
import { Panel } from "@/components/Dashboard/components";
import Meta from "@/components/Meta";
import { PageActions, PageTitleAddon } from "@/components/PageHeader/context";
import RefreshButton from "@/components/RefreshButton";
import RpcError from "@/components/RpcError";
import TimeAgo from "@/components/TimeAgo";
import { Alert, Button, SegmentedControl, Skeleton } from "@mantine/core";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  ArrowUpRight,
  Building2,
  ChevronRight,
  Flag,
  Globe,
  Inbox,
  LifeBuoy,
  MessagesSquare,
  Paperclip,
  Search,
  Send,
  Stethoscope,
  UserRound,
} from "lucide-react";
import * as React from "react";
import { Link, useSearchParams } from "react-router";
import {
  NewRequestButton,
  PortalLink,
  PriorityBadge,
  StateBadge,
  StateNotice,
  StatusBadge,
} from "./components";
import { supportKeys, useSupportInfo, useSupportRequests } from "./queries";
import { LEVEL_LABEL, supportRequestPath } from "./utils";

const FILTERS = [
  {
    value: "open",
    label: "Open",
    filter: ListSupportRequestOptions_Filter.OPEN,
  },
  {
    value: "closed",
    label: "Resolved",
    filter: ListSupportRequestOptions_Filter.CLOSED,
  },
  { value: "all", label: "All", filter: ListSupportRequestOptions_Filter.ALL },
];

const Fact = (props: {
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  label: string;
  children: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <div className="flex min-w-0 items-start gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
      <span className="mt-0.5 shrink-0 text-slate-400">
        <Icon size={13} strokeWidth={2.3} />
      </span>
      <div className="min-w-0">
        <div className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {props.label}
        </div>
        <div className="truncate text-body font-semibold text-slate-800">
          {props.children}
        </div>
      </div>
    </div>
  );
};

const Overview = (props: { info?: SupportInfo; loading: boolean }) => {
  const { info } = props;

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <div className="flex flex-col gap-4 px-4 py-4 sm:px-5 sm:py-5 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex min-w-0 items-start gap-3.5">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-600">
            <LifeBuoy size={20} strokeWidth={2.1} />
          </span>
          <div className="min-w-0">
            <h2 className="text-base font-bold text-slate-900">
              Octelium Support
            </h2>
            <p className="mt-0.5 max-w-2xl text-xs font-normal leading-5 text-slate-600">
              Report issues to the Octelium engineers and follow their answers
              without leaving the console. Requests opened here also appear in
              the customer portal of your organization.
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-2 border-t border-slate-100 px-4 py-4 sm:grid-cols-2 sm:px-5 xl:grid-cols-4">
        {props.loading ? (
          [0, 1, 2, 3].map((index) => (
            <Skeleton key={index} height={54} radius="md" />
          ))
        ) : (
          <>
            <Fact icon={Building2} label="Organization">
              {info?.organization?.displayName || "—"}
            </Fact>
            <Fact icon={LifeBuoy} label="Support">
              {info?.organization
                ? (LEVEL_LABEL[info.supportLevel] ?? "Unknown")
                : "—"}
            </Fact>
            <Fact icon={Globe} label="Cluster">
              {info?.clusterDomain || "—"}
            </Fact>
            <Fact icon={Flag} label="Installation ID">
              <span className="font-mono text-xs">
                {info?.installationID || "—"}
              </span>
            </Fact>
          </>
        )}
      </div>
    </section>
  );
};

const RequestRow = (props: { request: SupportRequest }) => {
  const { request } = props;

  return (
    <li>
      <Link
        to={supportRequestPath(request.id)}
        className="group flex min-w-0 items-center gap-3 px-4 py-3 outline-none transition-colors duration-150 hover:bg-slate-50 focus-visible:bg-slate-50 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400 sm:px-5"
      >
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <span className="shrink-0 font-mono text-micro font-semibold text-slate-500">
              #{request.number}
            </span>
            <span className="min-w-0 truncate text-body font-semibold text-slate-800 group-hover:text-slate-950">
              {request.subject}
            </span>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-micro font-normal text-slate-500">
            <StatusBadge status={request.status} />
            <PriorityBadge priority={request.priority} />
            {request.isEscalated && (
              <span className="inline-flex items-center gap-1 font-semibold text-amber-700">
                <Flag size={10} strokeWidth={2.5} />
                Escalated
              </span>
            )}
            {request.reporter?.name && (
              <span className="inline-flex min-w-0 items-center gap-1">
                <UserRound size={10} strokeWidth={2.4} />
                <span className="truncate">{request.reporter.name}</span>
              </span>
            )}
            {request.assignee && (
              <span className="hidden min-w-0 truncate sm:inline">
                Engineer: {request.assignee}
              </span>
            )}
          </div>
        </div>

        <div className="hidden shrink-0 flex-col items-end gap-1 text-micro font-normal text-slate-500 sm:flex">
          {request.lastActivityAt && (
            <TimeAgo rfc3339={request.lastActivityAt} />
          )}
          <span className="inline-flex items-center gap-2 tabular-nums">
            <span className="inline-flex items-center gap-1">
              <MessagesSquare size={10} strokeWidth={2.4} />
              {request.messageCount}
            </span>
            {request.attachmentCount > 0 && (
              <span className="inline-flex items-center gap-1">
                <Paperclip size={10} strokeWidth={2.4} />
                {request.attachmentCount}
              </span>
            )}
          </span>
        </div>
        <ChevronRight
          size={15}
          strokeWidth={2.3}
          className="shrink-0 text-slate-300 transition-transform duration-150 group-hover:translate-x-0.5 group-hover:text-slate-500"
        />
      </Link>
    </li>
  );
};

const Requests = (props: { info?: SupportInfo }) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const current =
    FILTERS.find((item) => item.value === searchParams.get("status")) ??
    FILTERS[0];
  const available = props.info?.state === SupportInfo_State.AVAILABLE;
  const query = useSupportRequests(current.filter, !!props.info && available);
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <header className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5 border-b border-slate-100 px-4 py-3.5 sm:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <Inbox size={15} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <h2 className="truncate text-body font-semibold text-slate-800">
              Requests from this Cluster
            </h2>
            <p className="truncate text-micro font-normal text-slate-500">
              The technical requests opened from the console of this Cluster
            </p>
          </div>
        </div>
        <SegmentedControl
          aria-label="Request status"
          size="xs"
          value={current.value}
          onChange={(value) =>
            setSearchParams(
              (params) => {
                params.set("status", value);
                return params;
              },
              { replace: true },
            )
          }
          data={FILTERS.map((item) => ({
            value: item.value,
            label: item.label,
          }))}
        />
      </header>

      {!props.info || (available && query.isLoading) ? (
        <div className="flex flex-col gap-2 px-4 py-4 sm:px-5">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} height={52} radius="md" />
          ))}
        </div>
      ) : !available ? (
        <div className="flex min-h-40 flex-col items-center justify-center gap-2 px-6 py-8 text-center">
          <span className="text-body font-semibold text-slate-700">
            Requests are not available right now
          </span>
          <span className="max-w-md text-micro font-normal text-slate-500">
            The requests of your organization remain available in the customer
            portal.
          </span>
        </div>
      ) : query.isError ? (
        <div className="px-4 py-4 sm:px-5">
          <Alert color="red" title="Could not load the requests">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <RpcError error={query.error} />
              <Button
                size="compact-xs"
                variant="outline"
                onClick={() => query.refetch()}
              >
                Try again
              </Button>
            </div>
          </Alert>
        </div>
      ) : items.length === 0 && query.hasNextPage ? (
        <div className="flex min-h-48 flex-col items-center justify-center gap-2.5 px-6 py-8 text-center">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
            <Search size={17} strokeWidth={2.2} />
          </span>
          <span className="text-body font-semibold text-slate-800">
            No requests of this Cluster in the latest activity
          </span>
          <span className="max-w-sm text-micro font-normal text-slate-500">
            The latest requests of your organization come from elsewhere. Older
            requests have not been searched yet.
          </span>
          <div className="mt-1">
            <Button
              variant="default"
              size="xs"
              loading={query.isFetchingNextPage}
              onClick={() => query.fetchNextPage()}
            >
              Search older requests
            </Button>
          </div>
        </div>
      ) : items.length === 0 ? (
        <div className="flex min-h-48 flex-col items-center justify-center gap-2.5 px-6 py-8 text-center">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
            <Inbox size={17} strokeWidth={2.2} />
          </span>
          <span className="text-body font-semibold text-slate-800">
            {current.value === "open"
              ? "No open requests"
              : current.value === "closed"
                ? "No resolved requests yet"
                : "No requests yet"}
          </span>
          <span className="max-w-sm text-micro font-normal text-slate-500">
            When something does not work as expected, open a request with the
            diagnostics of the Cluster attached.
          </span>
          <div className="mt-1">
            <NewRequestButton />
          </div>
        </div>
      ) : (
        <>
          <ul className="divide-y divide-slate-100" aria-label="Requests">
            {items.map((request) => (
              <RequestRow key={request.id} request={request} />
            ))}
          </ul>
          {query.hasNextPage && (
            <div className="flex justify-center border-t border-slate-100 px-4 py-3">
              <Button
                variant="default"
                size="xs"
                loading={query.isFetchingNextPage}
                onClick={() => query.fetchNextPage()}
              >
                Load more
              </Button>
            </div>
          )}
        </>
      )}
    </section>
  );
};

const Step = (props: {
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  title: string;
  children: React.ReactNode;
  last?: boolean;
}) => {
  const Icon = props.icon;
  return (
    <li className="relative flex gap-3 pb-4 last:pb-0">
      {!props.last && (
        <span
          aria-hidden="true"
          className="absolute bottom-0 left-[13px] top-7 w-px bg-slate-200"
        />
      )}
      <span className="relative z-10 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500">
        <Icon size={12} strokeWidth={2.4} />
      </span>
      <div className="min-w-0 pt-0.5">
        <div className="text-xs font-semibold text-slate-800">
          {props.title}
        </div>
        <p className="mt-0.5 text-micro font-normal leading-[1.55] text-slate-500">
          {props.children}
        </p>
      </div>
    </li>
  );
};

const HowItWorks = (props: { info?: SupportInfo }) => (
  <Panel
    icon={Stethoscope}
    title="How it works"
    description="What is shared with Octelium and when"
  >
    <ol className="flex flex-col">
      <Step icon={Send} title="Describe the issue">
        Choose a priority, describe what happens and attach files. The installed
        versions and the Regions of the Cluster are added to help the engineers.
      </Step>
      <Step icon={Stethoscope} title="Attach diagnostics when you want">
        Component logs are only sent when you collect, review and confirm them.
        Common secrets such as tokens and passwords are removed in your browser
        before you review them.
      </Step>
      <Step icon={MessagesSquare} title="Follow the conversation" last>
        Answers appear here and in the customer portal, where the members of
        your organization can follow the request.
      </Step>
    </ol>
    {props.info?.portalURL && (
      <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-slate-100 pt-3 text-micro font-normal text-slate-500">
        <PortalLink href={props.info.portalURL}>
          Open the customer portal
        </PortalLink>
        {props.info.supportEmail && (
          <span>
            or email{" "}
            <a
              href={`mailto:${props.info.supportEmail}`}
              className="font-semibold text-slate-700 underline decoration-slate-300 underline-offset-2 hover:text-slate-900"
            >
              {props.info.supportEmail}
            </a>
          </span>
        )}
      </div>
    )}
  </Panel>
);

const Main = () => {
  const info = useSupportInfo();
  const queryClient = useQueryClient();
  const fetching = useIsFetching({ queryKey: supportKeys.all });
  const available = info.data?.state === SupportInfo_State.AVAILABLE;

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex w-full flex-col gap-4 py-4"
    >
      <Meta title="Support" />

      <PageTitleAddon>
        {info.data && <StateBadge state={info.data.state} />}
      </PageTitleAddon>

      <PageActions>
        <RefreshButton
          label="Refresh the page"
          loading={fetching > 0}
          onClick={() =>
            queryClient.invalidateQueries({ queryKey: supportKeys.all })
          }
        />
        <NewRequestButton disabled={!available} />
      </PageActions>

      {info.isError ? (
        <Alert color="red" title="Could not check the support of the Cluster">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <RpcError error={info.error} />
            <Button
              size="compact-xs"
              variant="outline"
              onClick={() => info.refetch()}
            >
              Try again
            </Button>
          </div>
        </Alert>
      ) : (
        info.data && <StateNotice info={info.data} />
      )}

      <Overview info={info.data} loading={info.isLoading} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,340px)]">
        <div className="min-w-0">
          <Requests info={info.data} />
        </div>
        <aside className="flex min-w-0 flex-col gap-4">
          <HowItWorks info={info.data} />
          {info.data?.organization && info.data.portalURL && (
            <a
              href={`${info.data.portalURL}/organizations/${encodeURIComponent(info.data.organization.uid)}/support`}
              target="_blank"
              rel="noopener noreferrer"
              className="group flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3.5 outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              <div className="min-w-0">
                <div className="text-xs font-semibold text-slate-800">
                  All the requests of {info.data.organization.displayName}
                </div>
                <div className="text-micro font-normal text-slate-500">
                  Billing, licensing and requests from other Clusters
                </div>
              </div>
              <ArrowUpRight
                size={14}
                strokeWidth={2.4}
                className="shrink-0 text-slate-400 transition-transform duration-150 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-slate-600"
              />
            </a>
          )}
        </aside>
      </div>
    </motion.div>
  );
};

export default Main;
