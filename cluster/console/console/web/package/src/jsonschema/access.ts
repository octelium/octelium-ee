import Catalog from "./access/Catalog.json";
import ClusterConfig from "./access/ClusterConfig.json";
import Integration from "./access/Integration.json";
import Policy from "./access/Policy.json";
import Request from "./access/Request.json";
import Review from "./access/Review.json";
import Secret from "./access/Secret.json";

import { ResourceAccessName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceAccessName) => {
  return match(arg)
    .with("ClusterConfig", () => ClusterConfig)
    .with("Catalog", () => Catalog)
    .with("Policy", () => Policy)
    .with("Request", () => Request)
    .with("Review", () => Review)
    .with("Secret", () => Secret)
    .with("Integration", () => Integration)
    .otherwise(() => undefined);
};
