import { useSearchParams } from "react-router-dom";
import { parseQueryString } from "../ResourceLayout/queryParse";
import { ObjectReference } from "@/apis/metav1/metav1";
import type { AccessLogStatusFilter } from "./utils";

type objectRef = {
  uid?: string;
  name?: string;
};

export const useLogListReq = () => {
  let [searchParams, _] = useSearchParams();

  const searchParamsStr = searchParams.toString();

  if (searchParamsStr.length < 1) {
    return undefined;
  }

  let parsedQry = parseQueryString<{
    type?: string;
    mode: string;
    common?: {
      page?: number;
      itemsPerPage?: number;
      query?: string;
    };
    namespaceRef?: objectRef;
    userRef?: objectRef;
    sessionRef?: objectRef;
    serviceRef?: objectRef;
    identityProviderRef?: objectRef;
    credentialRef?: objectRef;
    authenticatorRef?: objectRef;
    resourceRef?: objectRef;
    regionRef?: objectRef;
    deviceRef?: objectRef;
    policyRef?: objectRef;
    status?: string;
    level?: string;
    component?: {
      namespace?: string;
      type?: string;
      uid?: string;
    };
  }>(searchParams.toString());

  if (parsedQry.common && parsedQry.common.page && parsedQry.common.page > 0) {
    parsedQry.common.page = parsedQry.common.page - 1;
  }

  const normalizedStatus = parsedQry.status?.toLowerCase();
  if (
    normalizedStatus === "all" ||
    normalizedStatus === "allowed" ||
    normalizedStatus === "denied"
  ) {
    parsedQry.status = normalizedStatus as AccessLogStatusFilter;
  }

  return parsedQry;
};

export const toObjectRef = (arg?: objectRef): ObjectReference | undefined => {
  if (!arg) {
    return undefined;
  }
  return ObjectReference.create({ name: arg.name, uid: arg.uid });
};
