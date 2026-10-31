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

export type ResourceInfoIcon = React.ComponentType<{
  size?: number | string;
  strokeWidth?: number | string;
  className?: string;
}>;

export interface ResourceInfoSection {
  value: string;
  label: string;
  icon?: React.ComponentType<{ size?: number | string }>;
  content: React.ReactNode;
}

export interface ResourceInfoGroup {
  title: string;
  icon?: ResourceInfoIcon;
  description?: string;
}

export interface ResourceMainInfo {
  items?: ResourceInfoMainItem[];
  status?: ResourceStatusInfo;
  actions?: React.ReactNode;
  groupOrder?: (string | ResourceInfoGroup)[];
  sections?: ResourceInfoSection[];
  header?: React.ReactNode;
}

export interface ResourceInfoMainItem {
  label: string;
  value: React.ReactNode;
  span?: "half" | "full";
  group?: string;
  primary?: boolean;
  icon?: ResourceInfoIcon;
  hint?: string;
  layout?: "inline" | "stacked";
}
