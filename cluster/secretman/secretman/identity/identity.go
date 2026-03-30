// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package identity

import (
	"context"
	"os"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
)

type IdentityToken struct {
	Token string
}

func GetIdentityToken(ctx context.Context, ss *enterprisev1.SecretStore) (*IdentityToken, error) {
	if ldflags.IsTest() {
		return &IdentityToken{
			Token: os.Getenv("OCTELIUM_TEST_ID_TOKEN"),
		}, nil
	}

	return getIdentityTokenK8s(ctx, ss)
}

func getIdentityTokenK8s(ctx context.Context, ss *enterprisev1.SecretStore) (*IdentityToken, error) {

	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, err
	}

	return &IdentityToken{
		Token: string(token),
	}, nil
}

/*
func getIdentityTokenGCP(ctx context.Context, ss *enterprisev1.SecretStore, domain string) (*IdentityToken, error) {

	res, err := resty.New().R().
		SetHeader("Metadata-Flavor", "Google").
		Get(fmt.Sprintf("http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s&format=full", domain))

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, errors.Errorf("Could not obtain GCP identity: status code %d", res.StatusCode())
	}

	return &IdentityToken{
		Token: res.String(),
	}, nil

}

func getIdentityTokenAzure(ctx context.Context, ss *enterprisev1.SecretStore) (*IdentityToken, error) {
	const (
		defaultMountPath     = "azure"
		defaultResourceURL   = "https://management.azure.com/"
		metadataEndpoint     = "http://169.254.169.254"
		metadataAPIVersion   = "2021-05-01"
		apiVersionQueryParam = "api-version"
		resourceQueryParam   = "resource"
		clientTimeout        = 10 * time.Second
	)

	type errorJSON struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	type responseJSON struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
		ExpiresOn    string `json:"expires_on"`
		NotBefore    string `json:"not_before"`
		Resource     string `json:"resource"`
		TokenType    string `json:"token_type"`
	}

	identityEndpoint, err := url.Parse(fmt.Sprintf("%s/metadata/identity/oauth2/token", metadataEndpoint))
	if err != nil {
		return nil, errors.Errorf("could not create Azure metadata URL: %+v", err)
	}

	identityParameters := identityEndpoint.Query()
	identityParameters.Add(apiVersionQueryParam, metadataAPIVersion)
	// identityParameters.Add(resourceQueryParam, aud)
	identityParameters.Add(resourceQueryParam, defaultResourceURL)
	identityEndpoint.RawQuery = identityParameters.Encode()

	req, err := http.NewRequest(http.MethodGet, identityEndpoint.String(), nil)
	if err != nil {
		return nil, errors.Errorf("Could not create Azure metadata url HTTP request: %+v", err)
	}
	req.Header.Add("Metadata", "true")

	client := &http.Client{
		Timeout: clientTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Errorf("Could not do HTTP request to Azure metadata url: %+v", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorJSON
		err = json.Unmarshal(responseBytes, &errResp)
		if err != nil {
			return nil, errors.Errorf("Could not unmarshal Azure metadata url error response: %+v", err)
		}
		return nil, errors.Errorf("Could not get token from Azure metadata url: %+v: %+v",
			errResp.Error, errResp.ErrorDescription)
	}

	var r responseJSON
	err = json.Unmarshal(responseBytes, &r)
	if err != nil {
		return nil, errors.Errorf("Could not unmarshal Azure metadata endpoint response: %+v", err)
	}

	return &IdentityToken{
		Token: r.AccessToken,
	}, nil
}
*/
