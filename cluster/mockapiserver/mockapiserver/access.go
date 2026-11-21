// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockAccessRequest struct {
	name          string
	age           time.Duration
	requester     string
	subject       string
	service       string
	catalog       string
	policy        string
	rule          int
	urgency       accessv1.Request_Spec_Urgency
	justification string
	duration      time.Duration
	states        []accessv1.Request_Status_State_Status
	decision      time.Duration
	accessFor     time.Duration
	deadline      time.Duration
	origin        accessv1.Origin_Type
	integration   string
	reviewers     []string
	reject        bool
	ticket        string
	startIn       time.Duration
	askedInfo     string
}

func genAccessResources(ctx context.Context, octeliumC octeliumc.ClientInterface, adminSrv *admin.Server) error {
	zap.L().Debug("Generating access resources")

	now := time.Now().UTC()

	groups := map[string]*corev1.Group{}
	for _, itm := range []struct {
		name        string
		displayName string
	}{
		{"sre-oncall", "SRE on-call"},
		{"security", "Security team"},
		{"db-admins", "Database administrators"},
		{"engineering", "Engineering"},
		{"contractors", "Contractors"},
	} {
		grp, err := adminSrv.CreateGroup(ctx, &corev1.Group{
			Metadata: &metav1.Metadata{
				Name:        itm.name,
				DisplayName: itm.displayName,
			},
			Spec: &corev1.Group_Spec{},
		})
		if err != nil {
			return err
		}
		groups[itm.name] = grp
	}

	users := map[string]*corev1.User{}
	for _, itm := range []struct {
		name        string
		displayName string
		groups      []string
	}{
		{"alice", "Alice Chen", []string{"sre-oncall"}},
		{"bob", "Bob Martin", []string{"engineering"}},
		{"carol", "Carol Diaz", []string{"sre-oncall", "db-admins"}},
		{"dave", "Dave Kim", []string{"security"}},
		{"erin", "Erin Lopez", []string{"engineering"}},
		{"frank", "Frank Osei", []string{"contractors"}},
		{"grace", "Grace Patel", []string{"security", "db-admins"}},
	} {
		usr, err := adminSrv.CreateUser(ctx, &corev1.User{
			Metadata: &metav1.Metadata{
				Name:        itm.name,
				DisplayName: itm.displayName,
				PicURL:      fmt.Sprintf("https://i.pravatar.cc/120?u=%s", itm.name),
			},
			Spec: &corev1.User_Spec{
				Type:   corev1.User_Spec_HUMAN,
				Email:  fmt.Sprintf("%s@acme.example.com", itm.name),
				Groups: itm.groups,
			},
		})
		if err != nil {
			return err
		}
		users[itm.name] = usr
	}

	services := map[string]*corev1.Service{}
	for _, itm := range []struct {
		name        string
		displayName string
		mode        corev1.Service_Spec_Mode
		port        uint32
	}{
		{"postgres-prod.db", "Production Postgres", corev1.Service_Spec_POSTGRES, 5432},
		{"billing-api.production", "Billing API", corev1.Service_Spec_HTTP, 8080},
		{"bastion.ssh", "Production bastion", corev1.Service_Spec_SSH, 22},
		{"k8s-prod.aws", "Production Kubernetes", corev1.Service_Spec_KUBERNETES, 6443},
		{"grafana.production", "Grafana", corev1.Service_Spec_WEB, 3000},
	} {
		svc, err := adminSrv.CreateService(ctx, &corev1.Service{
			Metadata: &metav1.Metadata{
				Name:        itm.name,
				DisplayName: itm.displayName,
			},
			Spec: &corev1.Service_Spec{
				Mode: itm.mode,
				Port: itm.port,
				Config: &corev1.Service_Spec_Config{
					Upstream: &corev1.Service_Spec_Config_Upstream{
						Type: &corev1.Service_Spec_Config_Upstream_Url{
							Url: "https://example.com",
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		services[itm.name] = svc
	}

	catalogs := map[string]*accessv1.Catalog{}
	for _, itm := range []struct {
		name         string
		displayName  string
		description  string
		services     []string
		namespaces   []string
		levels       []*accessv1.Catalog_Spec_AccessLevel
		audience     *accessv1.Catalog_Spec_Audience
		owners       *accessv1.Catalog_Spec_Owners
		requirements *accessv1.Catalog_Spec_Requirements
		partial      bool
		issues       []*accessv1.ConfigurationIssue
	}{
		{"production-support", "Production support",
			"Everything the on-call engineer needs during a production incident.",
			[]string{"billing-api.production", "postgres-prod.db"}, []string{"production"},
			[]*accessv1.Catalog_Spec_AccessLevel{
				restrictedLevel("read", "Read only",
					"View dashboards and query data without changing anything.",
					`ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]`, mockHours(8)),
				restrictedLevel("operate", "Read and write",
					"Everything in read only, plus the write operations needed to mitigate an incident.",
					`ctx.request.http.method != "DELETE"`, mockHours(4)),
			},
			nil,
			&accessv1.Catalog_Spec_Owners{Groups: []string{"sre-oncall"}},
			&accessv1.Catalog_Spec_Requirements{
				Justification: &accessv1.Catalog_Spec_Requirements_Justification{
					IsRequired: true,
					MinLength:  10,
				},
				Ticket: &accessv1.Catalog_Spec_Requirements_Ticket{
					Pattern: `^(INC|JIRA)-[0-9]+$`,
					Label:   "Incident or ticket",
				},
			},
			true, nil},
		{"database-admin", "Database administration",
			"Administrative access to every database Service.",
			nil, []string{"db"},
			[]*accessv1.Catalog_Spec_AccessLevel{
				{
					Name:        "admin",
					DisplayName: "Database administrator",
					Description: "Full administrative access to the database Services.",
					Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
						Full: &accessv1.Catalog_Spec_AccessLevel_Full{},
					},
					MaxDuration: mockHours(2),
					Verification: &accessv1.Catalog_Spec_AccessLevel_Verification{
						MinAAL: corev1.Session_Status_Authentication_Info_AAL2,
					},
				},
			},
			nil,
			&accessv1.Catalog_Spec_Owners{Users: []string{"carol"}},
			nil, false, nil},
		{"break-glass", "Break glass",
			"Emergency access to the whole production estate.",
			[]string{"bastion.ssh", "k8s-prod.aws"}, []string{"production", "db", "aws"},
			[]*accessv1.Catalog_Spec_AccessLevel{
				{
					Name:        "full",
					DisplayName: "Full access",
					Description: "Unrestricted access that also overrides the default-priority guardrails.",
					Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
						Full: &accessv1.Catalog_Spec_AccessLevel_Full{
							Priority: func() *int32 { p := int32(-2); return &p }(),
						},
					},
					MaxDuration: mockHours(1),
				},
			},
			&accessv1.Catalog_Spec_Audience{
				Groups: []string{"sre-oncall"},
				OnBehalf: &accessv1.Catalog_Spec_Audience_OnBehalf{
					Groups: []string{"sre-oncall"},
				},
			},
			&accessv1.Catalog_Spec_Owners{Groups: []string{"security"}},
			&accessv1.Catalog_Spec_Requirements{
				Justification: &accessv1.Catalog_Spec_Requirements_Justification{
					IsRequired: true,
					MinLength:  30,
				},
				Ticket: &accessv1.Catalog_Spec_Requirements_Ticket{
					IsRequired: true,
					Pattern:    `^INC-[0-9]+$`,
					Label:      "Incident",
				},
			},
			false,
			[]*accessv1.ConfigurationIssue{
				{
					Type:        accessv1.ConfigurationIssue_OVERRIDES_GUARDRAILS,
					Severity:    accessv1.ConfigurationIssue_WARNING,
					Message:     "The access level full takes precedence over the core Policy rules of the default priority, including their DENY rules",
					AccessLevel: "full",
				},
			}},
	} {
		cat, err := octeliumC.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
			Metadata: &metav1.Metadata{
				Name:        itm.name,
				DisplayName: itm.displayName,
				Description: itm.description,
			},
			Spec: &accessv1.Catalog_Spec{
				ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
					Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
						Services:   itm.services,
						Namespaces: itm.namespaces,
					},
				},
				AccessLevels:         itm.levels,
				Audience:             itm.audience,
				Owners:               itm.owners,
				Requirements:         itm.requirements,
				AllowPartialRequests: itm.partial,
			},
			Status: &accessv1.Catalog_Status{
				Issues:       itm.issues,
				TotalService: uint32(len(getMockCatalogServices(itm.services, itm.namespaces, services))),
				CheckedAt:    timestamppb.New(now),
			},
		})
		if err != nil {
			return err
		}
		catalogs[itm.name] = cat
	}

	for _, name := range []string{
		"slack-bot-token", "slack-signing-secret",
		"jira-api-token", "jira-webhook-secret",
		"siem-signing-secret",
	} {
		if _, err := octeliumC.AccessC().CreateSecret(ctx, &accessv1.Secret{
			Metadata: &metav1.Metadata{
				Name: name,
			},
			Spec:   &accessv1.Secret_Spec{},
			Status: &accessv1.Secret_Status{},
			Data: &accessv1.Secret_Data{
				Type: &accessv1.Secret_Data_Value{
					Value: utilrand.GetRandomString(32),
				},
			},
		}); err != nil {
			return err
		}
	}

	integrations := map[string]*accessv1.Integration{}
	for _, itm := range []*accessv1.Integration{
		{
			Metadata: &metav1.Metadata{
				Name:        "slack-access",
				DisplayName: "Slack · #access-reviews",
			},
			Spec: &accessv1.Integration_Spec{
				Type: &accessv1.Integration_Spec_Slack_{
					Slack: &accessv1.Integration_Spec_Slack{
						BotToken: &accessv1.Integration_Spec_Slack_BotToken{
							Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
								FromSecret: "slack-bot-token",
							},
						},
						SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
							Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
								FromSecret: "slack-signing-secret",
							},
						},
						ChannelID:          "C05ACCESS42",
						MentionUserGroupID: "S05ONCALL01",
					},
				},
			},
			Status: &accessv1.Integration_Status{
				Id:    utilrand.GetRandomStringCanonical(24),
				Type:  accessv1.Integration_Status_SLACK,
				State: accessv1.Integration_Status_READY,
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_NOTIFICATION,
					accessv1.Integration_Status_DIRECT_USER_DELIVERY,
					accessv1.Integration_Status_INTERACTIVE_REVIEW,
					accessv1.Integration_Status_REQUEST_CREATION,
					accessv1.Integration_Status_IDENTITY_RESOLUTION,
					accessv1.Integration_Status_PRESENTATION_UPDATE,
				},
				ExternalTenantID:   "T024BE7LD",
				ExternalTenantName: "Acme Corp",
				LastSuccessAt:      timestamppb.New(now.Add(-4 * time.Minute)),
				Synchronization: &accessv1.Integration_Status_Synchronization{
					CreatedAt:   timestamppb.New(now.Add(-6 * time.Hour)),
					State:       accessv1.Integration_Status_Synchronization_SUCCESS,
					CompletedAt: timestamppb.New(now.Add(-6*time.Hour + 2*time.Second)),
				},
			},
		},
		{
			Metadata: &metav1.Metadata{
				Name:        "jira-access",
				DisplayName: "Jira · ACCESS project",
			},
			Spec: &accessv1.Integration_Spec{
				Type: &accessv1.Integration_Spec_Jira_{
					Jira: &accessv1.Integration_Spec_Jira{
						Url:   "https://acme.atlassian.net",
						Email: "octelium-bot@acme.example.com",
						ApiToken: &accessv1.Integration_Spec_Jira_APIToken{
							Type: &accessv1.Integration_Spec_Jira_APIToken_FromSecret{
								FromSecret: "jira-api-token",
							},
						},
						WebhookSecret: &accessv1.Integration_Spec_Jira_WebhookSecret{
							Type: &accessv1.Integration_Spec_Jira_WebhookSecret_FromSecret{
								FromSecret: "jira-webhook-secret",
							},
						},
						ProjectKey:    "ACCESS",
						IssueTypeName: "Task",
						ApproveStatus: "Approved",
						RejectStatus:  "Declined",
					},
				},
			},
			Status: &accessv1.Integration_Status{
				Id:    utilrand.GetRandomStringCanonical(24),
				Type:  accessv1.Integration_Status_JIRA,
				State: accessv1.Integration_Status_READY,
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_NOTIFICATION,
					accessv1.Integration_Status_IDENTITY_RESOLUTION,
					accessv1.Integration_Status_PRESENTATION_UPDATE,
					accessv1.Integration_Status_INTERACTIVE_REVIEW,
				},
				ExternalTenantID:   "acme.atlassian.net",
				ExternalTenantName: "Octelium Bot",
				LastSuccessAt:      timestamppb.New(now.Add(-35 * time.Minute)),
				Synchronization: &accessv1.Integration_Status_Synchronization{
					CreatedAt:   timestamppb.New(now.Add(-26 * time.Hour)),
					State:       accessv1.Integration_Status_Synchronization_SUCCESS,
					CompletedAt: timestamppb.New(now.Add(-26*time.Hour + 3*time.Second)),
				},
			},
		},
		{
			Metadata: &metav1.Metadata{
				Name:        "siem-webhook",
				DisplayName: "SIEM audit webhook",
			},
			Spec: &accessv1.Integration_Spec{
				Type: &accessv1.Integration_Spec_Webhook_{
					Webhook: &accessv1.Integration_Spec_Webhook{
						Url:  "https://hooks.acme.example.com/octelium/access",
						Name: "siem",
						SigningSecret: &accessv1.Integration_Spec_Webhook_SigningSecret{
							Type: &accessv1.Integration_Spec_Webhook_SigningSecret_FromSecret{
								FromSecret: "siem-signing-secret",
							},
						},
					},
				},
			},
			Status: &accessv1.Integration_Status{
				Id:    utilrand.GetRandomStringCanonical(24),
				Type:  accessv1.Integration_Status_WEBHOOK,
				State: accessv1.Integration_Status_DEGRADED,
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_NOTIFICATION,
					accessv1.Integration_Status_PRESENTATION_UPDATE,
				},
				ExternalTenantID:   "hooks.acme.example.com",
				ExternalTenantName: "hooks.acme.example.com",
				LastSuccessAt:      timestamppb.New(now.Add(-3 * time.Hour)),
				LastFailureAt:      timestamppb.New(now.Add(-12 * time.Minute)),
				LastError:          "The webhook returned the status code 503",
				Synchronization: &accessv1.Integration_Status_Synchronization{
					CreatedAt:   timestamppb.New(now.Add(-2 * time.Hour)),
					State:       accessv1.Integration_Status_Synchronization_SUCCESS,
					CompletedAt: timestamppb.New(now.Add(-2*time.Hour + time.Second)),
				},
			},
		},
	} {
		integration, err := octeliumC.AccessC().CreateIntegration(ctx, itm)
		if err != nil {
			return err
		}
		integrations[itm.Metadata.Name] = integration
	}

	surface := func(integration string,
		audience accessv1.Policy_Spec_Rule_Surface_Destination_Audience,
		mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
		return &accessv1.Policy_Spec_Rule_Surface{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				IntegrationRef: umetav1.GetObjectReference(integrations[integration]),
				Audience:       audience,
			},
			InteractionMode: mode,
		}
	}

	groupReviewer := func(name string) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
		return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{
				Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{
					GroupRef: umetav1.GetObjectReference(groups[name]),
				},
			},
		}
	}

	userReviewer := func(name string) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
		return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
				User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
					UserRef: umetav1.GetObjectReference(users[name]),
				},
			},
		}
	}

	catalogCondition := func(name string) *accessv1.Policy_Spec_Rule_Condition {
		return &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
				Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
					Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{
						CatalogRef: umetav1.GetObjectReference(catalogs[name]),
					},
				},
			},
		}
	}

	hours := func(h uint32) *metav1.Duration {
		return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: h}}
	}

	minutes := func(m uint32) *metav1.Duration {
		return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: m}}
	}

	policies := map[string]*accessv1.Policy{}
	for _, itm := range []*accessv1.Policy{
		{
			Metadata: &metav1.Metadata{
				Name:        "production-access",
				DisplayName: "Production access",
				Description: "Just-in-time access to the production support Catalog.",
			},
			Spec: &accessv1.Policy_Spec{
				Rules: []*accessv1.Policy_Spec_Rule{
					{
						Name:     "oncall-auto-approve",
						Priority: -1,
						Condition: &accessv1.Policy_Spec_Rule_Condition{
							Type: &accessv1.Policy_Spec_Rule_Condition_All_{
								All: &accessv1.Policy_Spec_Rule_Condition_All{
									Of: []*accessv1.Policy_Spec_Rule_Condition{
										{
											Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
												Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
													Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{
														GroupRef: umetav1.GetObjectReference(groups["sre-oncall"]),
													},
												},
											},
										},
										catalogCondition("production-support"),
									},
								},
							},
						},
						Effect:      accessv1.Policy_Spec_Rule_AUTO_APPROVE,
						MaxDuration: hours(4),
						Message:     "On-call engineers get production support access without a review.",
						Notifications: []*accessv1.Policy_Spec_Rule_Surface{
							surface("slack-access", accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
								accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
						},
					},
					{
						Name:      "production-review",
						Condition: catalogCondition("production-support"),
						Effect:    accessv1.Policy_Spec_Rule_REVIEW,
						Action: &accessv1.Policy_Spec_Rule_Action{
							Type: &accessv1.Policy_Spec_Rule_Action_Review_{
								Review: &accessv1.Policy_Spec_Rule_Action_Review{
									Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
										{
											Name:                "team-lead",
											Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{groupReviewer("sre-oncall")},
											ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
											Timeout:             minutes(30),
											OnTimeout:           accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP,
											Surfaces: []*accessv1.Policy_Spec_Rule_Surface{
												surface("slack-access", accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
													accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
												surface("slack-access", accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
													accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
											},
										},
										{
											Name:                "security",
											Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{groupReviewer("security"), userReviewer("dave")},
											ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
											ApprovalCount:       1,
											Timeout:             hours(2),
											OnTimeout:           accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT,
											ReasonRequirement:   accessv1.Policy_Spec_Rule_Action_Review_Step_REJECT_ONLY,
											Escalation: &accessv1.Policy_Spec_Rule_Action_Review_Step_Escalation{
												After:     hours(1),
												Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{userReviewer("grace")},
											},
											Surfaces: []*accessv1.Policy_Spec_Rule_Surface{
												surface("jira-access", accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
													accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
											},
										},
									},
								},
							},
						},
						MaxDuration: hours(8),
						Notifications: []*accessv1.Policy_Spec_Rule_Surface{
							surface("slack-access", accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
								accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
						},
					},
				},
			},
			Status: &accessv1.Policy_Status{},
		},
		{
			Metadata: &metav1.Metadata{
				Name:        "database-access",
				DisplayName: "Database access",
			},
			Spec: &accessv1.Policy_Spec{
				Rules: []*accessv1.Policy_Spec_Rule{
					{
						Name:      "dba-review",
						Condition: catalogCondition("database-admin"),
						Effect:    accessv1.Policy_Spec_Rule_REVIEW,
						Action: &accessv1.Policy_Spec_Rule_Action{
							Type: &accessv1.Policy_Spec_Rule_Action_Review_{
								Review: &accessv1.Policy_Spec_Rule_Action_Review{
									Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
										{
											Name:                "dba",
											Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{groupReviewer("db-admins")},
											ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
											Timeout:             hours(4),
											SeparationOfDuties: &accessv1.Policy_Spec_Rule_SeparationOfDuties{
												AllowSubjectReview: false,
											},
										},
									},
								},
							},
						},
						MaxDuration: hours(2),
						Notifications: []*accessv1.Policy_Spec_Rule_Surface{
							surface("siem-webhook", accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
								accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
						},
					},
				},
			},
			Status: &accessv1.Policy_Status{},
		},
		{
			Metadata: &metav1.Metadata{
				Name:        "contractor-guardrails",
				DisplayName: "Contractor guardrails",
			},
			Spec: &accessv1.Policy_Spec{
				Rules: []*accessv1.Policy_Spec_Rule{
					{
						Name:     "deny-contractors",
						Priority: -4,
						Condition: &accessv1.Policy_Spec_Rule_Condition{
							Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
								Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
									Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{
										GroupRef: umetav1.GetObjectReference(groups["contractors"]),
									},
								},
							},
						},
						Effect:  accessv1.Policy_Spec_Rule_DENY,
						Message: "Contractors cannot request production access. Ask your Octelium contact.",
					},
				},
			},
			Status: &accessv1.Policy_Status{},
		},
		{
			Metadata: &metav1.Metadata{
				Name:        "break-glass",
				DisplayName: "Break glass",
			},
			Spec: &accessv1.Policy_Spec{
				IsDisabled: true,
				Rules: []*accessv1.Policy_Spec_Rule{
					{
						Name: "highest-urgency",
						Condition: &accessv1.Policy_Spec_Rule_Condition{
							Type: &accessv1.Policy_Spec_Rule_Condition_Match{
								Match: `ctx.request.spec.urgency == "HIGHEST"`,
							},
						},
						Effect:      accessv1.Policy_Spec_Rule_AUTO_APPROVE,
						MaxDuration: hours(1),
					},
				},
			},
			Status: &accessv1.Policy_Status{},
		},
	} {
		pol, err := octeliumC.AccessC().CreatePolicy(ctx, itm)
		if err != nil {
			return err
		}
		policies[itm.Metadata.Name] = pol
	}

	const (
		pending   = accessv1.Request_Status_State_PENDING
		approved  = accessv1.Request_Status_State_APPROVED
		rejected  = accessv1.Request_Status_State_REJECTED
		expired   = accessv1.Request_Status_State_EXPIRED
		revoked   = accessv1.Request_Status_State_REVOKED
		cancelled = accessv1.Request_Status_State_CANCELLED
	)

	reasons := []string{
		"Investigating elevated 5xx rates on the billing API (INC-4821).",
		"Rotating the replication credentials as part of the quarterly maintenance.",
		"Need to run a one-off data fix for the invoice backfill (JIRA-2219).",
		"Debugging a failed deployment on the production cluster.",
		"Customer escalation: verifying a payment record.",
		"Applying the approved schema migration for release 4.12.",
		"",
	}

	var items []*mockAccessRequest
	{
		requesters := []string{"bob", "erin", "alice", "bob", "erin", "grace", "frank"}
		for i := 0; i < 34; i++ {
			requester := requesters[i%len(requesters)]
			age := time.Duration(7*24*60-i*290-utilrand.GetRandomRangeMath(0, 180)) * time.Minute
			itm := &mockAccessRequest{
				name:          fmt.Sprintf("req-%s", utilrand.GetRandomStringCanonical(6)),
				age:           age,
				requester:     requester,
				subject:       requester,
				urgency:       accessv1.Request_Spec_Urgency(1 + i%6),
				justification: reasons[i%len(reasons)],
				duration:      time.Duration(1+i%4) * time.Hour,
				origin:        accessv1.Origin_API,
			}

			switch {
			case requester == "frank":
				itm.catalog = "production-support"
				itm.policy = "contractor-guardrails"
				itm.states = []accessv1.Request_Status_State_Status{rejected}
			case requester == "alice":
				itm.catalog = "production-support"
				itm.policy = "production-access"
				itm.states = []accessv1.Request_Status_State_Status{approved, expired}
				itm.accessFor = 4 * time.Hour
			case i%5 == 0:
				itm.catalog = "database-admin"
				itm.policy = "database-access"
				itm.reviewers = []string{"carol"}
				itm.decision = time.Duration(8+i%40) * time.Minute
				itm.states = []accessv1.Request_Status_State_Status{pending, approved, expired}
				itm.accessFor = 2 * time.Hour
				itm.integration = "siem-webhook"
			case i%7 == 3:
				itm.catalog = "break-glass"
				itm.states = []accessv1.Request_Status_State_Status{rejected}
			case i%4 == 1:
				itm.catalog = "production-support"
				itm.policy = "production-access"
				itm.rule = 1
				itm.reviewers = []string{"alice", "dave"}
				itm.decision = time.Duration(12+i%70) * time.Minute
				itm.states = []accessv1.Request_Status_State_Status{pending, approved, expired}
				itm.accessFor = 6 * time.Hour
				itm.integration = "slack-access"
				if i%3 == 0 {
					itm.origin = accessv1.Origin_INTEGRATION
				}
			case i%4 == 2:
				itm.catalog = "production-support"
				itm.policy = "production-access"
				itm.rule = 1
				itm.reviewers = []string{"carol"}
				itm.decision = time.Duration(5+i%30) * time.Minute
				itm.states = []accessv1.Request_Status_State_Status{pending, rejected}
				itm.reject = true
				itm.integration = "slack-access"
			default:
				itm.catalog = "production-support"
				itm.policy = "production-access"
				itm.rule = 1
				itm.states = []accessv1.Request_Status_State_Status{pending, cancelled}
			}

			items = append(items, itm)
		}
	}

	items = append(items,
		&mockAccessRequest{
			name: "req-incident-4821", age: 35 * time.Minute, requester: "erin", subject: "erin",
			catalog: "production-support", policy: "production-access", rule: 1, ticket: "INC-4821",
			urgency: accessv1.Request_Spec_HIGHEST, justification: reasons[0],
			duration: 2 * time.Hour, deadline: -5 * time.Minute,
			states: []accessv1.Request_Status_State_Status{pending}, origin: accessv1.Origin_INTEGRATION,
			integration: "slack-access",
		},
		&mockAccessRequest{
			name: "req-migration-412", age: 3 * time.Hour, requester: "bob", subject: "bob",
			catalog: "database-admin", policy: "database-access",
			urgency: accessv1.Request_Spec_HIGH, justification: reasons[5],
			duration: time.Hour, states: []accessv1.Request_Status_State_Status{pending},
			origin: accessv1.Origin_API, integration: "siem-webhook",
			askedInfo: "Which tables does the migration touch, and is there a rollback plan?",
		},
		&mockAccessRequest{
			name: "req-scheduled-window", age: 2 * time.Hour, requester: "erin", subject: "erin",
			catalog: "production-support", policy: "production-access", rule: 1,
			urgency: accessv1.Request_Spec_NORMAL, justification: reasons[5], ticket: "JIRA-2304",
			duration: 2 * time.Hour, states: []accessv1.Request_Status_State_Status{pending, approved},
			reviewers: []string{"alice", "dave"}, decision: 30 * time.Minute, startIn: 5 * time.Hour,
			origin: accessv1.Origin_API,
		},
		&mockAccessRequest{
			name: "req-stale-backfill", age: 31 * time.Hour, requester: "erin", subject: "bob",
			catalog: "database-admin", policy: "database-access",
			urgency: accessv1.Request_Spec_NORMAL, justification: reasons[2],
			duration: 4 * time.Hour, states: []accessv1.Request_Status_State_Status{pending},
			origin: accessv1.Origin_API,
		},
		&mockAccessRequest{
			name: "req-security-step", age: 50 * time.Minute, requester: "bob", subject: "bob",
			catalog: "production-support", policy: "production-access", rule: 1,
			urgency: accessv1.Request_Spec_VERY_HIGH, justification: reasons[3],
			duration: 3 * time.Hour, states: []accessv1.Request_Status_State_Status{pending},
			reviewers: []string{"carol"}, origin: accessv1.Origin_API, integration: "jira-access",
		},
		&mockAccessRequest{
			name: "req-oncall-live", age: 25 * time.Minute, requester: "alice", subject: "alice",
			catalog: "production-support", policy: "production-access",
			urgency: accessv1.Request_Spec_HIGH, justification: reasons[0],
			duration: time.Hour, states: []accessv1.Request_Status_State_Status{approved},
			accessFor: time.Hour, origin: accessv1.Origin_INTEGRATION, integration: "slack-access",
		},
		&mockAccessRequest{
			name: "req-dba-live", age: 90 * time.Minute, requester: "grace", subject: "grace",
			catalog: "database-admin", policy: "database-access",
			urgency: accessv1.Request_Spec_NORMAL, justification: reasons[1],
			duration: 2 * time.Hour, states: []accessv1.Request_Status_State_Status{pending, approved},
			reviewers: []string{"carol"}, decision: 18 * time.Minute, accessFor: 2 * time.Hour,
			origin: accessv1.Origin_API, integration: "siem-webhook",
		},
		&mockAccessRequest{
			name: "req-prod-live", age: 4 * time.Hour, requester: "erin", subject: "erin",
			catalog: "production-support", policy: "production-access", rule: 1,
			urgency: accessv1.Request_Spec_NORMAL, justification: reasons[4],
			duration: 6 * time.Hour, states: []accessv1.Request_Status_State_Status{pending, approved},
			reviewers: []string{"alice", "dave"}, decision: 42 * time.Minute, accessFor: 6 * time.Hour,
			origin: accessv1.Origin_API, integration: "slack-access",
		},
		&mockAccessRequest{
			name: "req-revoked", age: 2 * time.Hour, requester: "bob", subject: "bob",
			catalog: "production-support", policy: "production-access", rule: 1,
			urgency: accessv1.Request_Spec_LOW, justification: reasons[3],
			duration: time.Hour, states: []accessv1.Request_Status_State_Status{pending, approved, revoked},
			reviewers: []string{"carol", "grace"}, decision: 9 * time.Minute, accessFor: time.Hour,
			origin: accessv1.Origin_API, integration: "slack-access",
		},
	)

	for _, itm := range items {
		if err := genMockAccessRequest(ctx, octeliumC, now, itm, users, services, catalogs, policies, integrations); err != nil {
			return err
		}
	}

	for _, itm := range []struct {
		integration string
		user        string
		externalID  string
		username    string
	}{
		{"slack-access", "alice", "U04ALICE01", "alice.chen"},
		{"slack-access", "carol", "U04CAROL02", "carol"},
		{"slack-access", "dave", "U04DAVE003", "dave.kim"},
		{"slack-access", "erin", "U04ERIN004", "erin"},
		{"jira-access", "dave", "5b10ac8d82e05b22cc7d4ef5", "Dave Kim"},
		{"jira-access", "grace", "5b10ac8d82e05b22cc7d4ef6", "Grace Patel"},
	} {
		if _, err := octeliumC.AccessC().CreateIntegrationIdentity(ctx, &accessv1.IntegrationIdentity{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(10),
			},
			Spec: &accessv1.IntegrationIdentity_Spec{},
			Status: &accessv1.IntegrationIdentity_Status{
				IntegrationRef:   umetav1.GetObjectReference(integrations[itm.integration]),
				UserRef:          umetav1.GetObjectReference(users[itm.user]),
				ExternalID:       itm.externalID,
				Source:           accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
				VerifiedAt:       timestamppb.New(now.Add(-time.Duration(utilrand.GetRandomRangeMath(1, 72)) * time.Hour)),
				ExternalUsername: itm.username,
				ExternalEmail:    fmt.Sprintf("%s@acme.example.com", itm.user),
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func genMockAccessRequest(ctx context.Context, octeliumC octeliumc.ClientInterface, now time.Time,
	itm *mockAccessRequest,
	users map[string]*corev1.User,
	services map[string]*corev1.Service,
	catalogs map[string]*accessv1.Catalog,
	policies map[string]*accessv1.Policy,
	integrations map[string]*accessv1.Integration) error {

	createdAt := now.Add(-itm.age)

	req := &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: itm.name,
		},
		Spec: &accessv1.Request_Spec{
			Urgency:       itm.urgency,
			Justification: itm.justification,
			Resource:      &accessv1.Request_Spec_Resource{},
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{
					UserRef: umetav1.GetObjectReference(users[itm.subject]),
				},
			},
			Duration: &metav1.Duration{
				Type: &metav1.Duration_Minutes{Minutes: uint32(itm.duration / time.Minute)},
			},
		},
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(users[itm.requester]),
			Origin: &accessv1.Origin{
				Type: itm.origin,
			},
		},
	}

	if itm.origin == accessv1.Origin_INTEGRATION && itm.integration != "" {
		req.Status.Origin.IntegrationRef = umetav1.GetObjectReference(integrations[itm.integration])
		req.Status.Origin.ExternalActorID = "U04ERIN004"
	}

	if itm.deadline != 0 {
		req.Spec.Deadline = timestamppb.New(now.Add(itm.deadline))
	}

	catalog := catalogs[itm.catalog]
	level := catalog.Spec.AccessLevels[0]
	if itm.catalog == "production-support" && itm.policy == "production-access" && itm.rule == 0 {
		level = catalog.Spec.AccessLevels[1]
	}

	req.Spec.Ticket = itm.ticket
	req.Spec.Resource.Type = &accessv1.Request_Spec_Resource_Catalog_{
		Catalog: &accessv1.Request_Spec_Resource_Catalog{
			CatalogRef:  umetav1.GetObjectReference(catalog),
			AccessLevel: level.Name,
		},
	}

	if itm.startIn > 0 {
		req.Spec.StartAt = timestamppb.New(now.Add(itm.startIn))
	}

	priority := accesscmd.GetAccessLevelPriority(level, 0)
	scope := &accessv1.Request_Status_ResolvedScope{
		CatalogRef:             umetav1.GetObjectReference(catalog),
		CatalogResourceVersion: catalog.Metadata.ResourceVersion,
		CatalogDisplayName:     catalog.Metadata.DisplayName,
		AccessLevel:            level,
		Membership:             accessv1.Catalog_Spec_FIXED,
		ResolvedAt:             timestamppb.New(createdAt),
		Priority:               priority,
		OverridesGuardrails:    priority < 0,
	}
	coll := catalog.Spec.ResourceCollection.Service
	for _, svc := range getMockCatalogServices(coll.Services, coll.Namespaces, services) {
		scope.ServiceRefs = append(scope.ServiceRefs, umetav1.GetObjectReference(svc))
	}

	req.Status.ResolvedScope = scope
	req.Status.SpecRevision = 1
	req.Status.RequesterInfo = accesscmd.GetUserInfo(users[itm.requester])
	req.Status.SubjectInfo = accesscmd.GetUserInfo(users[itm.subject])
	req.Status.Evaluation = &accessv1.Request_Status_Evaluation{
		State:         accessv1.Request_Status_Evaluation_EVALUATED,
		LastAttemptAt: timestamppb.New(createdAt),
	}

	var rule *accessv1.Policy_Spec_Rule
	if pol, ok := policies[itm.policy]; ok {
		req.Status.PolicyRef = umetav1.GetObjectReference(pol)
		rule = pol.Spec.Rules[itm.rule]
		req.Status.Rule = rule

		workflow, err := accesscmd.BuildWorkflow(ctx, octeliumC, rule)
		if err != nil {
			return err
		}
		req.Status.Workflow = workflow
	}

	effectiveDuration := itm.duration
	if maxDuration := accesscmd.GetMaxDuration(level, rule); effectiveDuration > maxDuration {
		effectiveDuration = maxDuration
	}
	if rule != nil && rule.Effect != accessv1.Policy_Spec_Rule_DENY {
		req.Status.EffectiveDuration = &metav1.Duration{
			Type: &metav1.Duration_Minutes{Minutes: uint32(effectiveDuration / time.Minute)},
		}
	}

	req.Status.ApprovalStartAt = timestamppb.New(createdAt)

	stateAt := createdAt
	var approvedAt time.Time
	for i, state := range itm.states {
		switch {
		case i == 0 && state != accessv1.Request_Status_State_PENDING:
			stateAt = createdAt
		case state == accessv1.Request_Status_State_APPROVED || state == accessv1.Request_Status_State_REJECTED:
			stateAt = createdAt.Add(itm.decision)
		case state == accessv1.Request_Status_State_CANCELLED:
			stateAt = createdAt.Add(7 * time.Minute)
		case state == accessv1.Request_Status_State_EXPIRED:
			stateAt = stateAt.Add(itm.accessFor)
		case state == accessv1.Request_Status_State_REVOKED:
			stateAt = stateAt.Add(itm.accessFor / 3)
		}

		if req.Status.State != nil {
			req.Status.LastStates = append([]*accessv1.Request_Status_State{req.Status.State}, req.Status.LastStates...)
		}

		req.Status.State = &accessv1.Request_Status_State{
			CreatedAt: timestamppb.New(stateAt),
			Status:    state,
		}
		setMockStateReason(req.Status.State, itm, rule, users)

		if state != accessv1.Request_Status_State_PENDING && req.Status.ApprovalEndAt == nil {
			req.Status.ApprovalEndAt = timestamppb.New(stateAt)
		}

		switch state {
		case accessv1.Request_Status_State_APPROVED:
			approvedAt = stateAt
			startsAt := stateAt
			if req.Spec.StartAt != nil && req.Spec.StartAt.AsTime().After(startsAt) {
				startsAt = req.Spec.StartAt.AsTime()
			}

			req.Status.AccessStartsAt = timestamppb.New(startsAt)
			req.Status.AccessEndsAt = timestamppb.New(startsAt.Add(effectiveDuration))
			req.Status.Grant = &accessv1.Request_Status_Grant{
				State:       accessv1.Request_Status_Grant_ACTIVE,
				ActivatedAt: timestamppb.New(startsAt),
			}

			if startsAt.After(now) {
				req.Status.Grant.State = accessv1.Request_Status_Grant_SCHEDULED
				req.Status.Grant.ActivatedAt = nil
			}

		case accessv1.Request_Status_State_EXPIRED, accessv1.Request_Status_State_REVOKED:
			if req.Status.Grant != nil {
				req.Status.Grant.State = accessv1.Request_Status_Grant_ENDED
				req.Status.Grant.EndedAt = timestamppb.New(stateAt)
			}
		}
	}

	if req.Status.State.Status == accessv1.Request_Status_State_PENDING && rule != nil &&
		rule.Action.GetReview() != nil {
		req.Status.Review = &accessv1.Request_Status_Review{
			CurrentStepStartedAt: timestamppb.New(createdAt),
		}
		if len(itm.reviewers) > 0 && len(rule.Action.GetReview().Steps) > 1 {
			req.Status.Review.CurrentStep = 1
			req.Status.Review.CurrentStepStartedAt = timestamppb.New(createdAt.Add(12 * time.Minute))
		}

		if itm.askedInfo != "" {
			req.Status.Review.InformationRequests = append(req.Status.Review.InformationRequests,
				&accessv1.Request_Status_Review_InformationRequest{
					UserRef:     umetav1.GetObjectReference(users["carol"]),
					Message:     itm.askedInfo,
					RequestedAt: timestamppb.New(createdAt.Add(20 * time.Minute)),
				})
		}
	} else if rule != nil && rule.Action.GetReview() != nil {
		req.Status.Review = &accessv1.Request_Status_Review{
			CurrentStepStartedAt: timestamppb.New(createdAt),
		}
	}

	if req.Status.Review != nil {
		if err := setMockStepStates(ctx, octeliumC, req, createdAt, approvedAt); err != nil {
			return err
		}
	}

	created, err := octeliumC.AccessC().CreateRequest(ctx, req)
	if err != nil {
		return err
	}

	for i, reviewer := range itm.reviewers {
		decision := accessv1.Review_Spec_DECISION_APPROVE
		if itm.reject && i == len(itm.reviewers)-1 {
			decision = accessv1.Review_Spec_DECISION_REJECT
		}

		stepIndex := int32(i)
		steps := rule.Action.GetReview().Steps
		if int(stepIndex) >= len(steps) {
			stepIndex = int32(len(steps) - 1)
		}

		setAt := createdAt.Add(itm.decision * time.Duration(i+1) / time.Duration(len(itm.reviewers)))
		if itm.decision == 0 {
			setAt = createdAt.Add(12 * time.Minute)
		}

		origin := &accessv1.Origin{Type: accessv1.Origin_API}
		if itm.integration == "slack-access" && i%2 == 0 {
			origin = &accessv1.Origin{
				Type:            accessv1.Origin_INTEGRATION,
				IntegrationRef:  umetav1.GetObjectReference(integrations["slack-access"]),
				ExternalActorID: "U04CAROL02",
			}
		}

		rev, err := octeliumC.AccessC().CreateReview(ctx, &accessv1.Review{
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("%s.%s", utilrand.GetRandomStringCanonical(4), created.Metadata.Name),
			},
			Spec: &accessv1.Review_Spec{
				Decision:      decision,
				Justification: mockReviewJustification(decision),
			},
			Status: &accessv1.Review_Status{
				UserRef:      umetav1.GetObjectReference(users[reviewer]),
				RequestRef:   umetav1.GetObjectReference(created),
				SetAt:        timestamppb.New(setAt),
				StepIndex:    stepIndex,
				StepName:     steps[stepIndex].Name,
				Origin:       origin,
				SpecRevision: 1,
			},
		})
		if err != nil {
			return err
		}

		rev.Metadata.CreatedAt = timestamppb.New(setAt)
		if _, err := octeliumC.AccessC().UpdateReview(ctx, rev); err != nil {
			return err
		}

		if created.Status.Review == nil {
			created.Status.Review = &accessv1.Request_Status_Review{}
		}

		created.Status.Review.LastSteps = append(created.Status.Review.LastSteps, &accessv1.Request_Status_Review_Step{
			ReviewRef:     umetav1.GetObjectReference(rev),
			SetAt:         timestamppb.New(setAt),
			StepIndex:     stepIndex,
			Decision:      decision,
			UserRef:       umetav1.GetObjectReference(users[reviewer]),
			StepName:      steps[stepIndex].Name,
			Justification: mockReviewJustification(decision),
			SpecRevision:  1,
		})
	}

	created.Metadata.CreatedAt = timestamppb.New(createdAt)
	created, err = octeliumC.AccessC().UpdateRequest(ctx, created)
	if err != nil {
		return err
	}

	if itm.integration == "" || rule == nil {
		return nil
	}

	integration := integrations[itm.integration]
	isClosed := created.Status.State.Status != accessv1.Request_Status_State_PENDING

	type binding struct {
		purpose  accessv1.IntegrationBinding_Status_Purpose
		audience accessv1.Policy_Spec_Rule_Surface_Destination_Audience
		mode     accessv1.Policy_Spec_Rule_Surface_InteractionMode
	}

	var bindings []binding
	stepName := ""

	switch itm.integration {
	case "slack-access":
		stepName = "team-lead"
		bindings = append(bindings, binding{accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			accessv1.Policy_Spec_Rule_Surface_INTERACTIVE})
		if isClosed {
			bindings = append(bindings, binding{accessv1.IntegrationBinding_Status_NOTIFICATION,
				accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
				accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET})
		}
	case "jira-access":
		stepName = "security"
		bindings = append(bindings, binding{accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			accessv1.Policy_Spec_Rule_Surface_INTERACTIVE})
	default:
		bindings = append(bindings, binding{accessv1.IntegrationBinding_Status_NOTIFICATION,
			accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET})
	}

	for _, b := range bindings {
		state := accessv1.IntegrationBinding_Status_READY
		if isClosed && b.purpose == accessv1.IntegrationBinding_Status_REVIEW_SURFACE {
			state = accessv1.IntegrationBinding_Status_CLOSED
		}

		status := &accessv1.IntegrationBinding_Status{
			IntegrationRef:  umetav1.GetObjectReference(integration),
			RequestRef:      umetav1.GetObjectReference(created),
			Audience:        b.audience,
			Purpose:         b.purpose,
			InteractionMode: b.mode,
			State:           state,
			StepName:        stepName,
			DesiredRevision: "rev-2",
			AppliedRevision: "rev-2",
			LastSuccessAt:   timestamppb.New(createdAt.Add(5 * time.Second)),
		}

		switch itm.integration {
		case "slack-access":
			status.ExternalID = fmt.Sprintf("1729%d.000200", createdAt.Unix()%100000)
			status.ExternalRecipientID = "C05ACCESS42"
			status.ExternalURL = "https://acme.slack.com/archives/C05ACCESS42"
		case "jira-access":
			status.ExternalID = fmt.Sprintf("ACCESS-%d", 100+createdAt.Unix()%900)
			status.ExternalRecipientID = "ACCESS"
			status.ExternalURL = fmt.Sprintf("https://acme.atlassian.net/browse/%s", status.ExternalID)
		case "siem-webhook":
			status.ExternalRecipientID = "siem"
			if !isClosed {
				status.State = accessv1.IntegrationBinding_Status_DEGRADED
				status.Attempts = 4
				status.LastError = "The webhook returned the status code 503"
				status.NextAttemptAt = timestamppb.New(now.Add(3 * time.Minute))
				status.AppliedRevision = "rev-1"
			}
		}

		if _, err := octeliumC.AccessC().CreateIntegrationBinding(ctx, &accessv1.IntegrationBinding{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(12),
			},
			Spec:   &accessv1.IntegrationBinding_Spec{},
			Status: status,
		}); err != nil {
			return err
		}
	}

	return nil
}

