import {
  ReviewerServiceClient,
  UserServiceClient,
} from "@/apis/accessv1/accessv1.client";
import { MainServiceClient as UserMainServiceClient } from "@/apis/userv1/userv1.client";
import * as grpcWeb from "@protobuf-ts/grpcweb-transport";
import { getDomain, isDev } from "..";
import * as AuthGRPC from "../../apis/authv1/authv1.client";

export const getTransport = () => {
  const domain = getDomain();
  const scheme = location.protocol === "https:" ? "https" : "http";

  let baseUrl = `${scheme}://octelium-api.${domain}`;

  if (isDev()) {
    baseUrl = `http://${window.location.host}`;
  }

  return new grpcWeb.GrpcWebFetchTransport({
    baseUrl,

    fetchInit: {
      credentials: "include",
    },
  });
};

export const getUserClient = (): UserServiceClient =>
  new UserServiceClient(getTransport());

export const getReviewerClient = (): ReviewerServiceClient =>
  new ReviewerServiceClient(getTransport());

export const getClientAuth = (): AuthGRPC.MainServiceClient => {
  return new AuthGRPC.MainServiceClient(getTransport());
};

export const getUserMainClient = (): UserMainServiceClient =>
  new UserMainServiceClient(getTransport());
