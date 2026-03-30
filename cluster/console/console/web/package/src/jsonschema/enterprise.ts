import ClusterConfig from "./enterprise/ClusterConfig.json";
import Secret from "./enterprise/Secret.json";

import { ResourceEnterpriseName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceEnterpriseName) => {
  return match(arg)
    .with("ClusterConfig", () => ClusterConfig)
    .with("Secret", () => Secret)
    .otherwise(() => undefined);
};
