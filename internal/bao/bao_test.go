package bao

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openbao/openbao/api/v2"
)

func TestCleanErr(t *testing.T) {
	denied := fmt.Errorf("wrap: %w", &api.ResponseError{StatusCode: 403, URL: "http://x/v1/secret/data/p", Errors: []string{"permission denied"}})
	err := cleanErr("read p", denied)
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if strings.Contains(err.Error(), "http://") {
		t.Fatalf("message leaked the URL: %v", err)
	}
	other := cleanErr("read p", &api.ResponseError{StatusCode: 500})
	if other.Error() != "read p: vault returned HTTP 500" {
		t.Fatalf("err = %v", other)
	}
}

// vault starts a fake OpenBao that answers sys/wrapping/unwrap with the given
// status and reports the given seal state.
func vault(t *testing.T, unwrapStatus int, sealed bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/sys/wrapping/unwrap":
			w.WriteHeader(unwrapStatus)
			_, _ = w.Write([]byte(`{"errors":["refused"]}`))
		case "/v1/sys/seal-status":
			_, _ = fmt.Fprintf(w, `{"type":"shamir","initialized":true,"sealed":%t,"t":2,"n":3,"progress":0}`, sealed)
		case "/v1/sys/health":
			// The client asks for sealedcode=299 and friends, so a sealed
			// vault answers with a success code here.
			_, _ = fmt.Fprintf(w, `{"initialized":true,"sealed":%t,"standby":false,"version":"2.7.1"}`, sealed)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func openWith(t *testing.T, addr string) error {
	t.Helper()
	t.Setenv("BAO_ADDR", addr)
	t.Setenv(EnvWrappingToken, "wrapped")
	t.Setenv(EnvToken, "")
	_, err := Open(context.Background(), "secret", "my-project")
	return err
}

func TestOpenSealed(t *testing.T) {
	err := openWith(t, vault(t, 503, true).URL)
	if !errors.Is(err, ErrSealed) {
		t.Fatalf("err = %v, want ErrSealed", err)
	}
	if strings.Contains(err.Error(), "already used") {
		t.Fatalf("a sealed vault must not blame the token: %v", err)
	}
}

func TestOpenRefusedToken(t *testing.T) {
	err := openWith(t, vault(t, 400, false).URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400 (already used or expired?)") {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenUntrustedCertificate(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	err := openWith(t, srv.URL)
	if err == nil || !strings.Contains(err.Error(), "set BAO_CACERT") {
		t.Fatalf("err = %v, want the BAO_CACERT hint", err)
	}
	if strings.Contains(err.Error(), "/v1/") || strings.Contains(err.Error(), "already used") {
		t.Fatalf("message leaked the request URL or blamed the token: %v", err)
	}
}

func TestProbe(t *testing.T) {
	srv := vault(t, 503, true)
	t.Setenv("BAO_ADDR", srv.URL)
	st := Probe(context.Background(), "/secret/", "my-project/")
	if !st.Sealed || !st.Initialized || st.Version != "2.7.1" || st.Addr != srv.URL {
		t.Fatalf("status = %+v", st)
	}
	if st.Mount != "secret" || st.Scope != "my-project" {
		t.Fatalf("mount/scope = %q/%q", st.Mount, st.Scope)
	}
}
