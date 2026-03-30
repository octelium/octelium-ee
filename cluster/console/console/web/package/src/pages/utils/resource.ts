import { onError } from "@/utils";
import {
  getResourceClient,
  invalidateResource,
  invalidateResourceList,
  Resource,
} from "@/utils/pb";
import { useMutation } from "@tanstack/react-query";

export const useUpdateResource = () => {
  return useMutation({
    mutationFn: async (req: Resource) => {
      // @ts-ignore
      const { response } =
        // @ts-ignore
        await getResourceClient(req)[`update${req.kind}`](req);
      return response as Resource;
    },

    onSuccess: (response) => {
      invalidateResource(response);
      invalidateResourceList(response);
    },
    onError: onError,
  });
};
