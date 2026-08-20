import Catalog from "./access/Catalog.json";
import Policy from "./access/Policy.json";
import Request from "./access/Request.json";
import Review from "./access/Review.json";

import { ResourceAccessName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceAccessName) => {
  return match(arg)
    .with("Catalog", () => Catalog)
    .with("Policy", () => Policy)
    .with("Request", () => Request)
    .with("Review", () => Review)
    .otherwise(() => undefined);
};
