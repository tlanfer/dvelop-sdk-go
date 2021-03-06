package tenant_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/d-velop/dvelop-sdk-go/tenant"
)

const (
	systemBaseUriHeader  = "x-dv-baseuri"
	tenantIdHeader       = "x-dv-tenant-id"
	signatureHeader      = "x-dv-sig-1"
	defaultSystemBaseUri = "https://default.example.com"
	forwardedHeader      = "forwarded"
	xForwardedHostHeader = "x-forwarded-host"
	uriPrefix            = "https://"
)

func TestBaseUriHeaderAndEmptyDefaultBaseUri_UsesHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	req.Header.Set(signatureHeader, base64Signature(systemBaseUriFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}

	logSpy := loggerSpy{}
	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(systemBaseUriFromHeader); err != nil {
		t.Error(err)
	}
}

func TestNoBaseUriHeaderAndDefaultBaseUri_UsesDefaultBaseUri(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestBaseUriHeaderAndDefaultBaseUri_UsesHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://header.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	req.Header.Set(signatureHeader, base64Signature(systemBaseUriFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(systemBaseUriFromHeader); err != nil {
		t.Error(err)
	}
}

func TestNoBaseUriHeaderAndEmptyDefaultBaseUri_DoesntAddBaseUriToContext(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	handlerSpy := handlerSpy{}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(httptest.NewRecorder(), req)

	if err := handlerSpy.assertErrorReadingSystemBaseUri(); err != nil {
		t.Error(err)
	}
}

