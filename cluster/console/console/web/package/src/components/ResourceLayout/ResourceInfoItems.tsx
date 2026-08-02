import { ResourceMainInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";

type ResourceInfoGetter = (props: { item: Resource }) => ResourceMainInfo;

/**
 * Gives resource info providers their own React render boundary. Providers are
 * called unconditionally inside this component, so legacy providers that use
 * hooks do not alter the hook order of the generic list or main-page layouts.
 */
const ResourceInfoItems = (props: {
  getter: ResourceInfoGetter;
  item: Resource;
  children: (items: NonNullable<ResourceMainInfo["items"]>) => React.ReactNode;
}) => props.children(props.getter({ item: props.item }).items ?? []);

export default ResourceInfoItems;
