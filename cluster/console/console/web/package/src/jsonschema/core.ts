import Service from "./core/Service.json";
import Device from "./core/Device.json";
import User from "./core/User.json";
import Session from "./core/Session.json";
import ClusterConfig from "./core/ClusterConfig.json";
import Policy from "./core/Policy.json";
import Namespace from "./core/Namespace.json";
import Group from "./core/Group.json";
import Region from "./core/Region.json";
import Gateway from "./core/Gateway.json";
import Secret from "./core/Secret.json";
import Credential from "./core/Credential.json";
import IdentityProvider from "./core/IdentityProvider.json";

import { ResourceCoreName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceCoreName) => {
  return match(arg)
    .with("Group", () => Group)
    .with("User", () => User)
    .with("Service", () => Service)
    .with("Namespace", () => Namespace)
    .with("Device", () => Device)
    .with("Gateway", () => Gateway)
    .with("Region", () => Region)
    .with("Credential", () => Credential)
    .with("ClusterConfig", () => ClusterConfig)
    .with("Policy", () => Policy)
    .with("Session", () => Session)
    .with("Secret", () => Secret)
    .with("IdentityProvider", () => IdentityProvider)
    .otherwise(() => undefined);
};
