import GitProvider from "./cordium/GitProvider.json";
import Membership from "./cordium/Membership.json";
import Region from "./cordium/Region.json";
import Secret from "./cordium/Secret.json";
import Space from "./cordium/Space.json";
import Template from "./cordium/Template.json";
import UserSecret from "./cordium/UserSecret.json";
import Workspace from "./cordium/Workspace.json";

import { ResourceCordiumName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceCordiumName) => {
  return match(arg)
    .with("Workspace", () => Workspace)
    .with("Template", () => Template)
    .with("Space", () => Space)
    .with("Membership", () => Membership)
    .with("GitProvider", () => GitProvider)
    .with("Secret", () => Secret)
    .with("UserSecret", () => UserSecret)
    .with("Region", () => Region)
    .otherwise(() => undefined);
};
