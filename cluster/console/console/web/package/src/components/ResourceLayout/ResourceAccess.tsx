import * as AccessP from "@/apis/accessv1/accessv1";
import RefreshButton from "@/components/RefreshButton";
import RpcError from "@/components/RpcError";
import AccessListFilters from "@/pages/access/ListFilters";
import { DeliveryRow, IdentityRow } from "@/pages/access/Integration/Activity";
import { getAccessListFilters } from "@/pages/access/listFilters";
import { getResourceComponentInfo } from "@/pages/utils/resourceRegistry";
import { getResourcePath, listResourcesPB, Resource } from "@/utils/pb";
import { retryQuery } from "@/utils/queryPolicy";
import { Button, SegmentedControl, Select } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ShieldCheck } from "lucide-react";
import * as React from "react";
import {
  Navigate,
  useLocation,
  useNavigate,
  useParams,
} from "react-router-dom";
import Paginator, { buildFilterChips, FilterChips } from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import SearchList from "../SearchList";
import TimeAgo from "../TimeAgo";
import {
  buildResourceAccessListRequest,
  getResourceAccessSections,
  ResourceAccessSection,
} from "./resourceAccess";
import { useListParams } from "./ResourceListToolbar";
import { useContextResource } from "./utils";

type AccessList =
  | AccessP.RequestList
  | AccessP.ReviewList
  | AccessP.PolicyList
  | AccessP.CatalogList
  | AccessP.IntegrationIdentityList
  | AccessP.IntegrationBindingList;

const ListSkeleton = () => (
  <div
    className="flex flex-col gap-3"
    role="status"
    aria-label="Loading access resources"
  >
    {Array.from({ length: 3 }, (_, index) => (
      <div
        key={index}
        className="h-24 motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-100"
      />
    ))}
  </div>
);

const AccessItems = (props: {
  list: AccessList;
  section: ResourceAccessSection;
  resource: Resource;
}) => {
  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;
  const showIntegration = props.resource.kind !== "Integration";

  if (props.section.kind === "IntegrationBinding") {
    return (
      <ul className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        {(props.list as AccessP.IntegrationBindingList).items.map((item) => (
          <DeliveryRow
            key={item.metadata?.uid || item.metadata?.name}
            item={item}
            showIntegration={showIntegration}
          />
        ))}
      </ul>
    );
  }
  if (props.section.kind === "IntegrationIdentity") {
    return (
      <ul className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        {(props.list as AccessP.IntegrationIdentityList).items.map((item) => (
          <IdentityRow
            key={item.metadata?.uid || item.metadata?.name}
            item={item}
            showIntegration={showIntegration}
          />
        ))}
      </ul>
    );
  }

  const Labels = getResourceComponentInfo("access", props.section.kind)?.List
    .labelComponent;
  return (
    <ResourceListWrapper>
      {(props.list.items as Resource[]).map((item) => (
        <ResourceListItem
          key={item.metadata?.uid || item.metadata?.name}
          path={getResourcePath(item)}
          overlayLabel={`Open ${item.kind} ${item.metadata?.name}`}
          state={{ returnTo }}
          compact
        >
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <span className="truncate text-sm font-semibold text-slate-900">
              {item.metadata?.name}
            </span>
            {item.metadata?.displayName && (
              <span className="truncate text-sm font-normal text-slate-600">
                {item.metadata.displayName}
              </span>
            )}
            {item.metadata?.createdAt && (
              <span className="ml-auto text-xs font-normal text-slate-500">
                Created <TimeAgo rfc3339={item.metadata.createdAt} />
              </span>
            )}
          </div>
          {Labels && <Labels item={item} />}
        </ResourceListItem>
      ))}
    </ResourceListWrapper>
  );
};

