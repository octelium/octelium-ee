import {
  CommonListOptions,
  CommonListOptions_OrderBy_Type,
} from "@/apis/visibilityv1/meta/vmetav1";
import {
  API,
  getListOptionsPB,
  listResourcesPB,
  Resource,
  ResourceList,
} from "@/utils/pb";

export const SELECT_PAGE_SIZE = 100;

export const listResourcesForSelect = async (
  api: string,
  kind: string,
  query?: string,
): Promise<Resource[]> => {
  const requestType = getListOptionsPB(api as API, kind);
  if (!requestType?.create) {
    throw new Error(`The ${api}/${kind} list API is not available`);
  }

  const request = requestType.create({
    common: CommonListOptions.create({
      page: 0,
      itemsPerPage: SELECT_PAGE_SIZE,
      orderBy: { type: CommonListOptions_OrderBy_Type.NAME },
      query: query?.trim() ?? "",
    }),
  });

  const result = await listResourcesPB(api as API, kind, request);
  const response = result?.response as ResourceList | undefined;

  if (!response) {
    throw new Error(`The ${api}/${kind} list API returned no response`);
  }

  return response.items;
};
