package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListActiveEnergy_RequiresAuth(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/active-energy", "", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
