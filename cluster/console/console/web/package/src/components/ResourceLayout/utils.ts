import * as MetaPB from "@/apis/metav1/metav1";
import {
  getAPIFromAPIVersion,
  getAPIKindFromPath,
  getClient,
  getGetKeyFromPath,
  getGetKeyFromRef,
  Resource,
} from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useParams } from "react-router-dom";

export const useContextResource = () => {
  let { name } = useParams();

  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) {
    return undefined;
  }

  const { isSuccess, isLoading, isError, data } = useQuery({
    queryKey: [getGetKeyFromPath(loc.pathname), name],
    queryFn: () => {
      //@ts-ignore
      return getClient(apiKind.api)[`get${apiKind.kind}`]({
        name,
      } as any);
    },
  });

  return {
    isSuccess,
    isLoading,
    isError,
    data: data?.response as Resource | undefined,
  };
};

export const useResourceFromRef = (resourceRef: MetaPB.ObjectReference) => {
  const { isSuccess, isLoading, isError, data } = useQuery({
    queryKey: [getGetKeyFromRef(resourceRef!), resourceRef!.name],
    queryFn: () => {
      //@ts-ignore
      return getClient(getAPIFromAPIVersion(resourceRef.apiVersion))[
        `get${resourceRef!.kind}`
      ]({
        name: resourceRef!.name,
      } as any);
    },
    enabled: resourceRef?.name.length > 0,
  });

  return {
    isSuccess,
    isLoading,
    isError,
    data: data?.response as Resource | undefined,
  };
};
