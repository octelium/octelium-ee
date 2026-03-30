import { API, Resource, ResourceName } from "@/utils/pb";

export type ResourceComponentInfo = {
  API: API;
  Kind: ResourceName;
  List: ResourceComponentInfoList;
  Item: ResourceComponentInfoItem;
  unCreatable?: boolean;
  unDeletable?: boolean;
  unEditable?: boolean;
};

export type ResourceComponentInfoList = {
  labelComponent?: (props: { item: Resource }) => React.ReactNode;
  extraComponent?: (props: { item: Resource }) => React.ReactNode;
  listFilter?: () => React.ReactNode;
  SummaryComponent?: (...args: any[]) => React.ReactNode;
  AccessLogComponent?: () => React.ReactNode;
};

export type ResourceComponentInfoItem = {
  Main?: (props: { item: Resource }) => React.ReactNode;
  Edit?: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  itemInfo?: (props: { item: Resource }) => React.ReactNode;

  createResource?: () => Resource;
};
