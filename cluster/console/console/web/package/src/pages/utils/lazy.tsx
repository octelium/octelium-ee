import { Resource } from "@/utils/pb";
import * as React from "react";
import { ResourceInfoComponent, ResourceMainInfo } from "./types";

type Loader<T> = () => Promise<T>;

export const lazySpec = (
  loader: Loader<{ default: React.ComponentType<any> }>,
): React.ComponentType<any> => React.lazy(loader);

export const lazyNamed = (
  loader: Loader<Record<string, any>>,
  name: string,
): React.ComponentType<any> =>
  React.lazy(async () => {
    const mod = await loader();
    return { default: mod[name] as React.ComponentType<any> };
  });

export const lazyResourceInfo = (
  loader: Loader<{ MainInfo: (props: any) => ResourceMainInfo }>,
): ResourceInfoComponent =>
  React.lazy(async () => {
    const mod = await loader();
    const Info = (props: {
      item: Resource;
      children: (info: ResourceMainInfo) => React.ReactNode;
    }) => <>{props.children(mod.MainInfo({ item: props.item }))}</>;
    return { default: Info };
  });