const ResourceAccessList = (props: {
  resource: Resource;
  section: ResourceAccessSection;
}) => {
  const location = useLocation();
  const { searchParams, setParam, setParams, removeParam, clearFilters } =
    useListParams();
  const { request, filterParams, invalidFilters } =
    buildResourceAccessListRequest(
      props.resource,
      props.section,
      location.search,
    );
  const query = useQuery({
    queryKey: [`access.list${props.section.kind}`, "resource", request],
    queryFn: async ({ signal }) =>
      (
        await listResourcesPB("access", props.section.kind, request, {
          abort: signal,
        })
      ).response as AccessList,
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    retry: retryQuery,
  });
  const filters = getAccessListFilters(props.section.kind);
  const chips = buildFilterChips(filterParams).map((chip) => {
    const filter = filters.find((item) => item.key === chip.key);
    return {
      ...chip,
      label: filter?.label ?? chip.label,
      value:
        filter?.options?.find((option) => option.value === chip.value)?.label ??
        chip.value,
    };
  });
  const orderType =
    searchParams.get("common.orderBy.type") === "NAME" ? "NAME" : "CREATED_AT";
  const orderMode =
    searchParams.get("common.orderBy.mode") === "ASC" ? "ASC" : "DESC";
  const filtered = chips.length > 0 || !!searchParams.get("common.query");

  return (
    <section
      className="flex min-w-0 flex-col gap-4"
      aria-label={`${props.section.label} for this resource`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs font-normal text-slate-500">
          {props.section.description}
        </p>
        <RefreshButton
          label="Refresh access resources"
          loading={query.isFetching}
          onClick={() => query.refetch()}
        />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <div className="min-w-48 flex-1">
          <SearchList
            label="Search access resources"
            placeholder={`Search ${props.section.label.toLowerCase()}…`}
          />
        </div>
        <Select
          w={156}
          allowDeselect={false}
          aria-label="Sort access resources"
          value={`${orderType}.${orderMode}`}
          data={[
            { label: "Newest first", value: "CREATED_AT.DESC" },
            { label: "Oldest first", value: "CREATED_AT.ASC" },
            { label: "Name A–Z", value: "NAME.ASC" },
            { label: "Name Z–A", value: "NAME.DESC" },
          ]}
          onChange={(value) => {
            if (!value) return;
            const [type, mode] = value.split(".");
            setParams({
              "common.orderBy.type": type,
              "common.orderBy.mode": mode,
            });
          }}
        />
        <Select
          w={90}
          allowDeselect={false}
          aria-label="Access items per page"
          value={`${request.common.itemsPerPage}`}
          data={["10", "25", "50", "100"]}
          onChange={(value) => value && setParam("common.itemsPerPage", value)}
        />
      </div>
      <AccessListFilters kind={props.section.kind} />
      {filtered && (
        <div className="flex flex-wrap items-center gap-3">
          <FilterChips chips={chips} onRemove={removeParam} />
          <button
            type="button"
            className="text-xs font-semibold text-slate-600 hover:text-slate-950"
            onClick={clearFilters}
          >
            Clear filters
          </button>
        </div>
      )}
      {invalidFilters && (
        <p role="status" className="text-xs text-amber-700">
          Some URL filters were invalid and have been ignored.
        </p>
      )}
      {query.isError && (
        <div
          role="alert"
          className="flex flex-col items-start gap-3 rounded-xl border border-red-200 bg-red-50/60 p-4"
        >
          <span className="text-sm font-semibold text-slate-900">
            Access resources could not be loaded
          </span>
          <RpcError error={query.error} />
          <Button variant="default" size="xs" onClick={() => query.refetch()}>
            Try again
          </Button>
        </div>
      )}
      {query.isPending ? (
        <ListSkeleton />
      ) : (
        query.data && (
          <React.Suspense fallback={<ListSkeleton />}>
            {query.data.items.length > 0 ? (
              <AccessItems
                list={query.data}
                section={props.section}
                resource={props.resource}
              />
            ) : (
              <div
                role="status"
                className="rounded-xl border border-slate-200 bg-white px-4 py-10 text-center text-sm text-slate-500"
              >
                No {props.section.label.toLowerCase()} found
                {filtered ? " matching these filters" : " for this resource"}.
              </div>
            )}
            <Paginator meta={query.data.listResponseMeta} showFilters={false} />
          </React.Suspense>
        )
      )}
    </section>
  );
};

const ResourceAccess = () => {
  const ctx = useContextResource();
  const { section: selected } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  if (!ctx?.data) return null;

  const resource = ctx.data;
  const sections = getResourceAccessSections(
    resource.apiVersion,
    resource.kind,
  );
  if (sections.length === 0) return null;
  const path = `${getResourcePath(resource)}/access`;
  const section = sections.find((item) => item.value === selected);
  if (!section) {
    return (
      <Navigate
        to={`${path}/${sections[0].value}${location.search}`}
        state={location.state}
        replace
      />
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <header className="flex min-w-0 items-center gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-raised">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
          <ShieldCheck size={18} strokeWidth={2.2} />
        </div>
        <div className="min-w-0">
          <h1 className="text-base font-bold text-slate-900">Access</h1>
          <p className="truncate text-xs font-normal text-slate-500">
            {resource.kind} ·{" "}
            {resource.metadata?.displayName || resource.metadata?.name}
          </p>
        </div>
      </header>
      <div className="max-w-full overflow-x-auto">
        <SegmentedControl
          aria-label="Access sections"
          value={section.value}
          styles={{ root: { minWidth: "max-content" } }}
          data={sections.map(({ value, label }) => ({ value, label }))}
          onChange={(value) =>
            navigate(`${path}/${value}`, {
              state: location.state,
              preventScrollReset: true,
            })
          }
        />
      </div>
      <ResourceAccessList
        key={`${resource.metadata?.uid || resource.metadata?.name}/${section.value}`}
        resource={resource}
        section={section}
      />
    </div>
  );
};

export default ResourceAccess;
