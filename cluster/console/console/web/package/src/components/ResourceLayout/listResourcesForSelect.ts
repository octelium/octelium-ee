import {
  CommonListOptions,
  CommonListOptions_OrderBy_Type,
} from "@/apis/metav1/metav1";
import {
  getClientResourceList,
  getPBResourceListFromAPI,
  Resource,
  ResourceList,
} from "@/utils/pb";

const ITEMS_PER_PAGE = 1000;

export const listResourcesForSelect = async (
  api: string,
  kind: string,
): Promise<Resource[]> => {
  // @ts-ignore
  const requestType = getPBResourceListFromAPI(api)?.[`List${kind}Options`];
  const client = getClientResourceList(api as any) as any;
  const listMethod = client?.[`list${kind}`];

  if (!requestType?.create || !listMethod) {
    throw new Error(`The ${api}/${kind} list API is not available`);
  }

  const items: Resource[] = [];
  let page = 0;
  let hasMore = false;

  do {
    const request = requestType.create({
      common: CommonListOptions.create({
        page,
        itemsPerPage: ITEMS_PER_PAGE,
        orderBy: { type: CommonListOptions_OrderBy_Type.NAME },
      }),
    });

    const result = await listMethod.call(client, request);
    const response = result?.response as ResourceList | undefined;

    if (!response) {
      throw new Error(`The ${api}/${kind} list API returned no response`);
    }

    items.push(...response.items);
    hasMore = response.listResponseMeta?.hasMore ?? false;

    if (hasMore && response.items.length === 0) {
      throw new Error(
        `The ${api}/${kind} list API reported more pages but returned an empty page`,
      );
    }

    page += 1;
  } while (hasMore);

  return items;
};
