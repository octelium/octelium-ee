import {
  getAPI,
  Resource,
  ResourceCoreName,
  ResourceEnterpriseName,
} from "@/utils/pb";
import core from "./core";
import enterprise from "./enterprise";
import { match } from "ts-pattern";

export default (arg: Resource) => {
  return match(getAPI(arg))
    .with("core", () => core(arg.kind as ResourceCoreName))
    .with("enterprise", () => enterprise(arg.kind as ResourceEnterpriseName))
    .otherwise(() => undefined);
};
