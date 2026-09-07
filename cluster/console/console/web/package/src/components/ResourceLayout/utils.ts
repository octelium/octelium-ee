import * as MetaPB from "@/apis/metav1/metav1";
import {
  API,
  getAPIFromAPIVersion,
  getAPIKindFromPath,
  getGetKeyFromPath,
  getGetKeyFromRef,
  getResourcePB,
  Resource,
} from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useParams } from "react-router-dom";

export const useContextResource = () => {
  let { name } = useParams();

  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);

  const { isSuccess, isLoading, isError, error, data, refetch } = useQuery({
    queryKey: [getGetKeyFromPath(loc.pathname), name],
    queryFn: () => getResourcePB(apiKind!.api, apiKind!.kind, { name }),
    enabled: !!apiKind,
    retry: (failureCount, error) =>
      (error as { code?: string })?.code !== "NOT_FOUND" && failureCount < 3,
  });

  if (!apiKind) {
    return undefined;
  }

  return {
    isSuccess,
    isLoading,
    isPending: isLoading,
    isError,
    error,
    refetch,
    data: data?.response as Resource | undefined,
  };
};

export const useResourceFromRef = (resourceRef: MetaPB.ObjectReference) => {
  const { isSuccess, isLoading, isError, data } = useQuery({
    queryKey: [getGetKeyFromRef(resourceRef!), resourceRef!.name],
    queryFn: () =>
      getResourcePB(
        getAPIFromAPIVersion(resourceRef.apiVersion) as API,
        resourceRef.kind,
        { name: resourceRef.name },
      ),
    enabled: resourceRef?.name.length > 0,
  });

  return {
    isSuccess,
    isLoading,
    isError,
    data: data?.response as Resource | undefined,
  };
};
