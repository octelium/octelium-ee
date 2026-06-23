import {
  ReviewerServiceClient,
  UserServiceClient,
} from "@/apis/accessv1/accessv1.client";
import { MainServiceClient as UserMainServiceClient } from "@/apis/userv1/userv1.client";
import { GrpcWebFetchTransport } from "@protobuf-ts/grpcweb-transport";
import * as AuthGRPC from "../../apis/authv1/authv1.client";

const getBaseUrl = (): string => window.location.origin;

let transport: GrpcWebFetchTransport | undefined;

const getTransport = (): GrpcWebFetchTransport => {
  if (!transport) {
    transport = new GrpcWebFetchTransport({
      baseUrl: getBaseUrl(),
      fetchInit: { credentials: "include" },
    });
  }
  return transport;
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
