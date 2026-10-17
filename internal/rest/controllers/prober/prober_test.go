package prober

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/prober"
	"github.com/stretchr/testify/assert"
)

type mockProberService struct {
	locations []prober.LocationInfo
}

func (m *mockProberService) RegisterHeartbeat(ctx context.Context, info prober.LocationInfo) error {
	return nil
}

func (m *mockProberService) Unregister(ctx context.Context, locationID string) error {
	return nil
}

func (m *mockProberService) GetActiveLocations(ctx context.Context) ([]prober.LocationInfo, error) {
	return m.locations, nil
}

func TestGetLocations(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/prober/locations", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockSvc := &mockProberService{
		locations: []prober.LocationInfo{
			{
				ID:          "opsway-dk",
				Name:        "Denmark (opsway-dk)",
				CountryCode: "dk",
				Status:      "online",
			},
		},
	}

	h := &Handlers{
		proberService: mockSvc,
	}

	err := h.GetLocations(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var res GetLocationsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, []string{"opsway-dk"}, res.Locations)
	assert.Len(t, res.Details, 1)
	assert.Equal(t, "opsway-dk", res.Details[0].ID)
	assert.Equal(t, "Denmark (opsway-dk)", res.Details[0].Name)
	assert.Equal(t, "dk", res.Details[0].CountryCode)
	assert.Equal(t, "online", res.Details[0].Status)
}