func mockHours(h uint32) *metav1.Duration {
	return &metav1.Duration{
		Type: &metav1.Duration_Hours{Hours: h},
	}
}

func restrictedLevel(name, displayName, description, match string,
	maxDuration *metav1.Duration) *accessv1.Catalog_Spec_AccessLevel {
	return &accessv1.Catalog_Spec_AccessLevel{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
			Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{
				InlinePolicies: []*corev1.InlinePolicy{
					{
						Name: name,
						Spec: &corev1.Policy_Spec{
							Rules: []*corev1.Policy_Spec_Rule{
								{
									Effect: corev1.Policy_Spec_Rule_ALLOW,
									Condition: &corev1.Condition{
										Type: &corev1.Condition_Match{
											Match: match,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		MaxDuration: maxDuration,
	}
}

func getMockCatalogServices(names, namespaces []string,
	services map[string]*corev1.Service) []*corev1.Service {
	seen := map[string]struct{}{}
	var ret []*corev1.Service

	add := func(svc *corev1.Service) {
		if _, ok := seen[svc.Metadata.Uid]; ok {
			return
		}
		seen[svc.Metadata.Uid] = struct{}{}
		ret = append(ret, svc)
	}

	for _, name := range names {
		if svc, ok := services[name]; ok {
			add(svc)
		}
	}

	for _, ns := range namespaces {
		for _, svc := range services {
			if strings.HasSuffix(svc.Metadata.Name, "."+ns) {
				add(svc)
			}
		}
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Metadata.Name < ret[j].Metadata.Name
	})

	return ret
}

func mockReviewJustification(decision accessv1.Review_Spec_Decision) string {
	if decision == accessv1.Review_Spec_DECISION_REJECT {
		return "The linked incident is already resolved. Ask again if it reopens."
	}

	return "Matches the on-call schedule and the linked incident."
}

func setMockStateReason(state *accessv1.Request_Status_State, itm *mockAccessRequest,
	rule *accessv1.Policy_Spec_Rule, users map[string]*corev1.User) {
	switch state.Status {
	case accessv1.Request_Status_State_PENDING:
		state.Reason = accessv1.Request_Status_State_AWAITING_REVIEW
	case accessv1.Request_Status_State_APPROVED:
		if rule != nil && rule.Effect == accessv1.Policy_Spec_Rule_AUTO_APPROVE {
			state.Reason = accessv1.Request_Status_State_AUTO_APPROVED
			state.Message = rule.Message
		} else {
			state.Reason = accessv1.Request_Status_State_APPROVED_BY_REVIEWERS
			if len(itm.reviewers) > 0 {
				state.ActorRef = umetav1.GetObjectReference(users[itm.reviewers[len(itm.reviewers)-1]])
			}
		}
	case accessv1.Request_Status_State_REJECTED:
		switch {
		case rule == nil:
			state.Reason = accessv1.Request_Status_State_NO_MATCHING_RULE
		case rule.Effect == accessv1.Policy_Spec_Rule_DENY:
			state.Reason = accessv1.Request_Status_State_DENIED_BY_RULE
			state.Message = rule.Message
		default:
			state.Reason = accessv1.Request_Status_State_REJECTED_BY_REVIEWER
			state.Message = mockReviewJustification(accessv1.Review_Spec_DECISION_REJECT)
			if len(itm.reviewers) > 0 {
				state.ActorRef = umetav1.GetObjectReference(users[itm.reviewers[len(itm.reviewers)-1]])
			}
		}
	case accessv1.Request_Status_State_EXPIRED:
		state.Reason = accessv1.Request_Status_State_ACCESS_ENDED
	case accessv1.Request_Status_State_REVOKED:
		state.Reason = accessv1.Request_Status_State_ACCESS_REVOKED
		state.Message = "The incident was resolved."
		state.ActorRef = umetav1.GetObjectReference(users["dave"])
	case accessv1.Request_Status_State_CANCELLED:
		state.Reason = accessv1.Request_Status_State_CANCELLED_BY_USER
		state.ActorRef = umetav1.GetObjectReference(users[itm.requester])
	}
}

func setMockStepStates(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, createdAt, approvedAt time.Time) error {
	steps := req.Status.Rule.GetAction().GetReview().GetSteps()
	isPending := req.Status.State.Status == accessv1.Request_Status_State_PENDING

	last := int32(len(steps) - 1)
	if isPending {
		last = req.Status.Review.CurrentStep
	}

	for idx := int32(0); idx <= last && int(idx) < len(steps); idx++ {
		step := steps[idx]
		startedAt := createdAt.Add(time.Duration(idx) * 12 * time.Minute)

		stepState := &accessv1.Request_Status_Review_StepState{
			Name:      accesscmd.StepName(step, idx),
			Index:     idx,
			StartedAt: timestamppb.New(startedAt),
		}

		usrs, err := accesscmd.GetStepEligibleUsers(ctx, octeliumC, req, step, stepState, nil)
		if err != nil {
			return err
		}

		stepState.TotalEligibleReviewer = uint32(len(usrs))
		stepState.IsUnroutable = len(usrs) == 0
		stepState.RequiredApprovals = accesscmd.GetRequiredApprovals(step, stepState.TotalEligibleReviewer)
		for _, usr := range usrs {
			stepState.EligibleReviewers = append(stepState.EligibleReviewers, umetav1.GetObjectReference(usr))
		}

		switch {
		case isPending && idx == last:
		case req.Status.State.Status == accessv1.Request_Status_State_REJECTED && idx == last:
			stepState.Outcome = accessv1.Request_Status_Review_StepState_REJECTED
			stepState.EndedAt = req.Status.ApprovalEndAt
		case !approvedAt.IsZero() || isPending:
			stepState.Outcome = accessv1.Request_Status_Review_StepState_SATISFIED
			stepState.Approvals = stepState.RequiredApprovals
			stepState.EndedAt = timestamppb.New(startedAt.Add(10 * time.Minute))
		default:
			stepState.Outcome = accessv1.Request_Status_Review_StepState_INTERRUPTED
			stepState.EndedAt = req.Status.ApprovalEndAt
		}

		req.Status.Review.Steps = append(req.Status.Review.Steps, stepState)
	}

	return nil
}
