import { ObjectReference } from "@/apis/metav1/metav1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import {
  AccessListKind,
  getAccessListFilters,
} from "@/pages/access/listFilters";
import { getListOptionsPB, getResourceRef, Resource } from "@/utils/pb";
import { JsonObject } from "@protobuf-ts/runtime";

export type ResourceAccessSection = {
  value: string;
  label: string;
  kind: AccessListKind;
  refField: string;
  description: string;
};

const sections: Record<string, ResourceAccessSection[]> = {
  "core/v1/User": [
    {
      value: "requests",
      label: "Requests",
      kind: "Request",
      refField: "userRef",
      description: "Access requests created by this user.",
    },
    {
      value: "reviews",
      label: "Reviews",
      kind: "Review",
      refField: "userRef",
      description: "Access reviews by this user.",
    },
    {
      value: "subjects",
      label: "Subjects",
      kind: "Request",
      refField: "subjectUserRef",
      description:
        "Access requests whose subject is this user, including requests made on their behalf.",
    },
    {
      value: "reviewer",
      label: "Assigned requests",
      kind: "Request",
      refField: "reviewerRef",
      description:
        "Access requests whose review steps name this user as a reviewer.",
    },
    {
      value: "policies",
      label: "Policies",
      kind: "Policy",
      refField: "userRef",
      description: "Access policies that reference this user.",
    },
    {
      value: "identities",
      label: "Linked accounts",
      kind: "IntegrationIdentity",
      refField: "userRef",
      description: "Provider accounts linked to this user.",
    },
    {
      value: "deliveries",
      label: "Deliveries",
      kind: "IntegrationBinding",
      refField: "userRef",
      description: "Integration deliveries addressed to this user.",
    },
  ],
  "core/v1/Group": [
    {
      value: "policies",
      label: "Policies",
      kind: "Policy",
      refField: "groupRef",
      description: "Access policies that reference this group.",
    },
  ],
  "core/v1/Service": [
    {
      value: "requests",
      label: "Requests",
      kind: "Request",
      refField: "serviceRef",
      description: "Access requests for this service.",
    },
    {
      value: "policies",
      label: "Policies",
      kind: "Policy",
      refField: "serviceRef",
      description: "Access policies that reference this service.",
    },
    {
      value: "catalogs",
      label: "Catalogs",
      kind: "Catalog",
      refField: "serviceRef",
      description: "Access catalogs that include this service.",
    },
  ],
  "core/v1/Namespace": [
    {
      value: "catalogs",
      label: "Catalogs",
      kind: "Catalog",
      refField: "namespaceRef",
      description: "Access catalogs that include this namespace.",
    },
  ],
  "access/v1/Policy": [
    {
      value: "requests",
      label: "Requests",
      kind: "Request",
      refField: "policyRef",
      description: "Access requests that matched a rule of this policy.",
    },
  ],
  "access/v1/Catalog": [
    {
      value: "requests",
      label: "Requests",
      kind: "Request",
      refField: "catalogRef",
      description: "Access requests for this catalog.",
    },
    {
      value: "policies",
      label: "Policies",
      kind: "Policy",
      refField: "catalogRef",
      description: "Access policies that reference this catalog.",
    },
  ],
  "access/v1/Request": [
    {
      value: "reviews",
      label: "Reviews",
      kind: "Review",
      refField: "requestRef",
      description: "Reviews of this access request.",
    },
    {
      value: "deliveries",
      label: "Deliveries",
      kind: "IntegrationBinding",
      refField: "requestRef",
      description: "Integration deliveries presenting this access request.",
    },
  ],
  "access/v1/Integration": [
    {
      value: "deliveries",
      label: "Deliveries",
      kind: "IntegrationBinding",
      refField: "integrationRef",
      description: "Access requests presented through this integration.",
    },
    {
      value: "identities",
      label: "Linked accounts",
      kind: "IntegrationIdentity",
      refField: "integrationRef",
      description: "Provider accounts linked through this integration.",
    },
  ],
};

export const getResourceAccessSections = (
  apiVersion: string,
  kind: string,
): ResourceAccessSection[] => sections[`${apiVersion}/${kind}`] ?? [];

export const buildResourceAccessListRequest = (
  resource: Resource,
  section: ResourceAccessSection,
  search: string,
) => {
  const params = new URLSearchParams(search);
  const page = Number(params.get("common.page"));
  const itemsPerPage = Number(params.get("common.itemsPerPage"));
  const common = CommonListOptions.fromJson({
    page:
      Number.isFinite(page) && page >= 1
        ? Math.min(Math.floor(page), 100000) - 1
        : 0,
    itemsPerPage: [10, 25, 50, 100].includes(itemsPerPage) ? itemsPerPage : 25,
    query: params.get("common.query") ?? "",
    orderBy: {
      type:
        params.get("common.orderBy.type") === "NAME" ? "NAME" : "CREATED_AT",
      mode: params.get("common.orderBy.mode") === "ASC" ? "ASC" : "DESC",
    },
  });
  const json: JsonObject = { common: CommonListOptions.toJson(common) };
  const filterParams = new URLSearchParams();
  let invalidFilters = false;

  for (const filter of getAccessListFilters(section.kind)) {
    const value = params.get(filter.key);
    if (!value) continue;
    if (filter.options?.some((option) => option.value === value)) {
      json[filter.key] = value;
      filterParams.set(filter.key, value);
    } else if (!filter.options && (value === "true" || value === "false")) {
      json[filter.key] = value === "true";
      if (value === "true") filterParams.set(filter.key, value);
    } else {
      invalidFilters = true;
    }
  }

  json[section.refField] = ObjectReference.toJson(getResourceRef(resource));
  const request = getListOptionsPB("access", section.kind).fromJson(json);
  return { request, filterParams, invalidFilters };
};
