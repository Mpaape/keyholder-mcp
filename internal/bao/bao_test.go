package bao

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/openbao/openbao/api/v2"
)

func TestCleanErr(t *testing.T) {
	denied := fmt.Errorf("wrap: %w", &api.ResponseError{StatusCode: 403, URL: "http://x/v1/secret/data/p", Errors: []string{"permission denied"}})
	err := cleanErr("ler p", denied)
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if strings.Contains(err.Error(), "http://") {
		t.Fatalf("mensagem vazou a URL: %v", err)
	}
	other := cleanErr("ler p", &api.ResponseError{StatusCode: 500})
	if other.Error() != "ler p: cofre respondeu HTTP 500" {
		t.Fatalf("err = %v", other)
	}
}
