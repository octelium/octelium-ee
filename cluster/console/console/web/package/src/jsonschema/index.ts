import {
  getAPI,
  Resource,
  ResourceCoreName,
  ResourceEnterpriseName,
} from "@/utils/pb";
import core from "./core";
import enterprise from "./enterprise";
import access from "./access";
import { match } from "ts-pattern";
import { ResourceAccessName } from "@/utils/pb";

export default (arg: Resource) => {
  return match(getAPI(arg))
    .with("core", () => core(arg.kind as ResourceCoreName))
    .with("enterprise", () => enterprise(arg.kind as ResourceEnterpriseName))
    .with("access", () => access(arg.kind as ResourceAccessName))
    .otherwise(() => undefined);
};
