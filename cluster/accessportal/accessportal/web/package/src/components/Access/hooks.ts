import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import * as UserP from "@/apis/userv1/userv1";
import { useQuery } from "@tanstack/react-query";

import { getUserClient, getUserMainClient } from "@/utils/client";

const MAX_PAGES = 1000;

export const listAllServices = async (
  namespace = "",
  signal?: AbortSignal,
): Promise<UserP.Service[]> => {
  const items: UserP.Service[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listService(
      UserP.ListServiceOptions.create({
        common: { page, itemsPerPage: 500 },
        namespace,
      }),
      { abort: signal },
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const listAllNamespaces = async (
  signal?: AbortSignal,
): Promise<UserP.Namespace[]> => {
  const items: UserP.Namespace[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listNamespace(
      UserP.ListNamespaceOptions.create({
        common: { page, itemsPerPage: 500 },
      }),
      { abort: signal },
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const listAllCatalogs = async (
  signal?: AbortSignal,
): Promise<AccessP.UserCatalog[]> => {
  const items: AccessP.UserCatalog[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserClient().listCatalog(
      AccessP.ListUserCatalogOptions.create({
        common: { page, itemsPerPage: 500 },
      }),
      { abort: signal },
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const useServices = (namespace = "", enabled = true) =>
  useQuery({
    queryKey: ["userapi", "listService", namespace],
    queryFn: ({ signal }) => listAllServices(namespace, signal),
    staleTime: 60 * 1000,
    enabled,
  });

export const useNamespaces = (enabled = true) =>
  useQuery({
    queryKey: ["userapi", "listNamespace"],
    queryFn: ({ signal }) => listAllNamespaces(signal),
    staleTime: 5 * 60 * 1000,
    enabled,
  });

export const useCatalogs = () =>
  useQuery({
    queryKey: ["user", "listCatalog"],
    queryFn: ({ signal }) => listAllCatalogs(signal),
    staleTime: 60 * 1000,
  });

export const listCatalogServices = async (
  catalogName: string,
  signal?: AbortSignal,
): Promise<UserP.Service[]> => {
  const items: UserP.Service[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserClient().listCatalogService(
      AccessP.ListUserCatalogServiceOptions.create({
        common: { page, itemsPerPage: 500 },
        catalogRef: MetaP.ObjectReference.create({ name: catalogName }),
      }),
      { abort: signal },
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const useCatalogServices = (catalogName?: string) =>
  useQuery({
    queryKey: ["user", "listCatalogService", catalogName],
    enabled: !!catalogName,
    staleTime: 60 * 1000,
    queryFn: ({ signal }) => listCatalogServices(catalogName!, signal),
  });

export const useInbox = () =>
  useQuery({
    queryKey: ["user", "getInbox"],
    staleTime: 15 * 1000,
    refetchInterval: 30 * 1000,
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().getInbox(
        AccessP.GetInboxRequest.create({}),
        { abort: signal },
      );
      return response;
    },
  });

export const useRequestPreview = (specJson: string) =>
  useQuery({
    queryKey: ["user", "previewRequest", specJson],
    enabled: !!specJson,
    staleTime: 10 * 1000,
    retry: false,
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().previewRequest(
        AccessP.Request.create({
          spec: AccessP.Request_Spec.fromJsonString(specJson),
        }),
        { abort: signal },
      );
      return response;
    },
  });

export const useService = (name?: string) =>
  useQuery({
    queryKey: ["userapi", "getService", name],
    enabled: !!name,
    staleTime: 60 * 1000,
    queryFn: async ({ signal }) => {
      const { response } = await getUserMainClient().getService(
        MetaP.GetOptions.create({ name: name! }),
        { abort: signal },
      );
      return response;
    },
  });

export const useSubjectUser = (name?: string) =>
  useQuery({
    queryKey: ["access", "getSubjectUser", name],
    enabled: !!name,
    staleTime: 5 * 60 * 1000,
    retry: false,
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().getSubjectUser(
        AccessP.GetSubjectUserRequest.create({
          userRef: MetaP.ObjectReference.create({ name: name! }),
        }),
        { abort: signal },
      );
      return response;
    },
  });
