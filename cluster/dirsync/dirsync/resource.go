// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dirsync

type resourceCommon struct {
	ID         string       `json:"id,omitempty"`
	ExternalID string       `json:"externalId,omitempty"`
	Meta       resourceMeta `json:"meta,omitempty"`
	Schemas    []string     `json:"schemas,omitempty"`
}

type resourceMeta struct {
	ResourceType string `json:"resourceType,omitempty"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Version      string `json:"version,omitempty"`
	Location     string `json:"location,omitempty"`
}

type resourceUser struct {
	resourceCommon
	Name        resourceUserName     `json:"name,omitempty"`
	UserName    string               `json:"userName,omitempty"`
	Emails      []*resourceUserEmail `json:"emails,omitempty"`
	Password    string               `json:"password,omitempty"`
	Active      bool                 `json:"active,omitempty"`
	DisplayName string               `json:"displayName,omitempty"`
	Locale      string               `json:"locale,omitempty"`
	ProfileURL  string               `json:"profileUrl,omitempty"`
	Photos      []*resourceUserPhoto `json:"photos,omitempty"`
}

type resourceUserName struct {
	Formatted  string `json:"formatted,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type resourceUserEmail struct {
	Primary bool   `json:"primary,omitempty"`
	Value   string `json:"value,omitempty"`
	Type    string `json:"type,omitempty"`
	Display string `json:"display,omitempty"`
}

type resourceUserPhoto struct {
	Primary bool   `json:"primary,omitempty"`
	Value   string `json:"value,omitempty"`
	Type    string `json:"type,omitempty"`
}

type resourceGroup struct {
	resourceCommon
	DisplayName string                `json:"displayName,omitempty"`
	Members     []resourceGroupMember `json:"members,omitempty"`
}

type resourceGroupMember struct {
	Value   string `json:"value,omitempty"`
	Display string `json:"display,omitempty"`
}

type serviceProviderConfig struct {
	Schemas               []string                                    `json:"schemas,omitempty"`
	DocumentationURI      string                                      `json:"documentationUri,omitempty"`
	Patch                 serviceProviderConfigPatch                  `json:"patch,omitempty"`
	Bulk                  serviceProviderConfigBulk                   `json:"bulk,omitempty"`
	Filter                serviceProviderConfigFilter                 `json:"filter,omitempty"`
	ChangePassword        serviceProviderConfigChangePassword         `json:"changePassword,omitempty"`
	Sort                  serviceProviderConfigSort                   `json:"sort,omitempty"`
	Etag                  serviceProviderConfigEtag                   `json:"etag,omitempty"`
	AuthenticationSchemes []serviceProviderConfigAuthenticationScheme `json:"authenticationSchemes,omitempty"`
}

type serviceProviderConfigPatch struct {
	Supported bool `json:"supported,omitempty"`
}

type serviceProviderConfigBulk struct {
	Supported      bool `json:"supported,omitempty"`
	MaxOperations  int  `json:"maxOperations,omitempty"`
	MaxPayloadSize int  `json:"maxPayloadSize,omitempty"`
}

type serviceProviderConfigFilter struct {
	Supported  bool `json:"supported,omitempty"`
	MaxResults int  `json:"maxResults,omitempty"`
}

type serviceProviderConfigChangePassword struct {
	Supported bool `json:"supported,omitempty"`
}

type serviceProviderConfigSort struct {
	Supported bool `json:"supported,omitempty"`
}

type serviceProviderConfigEtag struct {
	Supported bool `json:"supported,omitempty"`
}

type serviceProviderConfigAuthenticationScheme struct {
	Description      string `json:"description,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Name             string `json:"name,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
	SpecURI          string `json:"specUri,omitempty"`
	Type             string `json:"type,omitempty"`
}

type schemasResponse struct {
	Schemas      []string                  `json:"schemas,omitempty"`
	ItemsPerPage int                       `json:"itemsPerPage,omitempty"`
	StartIndex   int                       `json:"startIndex,omitempty"`
	TotalResults int                       `json:"totalResults,omitempty"`
	Resources    []schemasResponseResource `json:"Resources,omitempty"`
}

type schemasResponseResource struct {
	Schemas     []string                             `json:"schemas,omitempty"`
	ID          string                               `json:"id,omitempty"`
	Name        string                               `json:"name,omitempty"`
	Description string                               `json:"description,omitempty"`
	Attributes  []schemasResponseResourceAttribute   `json:"attributes,omitempty"`
	Meta        schemasResponseResourceAttributeMeta `json:"meta,omitempty"`
}

type schemasResponseResourceAttribute struct {
	Name          string                               `json:"name,omitempty"`
	Type          string                               `json:"type,omitempty"`
	MultiValued   bool                                 `json:"multiValued,omitempty"`
	Description   string                               `json:"description,omitempty"`
	Required      bool                                 `json:"required,omitempty"`
	CaseExact     bool                                 `json:"caseExact,omitempty"`
	Mutability    string                               `json:"mutability,omitempty"`
	Returned      string                               `json:"returned,omitempty"`
	Uniqueness    string                               `json:"uniqueness,omitempty"`
	Meta          schemasResponseResourceAttributeMeta `json:"meta,omitempty"`
	SubAttributes []schemasResponseResourceAttribute   `json:"subAttributes,omitempty"`
}

type schemasResponseResourceAttributeMeta struct {
	ResourceType string `json:"resourceType,omitempty"`
	Location     string `json:"location,omitempty"`
}

type responseError struct {
	Schemas []string `json:"schemas,omitempty"`
	Status  string   `json:"status,omitempty"`
	Detail  string   `json:"detail,omitempty"`
}

type responseUserList struct {
	Schemas      []string       `json:"schemas,omitempty"`
	ItemsPerPage int            `json:"itemsPerPage,omitempty"`
	StartIndex   int            `json:"startIndex,omitempty"`
	TotalResults int            `json:"totalResults,omitempty"`
	Resources    []resourceUser `json:"Resources,omitempty"`
}

type responseGroupList struct {
	Schemas      []string        `json:"schemas,omitempty"`
	ItemsPerPage int             `json:"itemsPerPage,omitempty"`
	StartIndex   int             `json:"startIndex,omitempty"`
	TotalResults int             `json:"totalResults,omitempty"`
	Resources    []resourceGroup `json:"Resources,omitempty"`
}

type patchRequest struct {
	Schemas    []string                 `json:"schemas,omitempty"`
	Operations []patchRequestOperations `json:"Operations,omitempty"`
}

type patchRequestOperations struct {
	Op    string `json:"op,omitempty"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

type responseResourceTypes struct {
	Schemas      []string                        `json:"schemas,omitempty"`
	ItemsPerPage int                             `json:"itemsPerPage,omitempty"`
	StartIndex   int                             `json:"startIndex,omitempty"`
	TotalResults int                             `json:"totalResults,omitempty"`
	Resources    []responseResourceTypesResource `json:"Resources,omitempty"`
}

type responseResourceTypesResource struct {
	resourceCommon
	Name        string `json:"name,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema,omitempty"`
}
