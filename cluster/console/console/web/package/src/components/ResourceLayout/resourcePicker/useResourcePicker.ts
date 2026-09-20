import {
  API,
  getResourcePathFromAPIKind,
  Resource,
  ResourceName,
} from "@/utils/pb";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { listResourcesForSelect } from "../listResourcesForSelect";
import { NON_CREATABLE } from "./constants";

type CreateReturnState = {
  reopenSelect?: string;
  createdResourceName?: string;
};

export const useResourcePicker = (args: {
  api: string;
  kind: string;
  onCreated: (
    item: Resource,
    resolve: (name: string) => Resource | undefined,
  ) => void;
}) => {
  const { api, kind } = args;
  const location = useLocation();
  const navigate = useNavigate();
  const instanceId = React.useId();
  const token = `${api}/${kind}${instanceId}`;

  const [opened, setOpened] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const [debouncedSearch, setDebouncedSearch] = React.useState("");
  const claimed = React.useRef<string | undefined>(undefined);
  const seen = React.useRef(new Map<string, Resource>());
  const onCreated = React.useRef(args.onCreated);

  React.useEffect(() => {
    onCreated.current = args.onCreated;
  });

  React.useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedSearch(search), 220);
    return () => window.clearTimeout(timeout);
  }, [search]);

  const query = useQuery({
    queryKey: ["listSelectComponent", api, kind, debouncedSearch],
    queryFn: () => listResourcesForSelect(api, kind, debouncedSearch),
    placeholderData: keepPreviousData,
  });

  const items = React.useMemo(() => query.data ?? [], [query.data]);

  React.useEffect(() => {
    for (const item of items) {
      const name = item.metadata?.name;
      if (name) seen.current.set(name, item);
    }
  }, [items]);

  const resolve = React.useCallback(
    (name?: string) => (name ? seen.current.get(name) : undefined),
    [],
  );

  const resourcePath = getResourcePathFromAPIKind({
    api: api as API,
    kind: kind as ResourceName,
  });
  const canCreate =
    resourcePath.length > 0 && !NON_CREATABLE.has(`${api}/${kind}`);

  const openCreate = React.useCallback(() => {
    const previous =
      location.state && typeof location.state === "object"
        ? (location.state as Record<string, unknown>)
        : {};

    navigate(`/${api}/${resourcePath}/create`, {
      state: {
        createInDrawer: true,
        returnTo: `${location.pathname}${location.search}`,
        returnState: { ...previous, reopenSelect: token },
      },
    });
  }, [
    api,
    location.pathname,
    location.search,
    location.state,
    navigate,
    resourcePath,
    token,
  ]);

  const state = location.state as CreateReturnState | null;
  const reopenToken = state?.reopenSelect;

  React.useEffect(() => {
    if (reopenToken !== token) return;

    claimed.current = state?.createdResourceName;
    setOpened(true);
    query.refetch();

    const next =
      location.state && typeof location.state === "object"
        ? { ...(location.state as Record<string, unknown>) }
        : {};
    delete next.reopenSelect;
    delete next.createdResourceName;

    navigate(`${location.pathname}${location.search}`, {
      replace: true,
      preventScrollReset: true,
      state: Object.keys(next).length > 0 ? next : undefined,
    });
  }, [reopenToken, token]);

  React.useEffect(() => {
    const name = claimed.current;
    if (!name) return;

    const created = seen.current.get(name);
    if (!created) return;

    claimed.current = undefined;
    onCreated.current(created, (value) => seen.current.get(value));
  }, [items]);

  return {
    opened,
    open: () => setOpened(true),
    close: () => setOpened(false),
    search,
    setSearch,
    items,
    resolve,
    canCreate,
    openCreate,
    isLoading: query.isLoading,
    isError: query.isError,
    errorMessage: query.error?.message,
    refetch: () => {
      query.refetch();
    },
  };
};
