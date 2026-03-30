import { useLocation, useSearchParams } from "react-router-dom";
import { parseQueryString } from "../ResourceLayout/queryParse";
import { ObjectReference } from "@/apis/metav1/metav1";

type objectRef = {
  uid?: string;
  name?: string;
};

export const useLogListReq = () => {
  let [searchParams, _] = useSearchParams();

  const searchParamsStr = searchParams.toString();

  const loc = useLocation();

  if (searchParamsStr.length < 1) {
    return undefined;
  }

  let parsedQry = parseQueryString<{
    type?: string;
    mode: string;
    common?: {
      page?: number;
      itemsPerPage?: number;
    };
    namespaceRef?: objectRef;
    userRef?: objectRef;
    sessionRef?: objectRef;
    serviceRef?: objectRef;
    identityProviderRef?: objectRef;
    resourceRef?: objectRef;
    regionRef?: objectRef;
    deviceRef?: objectRef;
    policyRef?: objectRef;
  }>(searchParams.toString());

  if (parsedQry.common && parsedQry.common.page && parsedQry.common.page > 0) {
    parsedQry.common.page = parsedQry.common.page - 1;
  }

  return parsedQry;
};

export const toObjectRef = (arg?: objectRef): ObjectReference | undefined => {
  if (!arg) {
    return undefined;
  }
  return ObjectReference.create({ name: arg.name, uid: arg.uid });
};
