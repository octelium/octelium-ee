import * as AccessP from "@/apis/accessv1/accessv1";
import { Pagination } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Check, UserRound } from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";

import {
  Avatar,
  Badge,
  EmptyState,
  ErrorState,
  Eyebrow,
  SearchInput,
  SkeletonRows,
} from "@/ui";
import { isWorkloadUser, shortName } from "@/utils";
import { getUserClient } from "@/utils/client";

const ITEMS_PER_PAGE = 6;
const MIN_QUERY_LENGTH = 2;

const SubjectPicker = (props: {
  value?: AccessP.SubjectUser;
  onChange: (subject: AccessP.SubjectUser) => void;
}) => {
  const [query, setQuery] = React.useState("");
  const [debouncedQuery, setDebouncedQuery] = React.useState("");
  const [page, setPage] = React.useState(1);

  React.useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedQuery(query.trim());
      setPage(1);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [query]);

  const enabled = debouncedQuery.length >= MIN_QUERY_LENGTH;
  const settling = query.trim() !== debouncedQuery;

  const qry = useQuery({
    queryKey: ["user", "listSubjectUser", debouncedQuery, page],
    enabled,
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().listSubjectUser(
        AccessP.ListSubjectUserOptions.create({
          common: { page: page - 1, itemsPerPage: ITEMS_PER_PAGE },
          query: debouncedQuery,
        }),
        { abort: signal },
      );
      return response;
    },
    placeholderData: keepPreviousData,
  });

  const meta = qry.data?.listResponseMeta;
  const totalPages = meta
    ? Math.max(1, Math.ceil(meta.totalCount / (meta.itemsPerPage || ITEMS_PER_PAGE)))
    : 1;
  const items = qry.data?.items ?? [];
  const stale = qry.isPlaceholderData || settling;

  return (
    <div className="flex flex-col gap-3">
      {props.value && (
        <div className="flex items-center gap-3 rounded-lg border border-slate-400 bg-slate-50 px-3 py-2.5 ring-1 ring-slate-300">
          <Avatar
            src={props.value.picURL}
            name={props.value.displayName || shortName(props.value.userRef?.name)}
            size="sm"
          />
          <div className="min-w-0 flex-1">
            <p className="truncate text-body font-semibold text-slate-800">
              {props.value.displayName || shortName(props.value.userRef?.name)}
            </p>
            <p className="truncate text-xs font-normal text-slate-500">
              {props.value.email || props.value.userRef?.name}
            </p>
          </div>
          <Badge>Recipient</Badge>
        </div>
      )}

      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder={
          props.value
            ? "Search for a different user"
            : "Search users by name or email"
        }
        ariaLabel="Search users"
      />

      {!enabled ? (
        <EmptyState
          icon={<UserRound size={18} strokeWidth={2} />}
          title={props.value ? "Change the recipient" : "Search for a user"}
          description={`Type at least ${MIN_QUERY_LENGTH} characters to find the user who should receive the access.`}
        />
      ) : qry.isLoading ? (
        <SkeletonRows rows={3} />
      ) : qry.isError && !qry.data ? (
        <ErrorState
          title="Could not load users"
          error={qry.error}
          onRetry={() => qry.refetch()}
        />
      ) : items.length === 0 && !stale ? (
        <EmptyState
          icon={<UserRound size={18} strokeWidth={2} />}
          title="No users found"
          description="Try a different name or email address."
        />
      ) : (
        <>
          <Eyebrow>
            {stale
              ? "Searching…"
              : `${meta?.totalCount ?? items.length} match${
                  (meta?.totalCount ?? items.length) === 1 ? "" : "es"
                }`}
          </Eyebrow>
          <div
            aria-busy={stale || undefined}
            className={twMerge(
              "flex flex-col gap-2 transition-opacity",
              stale && "opacity-50",
            )}
          >
            {items.map((subject) => {
              const name =
                subject.displayName || shortName(subject.userRef?.name) || "User";
              const selected =
                props.value?.userRef?.name === subject.userRef?.name;
              return (
                <button
                  key={subject.userRef?.name}
                  type="button"
                  disabled={stale}
                  onClick={() => props.onChange(subject)}
                  aria-pressed={selected}
                  className={twMerge(
                    "flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left outline-none transition-[border-color,box-shadow,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-wait",
                    selected
                      ? "border-slate-400 bg-slate-50 ring-1 ring-slate-300"
                      : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
                  )}
                >
                  <Avatar src={subject.picURL} name={name} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-body font-semibold text-slate-800">
                        {name}
                      </span>
                      {isWorkloadUser(subject.type) && <Badge>Workload</Badge>}
                    </div>
                    <div className="truncate text-xs font-normal text-slate-500">
                      {subject.email || subject.userRef?.name || "—"}
                    </div>
                  </div>
                  {selected && (
                    <Check size={15} strokeWidth={2.75} aria-hidden className="text-slate-900" />
                  )}
                </button>
              );
            })}
          </div>
        </>
      )}

      {enabled && totalPages > 1 && (
        <div className="flex justify-center pt-1">
          <Pagination
            value={page}
            total={totalPages}
            onChange={setPage}
            color="ink"
            size="sm"
            radius="md"
          />
        </div>
      )}
    </div>
  );
};

export default SubjectPicker;
