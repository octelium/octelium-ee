import Catalog from "./access/Catalog.json";
import Integration from "./access/Integration.json";
import IntegrationBinding from "./access/IntegrationBinding.json";
import IntegrationIdentity from "./access/IntegrationIdentity.json";
import Policy from "./access/Policy.json";
import Request from "./access/Request.json";
import Review from "./access/Review.json";
import Secret from "./access/Secret.json";

import { ResourceAccessName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceAccessName) => {
  return match(arg)
    .with("Catalog", () => Catalog)
    .with("Policy", () => Policy)
    .with("Request", () => Request)
    .with("Review", () => Review)
    .with("Secret", () => Secret)
    .with("Integration", () => Integration)
    .with("IntegrationIdentity", () => IntegrationIdentity)
    .with("IntegrationBinding", () => IntegrationBinding)
    .otherwise(() => undefined);
};
