import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import * as UserP from "@/apis/userv1/userv1";
import { useQuery } from "@tanstack/react-query";

import { namespaceFromName } from "@/utils";
import { getUserClient, getUserMainClient } from "@/utils/client";

const MAX_PAGES = 1000;

export const listAllServices = async (
  namespace = "",
): Promise<UserP.Service[]> => {
  const items: UserP.Service[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listService(
      UserP.ListServiceOptions.create({
        common: { page, itemsPerPage: 500 },
        namespace,
      }),
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const listAllNamespaces = async (): Promise<UserP.Namespace[]> => {
  const items: UserP.Namespace[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listNamespace(
      UserP.ListNamespaceOptions.create({
        common: { page, itemsPerPage: 500 },
      }),
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const listAllCatalogs = async (): Promise<AccessP.Catalog[]> => {
  const items: AccessP.Catalog[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserClient().listCatalog(
      AccessP.ListUserCatalogOptions.create({
        common: { page, itemsPerPage: 500 },
      }),
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > MAX_PAGES) break;
    page += 1;
  }

  return items;
};

export const useServices = (namespace = "") =>
  useQuery({
    queryKey: ["userapi", "listService", namespace],
    queryFn: () => listAllServices(namespace),
    staleTime: 60 * 1000,
  });

export const useNamespaces = () =>
  useQuery({
    queryKey: ["userapi", "listNamespace"],
    queryFn: listAllNamespaces,
    staleTime: 5 * 60 * 1000,
  });

export const useCatalogs = () =>
  useQuery({
    queryKey: ["user", "listCatalog"],
    queryFn: listAllCatalogs,
    staleTime: 60 * 1000,
  });

export const useService = (name?: string) =>
  useQuery({
    queryKey: ["userapi", "getService", name],
    enabled: !!name,
    staleTime: 60 * 1000,
    queryFn: async () => {
      const items = await listAllServices(namespaceFromName(name));
      return items.find((item) => item.metadata?.name === name);
    },
  });

export const useSubjectUser = (name?: string) =>
  useQuery({
    queryKey: ["access", "getSubjectUser", name],
    enabled: !!name,
    staleTime: 5 * 60 * 1000,
    retry: false,
    queryFn: async () => {
      const { response } = await getUserClient().getSubjectUser(
        AccessP.GetSubjectUserRequest.create({
          userRef: MetaP.ObjectReference.create({ name: name! }),
        }),
      );
      return response;
    },
  });