func TestTenantIdHeader_UsesHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	req.Header.Set(signatureHeader, base64Signature(tenantIdFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs(tenantIdFromHeader); err != nil {
		t.Error(err)
	}
}

func TestNoTenantIdHeader_UsesTenantIdZero(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs("0"); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_UsesForwardedHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const forwardedHostValue = "forwarded.example.com"
	const forwardedHeaderValue = "host=" + forwardedHostValue
	req.Header.Set(forwardedHeader, forwardedHeaderValue)
	req.Header.Set(signatureHeader, base64Signature(forwardedHeaderValue, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(uriPrefix + forwardedHostValue); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_UsesForwardedHeaderMultipleHosts(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const forwardedHostValue = "forwarded.example.com"
	const forwardedHeaderValue = "host=" + forwardedHostValue + ",secondhost.example.com"
	req.Header.Set(forwardedHeader, forwardedHeaderValue)
	req.Header.Set(signatureHeader, base64Signature(forwardedHeaderValue, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(uriPrefix + forwardedHostValue); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_UsesXForwardedHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const xForwardedHostValue = "xforwarded.example.com"
	req.Header.Set(xForwardedHostHeader, xForwardedHostValue)
	req.Header.Set(signatureHeader, base64Signature(xForwardedHostValue, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(uriPrefix + xForwardedHostValue); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_UsesXForwardedHeaderMultipleHosts(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const xForwardedHostValue = "xforwarded.example.com"
	const xForwardedHostMultiValue = xForwardedHostValue + ",secondhost.example.com"
	req.Header.Set(xForwardedHostHeader, xForwardedHostMultiValue)
	req.Header.Set(signatureHeader, base64Signature(xForwardedHostMultiValue, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(uriPrefix + xForwardedHostValue); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_EmptyForwardedHeadersNoSystemBaseUri(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set(signatureHeader, base64Signature("", signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(""); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_EmptyForwardedHeadersWithSystemBaseUri(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUri = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUri)
	req.Header.Set(signatureHeader, base64Signature(systemBaseUri, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(systemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestInitiatorSystemBaseUriHeader_EmptyForwardedHeadersWithDefaultSystemBaseUri(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const defaultSystemBaseUri = "https://sample.example.com"

	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestTenantIdHeaderAndBaseUriHeader_UsesHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	const systemBaseUriFromHeader = "https://header.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	req.Header.Set(signatureHeader, base64Signature(systemBaseUriFromHeader+tenantIdFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs(tenantIdFromHeader); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(systemBaseUriFromHeader); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(systemBaseUriFromHeader); err != nil {
		t.Error(err)
	}
}

func TestTenantIdHeaderAndNoBaseUriHeader_UsesTenantIdHeaderAndDefaultSystemBaseUri(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	req.Header.Set(signatureHeader, base64Signature(tenantIdFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs(tenantIdFromHeader); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestNoHeadersButDefaultSystemBaseUri_UsesDefaultBaseUriAndTenantIdZero(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs("0"); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestNoHeadersButDefaultSystemBaseUriAndNoSignatureSecretKey_UsesDefaultBaseUriAndTenantIdZero(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx(defaultSystemBaseUri, nil, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusOK); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertTenantIdIs("0"); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
	if err := handlerSpy.assertInitiatorSystemBaseUriIs(defaultSystemBaseUri); err != nil {
		t.Error(err)
	}
}

func TestWrongDataSignedWithValidSignatureKey_Returns403(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	req.Header.Set(signatureHeader, base64Signature("wrong data", signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusForbidden); err != nil {
		t.Error(err)
	}
	if handlerSpy.hasBeenCalled {
		t.Error("inner handler should not have been called")
	}

	if err := logSpy.assertLogContains("signature"); err != nil {
		t.Error(err)
	}
}

func TestNoneBase64Signature_Returns403(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	req.Header.Set(signatureHeader, "abc+(9-!")
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusForbidden); err != nil {
		t.Error(err)
	}
	if handlerSpy.hasBeenCalled {
		t.Error("inner handler should not have been called")
	}

	if err := logSpy.assertLogContains("illegal base64"); err != nil {
		t.Error(err)
	}
}

func TestWrongSignatureKey_Returns403(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	wrongSignatureKey := []byte{167, 219, 144, 209, 189, 1, 178, 73, 139, 47, 21, 236, 142, 56, 71, 245, 43, 188, 163, 52, 239, 102, 94, 153, 255, 159, 199, 149, 163, 145, 161, 24}
	req.Header.Set(signatureHeader, base64Signature(systemBaseUriFromHeader+tenantIdFromHeader, wrongSignatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusForbidden); err != nil {
		t.Error(err)
	}
	if handlerSpy.hasBeenCalled {
		t.Error("inner handler should not have been called")
	}

	if err := logSpy.assertLogContains("signature"); err != nil {
		t.Error(err)
	}
}

func TestHeadersWithoutSignature_Returns403(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", signatureKey, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusForbidden); err != nil {
		t.Error(err)
	}
	if handlerSpy.hasBeenCalled {
		t.Error("inner handler should not have been called")
	}

	if err := logSpy.assertLogContains("signature"); err != nil {
		t.Error(err)
	}
}

func TestHeadersAndNoSignatureSecretKey_Returns500(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	const systemBaseUriFromHeader = "https://sample.example.com"
	req.Header.Set(systemBaseUriHeader, systemBaseUriFromHeader)
	const tenantIdFromHeader = "a12be5"
	req.Header.Set(tenantIdHeader, tenantIdFromHeader)
	req.Header.Set(signatureHeader, base64Signature(systemBaseUriFromHeader+tenantIdFromHeader, signatureKey))
	handlerSpy := handlerSpy{}
	responseSpy := responseSpy{httptest.NewRecorder()}
	logSpy := loggerSpy{}

	tenant.AddToCtx("", nil, logSpy.logError)(&handlerSpy).ServeHTTP(responseSpy, req)

	if err := responseSpy.assertStatusCodeIs(http.StatusInternalServerError); err != nil {
		t.Error(err)
	}
	if handlerSpy.hasBeenCalled {
		t.Error("inner handler should not have been called")
	}

	if err := logSpy.assertLogContains("secret"); err != nil {
		t.Error(err)
	}
}

func TestNoIdOnContext_SetId_ReturnsContextWithId(t *testing.T) {
	ctx := tenant.SetId(context.Background(), "123ABC")
	if id, _ := tenant.IdFromCtx(ctx); id != "123ABC" {
		t.Errorf("got wrong tenantId from context: got %v want %v", id, "123ABC")
	}
}

func TestIdOnContext_SetId_ReturnsContextWithNewId(t *testing.T) {
	ctx := tenant.SetId(context.Background(), "123ABC")
	ctx = tenant.SetId(ctx, "XYZ")
	if id, _ := tenant.IdFromCtx(ctx); id != "XYZ" {
		t.Errorf("got wrong tenantId from context: got %v want %v", id, "XYZ")
	}
}

func TestSystemBaseUriOnContext_SetSystemBaseUri_ReturnsContextWithSystemBaseUri(t *testing.T) {
	ctx := tenant.SetSystemBaseUri(context.Background(), "https://xyz.example.com")
	if u, _ := tenant.SystemBaseUriFromCtx(ctx); u != "https://xyz.example.com" {
		t.Errorf("got wrong systemBaseUri from context: got %v want %v", u, "https://xyz.example.com")
	}
}

func TestSystemBaseUriOnContext_SetSystemBaseUri_ReturnsContextWithNewSystemBaseUri(t *testing.T) {
	ctx := tenant.SetSystemBaseUri(context.Background(), "https://xyz.example.com")
	ctx = tenant.SetSystemBaseUri(context.Background(), "https://abc.example.com")
	if u, _ := tenant.SystemBaseUriFromCtx(ctx); u != "https://abc.example.com" {
		t.Errorf("got wrong systemBaseUri from context: got %v want %v", u, "https://abc.example.com")
	}
}

func TestInitiatorSystemBaseUriOnContext_SetInitiatorSystemBaseUri_ReturnsContextWithInitiatorSystemBaseUri(t *testing.T) {
	ctx := tenant.SetInitiatorSystemBaseUri(context.Background(), "https://initial.example.com")
	if u, _ := tenant.InitiatorSystemBaseUriFromCtx(ctx); u != "https://initial.example.com" {
		t.Errorf("got wrong initiatorSystemBaseUri from context: got %v want %v", u, "https://initial.example.com")
	}
}

func TestInitiatorSystemBaseUriOnContext_SetInitiatorSystemBaseUri_ReturnsContextWithNewInitiatorSystemBaseUri(t *testing.T) {
	ctx := tenant.SetInitiatorSystemBaseUri(context.Background(), "https://initial.example.com")
	ctx = tenant.SetInitiatorSystemBaseUri(context.Background(), "https://new.example.com")
	if u, _ := tenant.InitiatorSystemBaseUriFromCtx(ctx); u != "https://new.example.com" {
		t.Errorf("got wrong initiatorSystemBaseUri from context: got %v want %v", u, "https://new.example.com")
	}
}

var signatureKey = []byte{166, 219, 144, 209, 189, 1, 178, 73, 139, 47, 21, 236, 142, 56, 71, 245, 43, 188, 163, 52, 239, 102, 94, 153, 255, 159, 199, 149, 163, 145, 161, 24}

func base64Signature(message string, sigKey []byte) string {
	mac := hmac.New(sha256.New, sigKey)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type handlerSpy struct {
	systemBaseUri                      string
	tenantId                           string
	initiatorSystemBaseUri             string
	errorReadingSystemBaseUri          error
	errorReadingTenantId               error
	errorReadingInitiatorSystemBaseUri error
	hasBeenCalled                      bool
}

func (spy *handlerSpy) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	spy.hasBeenCalled = true
	spy.systemBaseUri, spy.errorReadingSystemBaseUri = tenant.SystemBaseUriFromCtx(r.Context())
	spy.tenantId, spy.errorReadingTenantId = tenant.IdFromCtx(r.Context())
	spy.initiatorSystemBaseUri, spy.errorReadingInitiatorSystemBaseUri = tenant.InitiatorSystemBaseUriFromCtx(r.Context())
}

func (spy *handlerSpy) assertBaseUriIs(expected string) error {
	if spy.systemBaseUri != expected {
		return fmt.Errorf("handler set wrong systemBaseUri on context: got %v want %v", spy.systemBaseUri, expected)
	}
	return nil
}

func (spy *handlerSpy) assertTenantIdIs(expected string) error {
	if spy.tenantId != expected {
		return fmt.Errorf("handler set wrong tenantId on context: got %v want %v", spy.tenantId, expected)
	}
	return nil
}

func (spy *handlerSpy) assertInitiatorSystemBaseUriIs(expected string) error {
	if spy.initiatorSystemBaseUri != expected {
		return fmt.Errorf("handler set wrong initiatorSystemBaseUri on context: got %v want %v", spy.initiatorSystemBaseUri, expected)
	}
	return nil
}

func (spy *handlerSpy) assertErrorReadingSystemBaseUri() error {
	if spy.errorReadingSystemBaseUri == nil {
		return fmt.Errorf("expected error while reading systemBaseUri from context")
	}
	return nil
}

func (spy *handlerSpy) assertErrorReadingTenantId() error {
	if spy.errorReadingTenantId == nil {
		return fmt.Errorf("expected error while reading tenantId from context")
	}
	return nil
}

func (spy *handlerSpy) assertErrorReadingInitiatorSystemBaseUri() error {
	if spy.errorReadingTenantId == nil {
		return fmt.Errorf("expected error while reading initiatorSystembaseUri from context")
	}
	return nil
}

type responseSpy struct {
	*httptest.ResponseRecorder
}

func (spy *responseSpy) assertStatusCodeIs(expectedStatusCode int) error {
	if status := spy.Code; status != expectedStatusCode {
		return fmt.Errorf("handler returned wrong status code: got %v want %v", status, expectedStatusCode)
	}
	return nil
}

type loggerSpy struct {
	hasBeenCalled bool
	lastMessage   string
}

func (spy *loggerSpy) logError(ctx context.Context, message string) {
	spy.hasBeenCalled = true
	spy.lastMessage = message
	fmt.Println(message)
}

func (spy *loggerSpy) assertLogContains(term string) error {
	if !spy.hasBeenCalled {
		return fmt.Errorf("log should have been written")
	}
	if !strings.Contains(spy.lastMessage, term) {
		return fmt.Errorf("expected log to contain the term '%v'", term)
	}
	return nil
}