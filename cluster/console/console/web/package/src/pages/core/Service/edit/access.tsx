import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import {
  Callout,
  ChoiceGroup,
  FormSection,
  ToggleList,
  ToggleRow,
} from "@/pages/enterprise/CollectorExporter/fields";
import * as CoreP from "@/apis/corev1/corev1";
import { Globe, ShieldCheck } from "lucide-react";
import * as React from "react";
import { Update } from "./fields";
import { applyExposure, pruneAuthorization, Spec } from "./model";
import { ModeInfo } from "./modes";

type Exposure = "private" | "public" | "anonymous";

const exposureOf = (spec: Spec): Exposure =>
  spec.isAnonymous ? "anonymous" : spec.isPublic ? "public" : "private";

const AuthorizationFields = (props: { spec: Spec; update: Update }) => {
  const { spec, update } = props;
  const auth =
    spec.authorization ?? CoreP.Service_Spec_Authorization.create();
  const commit = () => {
    spec.authorization = auth;
    pruneAuthorization(spec);
    update();
  };
  return (
    <>
      <SelectPolicies
        policies={auth.policies}
        onUpdate={(policies) => {
          auth.policies = policies ?? [];
          commit();
        }}
      />
      <SelectInlinePolicies
        inlinePolicies={auth.inlinePolicies}
        onUpdate={(inlinePolicies) => {
          auth.inlinePolicies = inlinePolicies;
          commit();
        }}
      />
    </>
  );
};

export const AccessForm = (props: {
  spec: Spec;
  info: ModeInfo;
  update: Update;
}) => {
  const { spec, info, update } = props;
  const memory = React.useRef<CoreP.Service_Spec_Authorization | undefined>(
    undefined,
  );
  const anonymous = spec.isAnonymous;
  const enforced = !!spec.authorization?.enableAnonymous;

  return (
    <>
      {info.isPublic && (
        <FormSection
          icon={Globe}
          title="Exposure"
          description="Who can reach this Service, and how."
        >
          <ChoiceGroup<Exposure>
            label="Exposure"
            hideLabel
            value={exposureOf(spec)}
            options={[
              {
                value: "private",
                label: "Private",
                hint: "Through a connected Octelium client",
              },
              {
                value: "public",
                label: "Public",
                hint: "Clientless access, with authentication",
              },
              ...(info.isAnonymous
                ? [
                    {
                      value: "anonymous" as const,
                      label: "Anonymous",
                      hint: "Open to anyone, without authentication",
                    },
                  ]
                : []),
            ]}
            onChange={(next) => {
              applyExposure(spec, next);
              update();
            }}
          />
          {anonymous && (
            <Callout title="Anyone can reach this Service without logging in">
              Authentication and authorization are off unless you evaluate
              Policies below.
            </Callout>
          )}
        </FormSection>
      )}

      <FormSection
        icon={ShieldCheck}
        title="Authorization"
        description="The Policies that decide who is allowed. A request that matches no rule is denied."
      >
        {anonymous && (
          <ToggleList>
            <ToggleRow
              label="Evaluate Policies for anonymous requests"
              description="Requests that match no rule are denied, even without an identity."
              checked={enforced}
              onChange={(checked) => {
                if (checked) {
                  const next =
                    memory.current ?? CoreP.Service_Spec_Authorization.create();
                  next.enableAnonymous = true;
                  spec.authorization = next;
                } else {
                  memory.current = spec.authorization;
                  spec.authorization = undefined;
                }
                update();
              }}
            />
          </ToggleList>
        )}
        {(!anonymous || enforced) && (
          <AuthorizationFields spec={spec} update={update} />
        )}
      </FormSection>
    </>
  );
};
