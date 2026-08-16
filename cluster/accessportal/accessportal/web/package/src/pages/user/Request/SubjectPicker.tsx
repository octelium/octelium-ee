import * as AccessP from "@/apis/accessv1/accessv1";
import { Pagination } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { Search, UserRound } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

import { Avatar, Badge, EmptyState, ErrorState, Loading } from "../../../ui";
import { userTypeMeta } from "../../../utils";
import { getUserClient } from "../../../utils/client";

const ITEMS_PER_PAGE = 8;

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

  const qry = useQuery({
    queryKey: ["user", "listSubjectUser", debouncedQuery, page],
    queryFn: async () => {
      const { response } = await getUserClient().listSubjectUser(
        AccessP.ListSubjectUserOptions.create({
          common: { page: page - 1, itemsPerPage: ITEMS_PER_PAGE },
          query: debouncedQuery,
        }),
      );
      return response;
    },
    placeholderData: (previous) => previous,
  });

  const meta = qry.data?.listResponseMeta;
  const totalPages = meta
    ? Math.max(1, Math.ceil(meta.totalCount / (meta.itemsPerPage || ITEMS_PER_PAGE)))
    : 1;

  return (
    <div className="flex flex-col gap-3">
      <div className="relative">
        <Search
          size={13}
          strokeWidth={2.5}
          className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
        />
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search by name or email..."
          aria-label="Search users"
          className="w-full pl-8 pr-3 h-9 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 focus:shadow-[0_0_0_2px_rgba(148,163,184,0.2)] transition-all duration-150 placeholder:text-slate-400 placeholder:font-semibold"
        />
      </div>

      {qry.isLoading ? (
        <Loading label="Loading users..." />
      ) : qry.isError ? (
        <ErrorState title="Could not load users" onRetry={() => qry.refetch()} />
      ) : (qry.data?.items.length ?? 0) === 0 ? (
        <EmptyState
          icon={<UserRound size={20} strokeWidth={2} />}
          title="No users found"
          description={query ? "Try a different name or email." : "There are no users available for delegated requests."}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {qry.data!.items.map((subject) => {
            const name = subject.displayName || subject.userRef?.name || "User";
            const type = userTypeMeta(subject.type);
            const selected = props.value?.userRef?.name === subject.userRef?.name;
            return (
              <button
                key={subject.userRef?.name}
                type="button"
                onClick={() => props.onChange(subject)}
                aria-pressed={selected}
                className={twMerge(
                  "w-full flex items-center gap-3 text-left rounded-lg border px-3 py-2.5 transition-[border-color,box-shadow,background-color] duration-150",
                  selected
                    ? "border-slate-900 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
                    : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
                )}
              >
                <Avatar src={subject.picURL} name={name} size="sm" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-[0.82rem] font-bold text-slate-800 truncate">{name}</span>
                    <Badge tone={type.tone}>{type.label}</Badge>
                  </div>
                  <div className="text-[0.7rem] font-medium text-slate-400 truncate">
                    {subject.email || subject.userRef?.name || "—"}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex justify-center pt-1">
          <Pagination value={page} total={totalPages} onChange={setPage} color="dark" size="sm" radius="md" />
        </div>
      )}
    </div>
  );
};

export default SubjectPicker;
