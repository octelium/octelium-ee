// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-acme/lego/v5/acme"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestACMEServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Replay-Nonce", utilrand.GetRandomStringCanonical(32))

		switch r.URL.Path {
		case "/directory":
			baseURL := "https://" + r.Host
			assert.Nil(t, json.NewEncoder(w).Encode(acme.Directory{
				NewNonceURL:   baseURL + "/nonce",
				NewAccountURL: baseURL + "/new-account",
				NewOrderURL:   baseURL + "/new-order",
			}))
		case "/nonce":
			w.WriteHeader(http.StatusOK)
		default:
			handler(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	previous := http.DefaultTransport
	transport := previous.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	pool, err := x509.SystemCertPool()
	require.NoError(t, err)
	pool.AddCert(srv.Certificate())
	transport.TLSClientConfig.RootCAs = pool
	http.DefaultTransport = transport
	t.Cleanup(func() {
		http.DefaultTransport = previous
		transport.CloseIdleConnections()
	})

	return srv
}

func readTestACMEPayload(t *testing.T, r *http.Request) []byte {
	t.Helper()

	var msg struct {
		Payload   string `json:"payload"`
		Protected string `json:"protected"`
	}
	if !assert.Nil(t, json.NewDecoder(r.Body).Decode(&msg)) {
		return nil
	}

	protected, err := base64.RawURLEncoding.DecodeString(msg.Protected)
	assert.Nil(t, err)
	var header struct {
		KeyID string          `json:"kid"`
		Key   json.RawMessage `json:"jwk"`
		URL   string          `json:"url"`
	}
	assert.Nil(t, json.Unmarshal(protected, &header))
	assert.Equal(t, "https://"+r.Host+r.URL.Path, header.URL)
	if r.URL.Path == "/new-account" {
		assert.Empty(t, header.KeyID)
		assert.NotEmpty(t, header.Key)
	} else {
		assert.NotEmpty(t, header.KeyID)
		assert.Empty(t, header.Key)
	}

	payload, err := base64.RawURLEncoding.DecodeString(msg.Payload)
	assert.Nil(t, err)
	return payload
}

func TestACMEAccountRegistration(t *testing.T) {
	val := `{"email":"contact@example.com","registration":{"body":{"status":"valid","contact":["mailto:contact@example.com"],"termsOfServiceAgreed":true,"orders":"https://acme.example.com/orders/1"},"uri":"https://acme.example.com/account/1"}}`

	var account Account
	assert.Nil(t, account.GetRegistration())
	require.NoError(t, json.Unmarshal([]byte(val), &account))

	reg := account.GetRegistration()
	require.NotNil(t, reg)
	assert.Equal(t, "https://acme.example.com/account/1", reg.Location)
	assert.Equal(t, acme.StatusValid, reg.Status)
	assert.Equal(t, []string{"mailto:contact@example.com"}, reg.Contact)
	assert.Equal(t, "https://acme.example.com/orders/1", reg.Orders)
	assert.True(t, reg.TermsOfServiceAgreed)

	account.setRegistration(reg)
	encoded, err := json.Marshal(&account)
	require.NoError(t, err)
	assert.JSONEq(t, val, string(encoded))
}

func TestRegisterACMEAccount(t *testing.T) {
	for _, itm := range []struct {
		name         string
		saved        bool
		missing      bool
		exists       bool
		unauthorized bool
		wantCalls    []string
	}{
		{name: "Saved", saved: true, exists: true, wantCalls: []string{"query", "update"}},
		{name: "Recovered", exists: true, wantCalls: []string{"lookup", "update"}},
		{name: "New", wantCalls: []string{"lookup", "register"}},
		{name: "MissingSaved", saved: true, missing: true, exists: true, wantCalls: []string{"missing", "lookup", "update"}},
		{name: "Removed", saved: true, missing: true, wantCalls: []string{"missing", "lookup", "register"}},
		{name: "QueryError", saved: true, unauthorized: true, wantCalls: []string{"query"}},
		{name: "LookupError", unauthorized: true, wantCalls: []string{"lookup"}},
	} {
		t.Run(itm.name, func(t *testing.T) {
			calls := make(chan string, 8)
			srv := newTestACMEServer(t, func(w http.ResponseWriter, r *http.Request) {
				payload := readTestACMEPayload(t, r)
				var account acme.Account
				if len(payload) > 0 {
					assert.Nil(t, json.Unmarshal(payload, &account))
				}

				switch r.URL.Path {
				case "/missing":
					calls <- "missing"
				case "/account/1":
					if len(payload) == 0 {
						calls <- "query"
					} else {
						calls <- "update"
					}
				case "/new-account":
					if account.OnlyReturnExisting {
						calls <- "lookup"
					} else {
						calls <- "register"
					}
				default:
					t.Errorf("Unexpected ACME request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				if itm.unauthorized {
					w.WriteHeader(http.StatusForbidden)
					assert.Nil(t, json.NewEncoder(w).Encode(acme.ProblemDetails{
						Type: acme.UnauthorizedErrorType,
					}))
					return
				}

				if r.URL.Path == "/missing" || (account.OnlyReturnExisting && !itm.exists) {
					w.WriteHeader(http.StatusBadRequest)
					assert.Nil(t, json.NewEncoder(w).Encode(acme.ProblemDetails{
						Type: acme.AccountDoesNotExistErrorType,
					}))
					return
				}

				if !account.OnlyReturnExisting && len(payload) > 0 {
					assert.Equal(t, []string{"mailto:contact@example.com"}, account.Contact)
					assert.True(t, account.TermsOfServiceAgreed)
				}

				w.Header().Set("Location", "https://"+r.Host+"/account/1")
				assert.Nil(t, json.NewEncoder(w).Encode(acme.Account{Status: acme.StatusValid}))
			})

			key, err := utils_cert.GenerateECDSA()
			require.NoError(t, err)
			account := &Account{Email: "contact@example.com", key: key.PrivateKey}
			if itm.saved {
				uri := srv.URL + "/account/1"
				if itm.missing {
					uri = srv.URL + "/missing"
				}
				account.setRegistration(&acme.ExtendedAccount{Location: uri})
			}

			c := NewController(nil)
			reg, err := c.registerACMEAccount(t.Context(), account, srv.URL+"/directory")
			if itm.unauthorized {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, reg)
				assert.Equal(t, srv.URL+"/account/1", reg.Location)
				assert.Equal(t, acme.StatusValid, reg.Status)
				assert.Equal(t, key.PrivateKey, account.GetPrivateKey())
			}

			var gotCalls []string
			for len(calls) > 0 {
				gotCalls = append(gotCalls, <-calls)
			}
			assert.Equal(t, itm.wantCalls, gotCalls)

			cancelCtx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err = c.registerACMEAccount(cancelCtx, account, srv.URL+"/directory")
			assert.ErrorIs(t, err, context.Canceled)
			assert.Empty(t, calls)
		})
	}
}

func TestCertificateIssuer(t *testing.T) {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	c := NewController(fakeC.OcteliumC)
	t.Cleanup(c.shutdown)

	srv := newTestACMEServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://"+r.Host+"/account/1")
		assert.Nil(t, json.NewEncoder(w).Encode(acme.Account{Status: acme.StatusValid}))
	})

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)
	cc.Status.Domain = fmt.Sprintf("%s.octelium.org", utilrand.GetRandomStringCanonical(8))
	cc, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)

	newIssuer := func(acme *enterprisev1.CertificateIssuer_Spec_ACME) *enterprisev1.CertificateIssuer {
		iss, err := fakeC.OcteliumC.EnterpriseC().CreateCertificateIssuer(ctx,
			&enterprisev1.CertificateIssuer{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Spec: &enterprisev1.CertificateIssuer_Spec{
					Type: &enterprisev1.CertificateIssuer_Spec_Acme{
						Acme: acme,
					},
				},
				Status: &enterprisev1.CertificateIssuer_Status{
					State: enterprisev1.CertificateIssuer_Status_NOT_READY,
				},
			})
		assert.Nil(t, err, "%+v", err)

		return iss
	}

	getIssuer := func(uid string) *enterprisev1.CertificateIssuer {
		iss, err := fakeC.OcteliumC.EnterpriseC().GetCertificateIssuer(ctx,
			&rmetav1.GetOptions{Uid: uid})
		assert.Nil(t, err)
		return iss
	}

	t.Run("NotAnACMEIssuer", func(t *testing.T) {
		iss, err := fakeC.OcteliumC.EnterpriseC().CreateCertificateIssuer(ctx,
			&enterprisev1.CertificateIssuer{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Spec:   &enterprisev1.CertificateIssuer_Spec{},
				Status: &enterprisev1.CertificateIssuer_Status{},
			})
		assert.Nil(t, err)

		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_STATE_UNKNOWN,
			getIssuer(iss.Metadata.Uid).Status.State)
	})

	t.Run("NonExistentIssuer", func(t *testing.T) {
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, vutils.UUIDv4()))
	})

	t.Run("InvalidDirectoryURL", func(t *testing.T) {
		iss := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{
			Server: "ftp://acme.example.com/directory",
		})

		err := c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid)
		assert.NotNil(t, err)
		assert.True(t, getRetryDelay(1, err) >= 10*time.Minute)

		assert.Equal(t, enterprisev1.CertificateIssuer_Status_NOT_READY,
			getIssuer(iss.Metadata.Uid).Status.State)
	})

	iss := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{
		Server: srv.URL + "/directory",
		Email:  fmt.Sprintf("contact@%s", cc.Status.Domain),
		Solver: &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
			Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{
				Dns: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_DNS{},
			},
		},
	})

	err = c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid)
	assert.Nil(t, err, "%+v", err)

	iss = getIssuer(iss.Metadata.Uid)
	assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, iss.Status.State)
	assert.NotNil(t, iss.Status.GetAcme())
	assert.NotNil(t, iss.Status.GetAcme().SecretRef)

	sec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Uid: iss.Status.GetAcme().SecretRef.Uid,
	})
	assert.Nil(t, err)
	assert.Equal(t, uenterprisev1.ToCertificateIssuer(iss).GetACMEAccountSecretName(),
		sec.Metadata.Name)
	assert.True(t, sec.Metadata.IsSystem)

	dataMap := sec.Data.GetDataMap().GetMap()
	assert.Equal(t, getCADirURL(iss), string(dataMap[acmeSecretDirectoryURLKey]))

	privateKeyPEM := dataMap[acmeSecretPrivateKeyKey]
	_, err = utils_cert.ParsePrivateKeyPEM(privateKeyPEM)
	assert.Nil(t, err)

	{
		var account Account
		assert.Nil(t, json.Unmarshal(dataMap[acmeSecretAccountKey], &account))
		assert.NotNil(t, account.Registration)
		assert.Equal(t, fmt.Sprintf("contact@%s", cc.Status.Domain), account.Email)
	}

	t.Run("AlreadyReadyIsANoOp", func(t *testing.T) {
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, iss.Metadata.ResourceVersion, cur.Metadata.ResourceVersion)
	})

	t.Run("LoadedAccount", func(t *testing.T) {
		loaded, err := c.loadACMEAccount(ctx, iss, "contact@example.com")
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, loaded)
		assert.Equal(t, "contact@example.com", loaded.account.Email)
		assert.NotNil(t, loaded.account.GetPrivateKey())
		assert.NotNil(t, loaded.account.GetRegistration())
		assert.Equal(t, getCADirURL(iss), loaded.directoryURL)

		account, err := c.getACMEAccount(ctx, iss)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, account.GetRegistration())

		acmeC, err := c.newLegoClient(ctx, iss)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, acmeC)
	})

	t.Run("ForceKeepsTheAccountKey", func(t *testing.T) {
		assert.Nil(t, c.SetCertificateIssuer(ctx, iss, true))
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, cur.Status.State)
		assert.False(t, c.hasIssuerForce(cur.Metadata.Uid))

		curSec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: cur.Status.GetAcme().SecretRef.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, string(privateKeyPEM),
			string(curSec.Data.GetDataMap().GetMap()[acmeSecretPrivateKeyKey]))
	})

	t.Run("ACorruptAccountIsReRegistered", func(t *testing.T) {
		curSec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: iss.Status.GetAcme().SecretRef.Uid,
		})
		assert.Nil(t, err)

		curSec.Data.GetDataMap().Map[acmeSecretAccountKey] = []byte("not-an-account")
		_, err = fakeC.OcteliumC.EnterpriseC().UpdateSecret(ctx, curSec)
		assert.Nil(t, err)

		loaded, err := c.loadACMEAccount(ctx, iss, "contact@example.com")
		assert.Nil(t, err)
		assert.Nil(t, loaded)

		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, cur.Status.State)

		account, err := c.getACMEAccount(ctx, cur)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, account.GetRegistration())
	})

	t.Run("EmailFallsBackToTheClusterDomain", func(t *testing.T) {
		other := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{})

		email, err := c.getIssuerEmail(ctx, other)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("contact@%s", cc.Status.Domain), email)
	})

	t.Run("SetIssuerState", func(t *testing.T) {
		cur, err := c.setIssuerState(ctx, iss.Metadata.Uid,
			enterprisev1.CertificateIssuer_Status_NOT_READY, nil)
		assert.Nil(t, err)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_NOT_READY, cur.Status.State)
		assert.NotNil(t, cur.Status.GetAcme())

		_, err = c.setIssuerState(ctx, vutils.UUIDv4(),
			enterprisev1.CertificateIssuer_Status_NOT_READY, nil)
		assert.NotNil(t, err)
	})
}
