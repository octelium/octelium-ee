import { Outlet, RouteObject } from "react-router-dom";
import { ResourceComponentInfo } from "./types";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { getResourcePathKind } from "@/utils/pb";

export const toURLWithQry = (
  path: string,
  _params: Record<string, string> | string
): string => {
  return `${path}?${new URLSearchParams(_params).toString()}`;
};
