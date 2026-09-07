import { API, Resource, ResourceName } from "@/utils/pb";

export type ResourceComponentInfo = {
  API: API;
  Kind: ResourceName;
  List: ResourceComponentInfoList;
  Item: ResourceComponentInfoItem;
  unCreatable?: boolean;
  unDeletable?: boolean;
  unEditable?: boolean;
  readOnlyEdit?: boolean;
  cloneable?: boolean;
  infoItemsGetter?: ResourceInfoComponent;
};

export type ResourceComponentInfoList = {
  labelComponent?: React.ComponentType<{ item: Resource }>;
  SummaryComponent?: React.ComponentType<any>;
};

export type ResourceComponentInfoItem = {
  hasMain?: boolean;
  MainAction?: React.ComponentType<{ item: Resource }>;
  Edit?: React.ComponentType<{
    item: Resource;
    onUpdate: (item: Resource) => void;
  }>;
  createResource?: () => Resource;
};

export type ResourceInfoComponent = React.ComponentType<{
  item: Resource;
  children: (info: ResourceMainInfo) => React.ReactNode;
}>;

export type ResourceStatusTone =
  "neutral" | "success" | "warning" | "danger" | "info";

export interface ResourceStatusInfo {
  label: string;
  tone?: ResourceStatusTone;
  hint?: React.ReactNode;
  control?: React.ReactNode;
}

export interface ResourceMainInfo {
  items?: ResourceInfoMainItem[];
  status?: ResourceStatusInfo;
  actions?: React.ReactNode;
  groupOrder?: string[];
}

export interface ResourceInfoMainItem {
  label: string;
  value: React.ReactNode;
  span?: "half" | "full";
  group?: string;
  primary?: boolean;
  hint?: string;
}
