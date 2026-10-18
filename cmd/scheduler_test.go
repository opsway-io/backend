package cmd

import (
	"context"
	"testing"

	proberPkg "github.com/opsway-io/backend/internal/prober"
	"github.com/stretchr/testify/assert"
)

type stubProberService struct {
	locations []proberPkg.LocationInfo
	err       error
}

func (s *stubProberService) RegisterHeartbeat(ctx context.Context, info proberPkg.LocationInfo) error {
	return nil
}

func (s *stubProberService) Unregister(ctx context.Context, locationID string) error {
	return nil
}

func (s *stubProberService) GetActiveLocations(ctx context.Context) ([]proberPkg.LocationInfo, error) {
	return s.locations, s.err
}

func TestResolveTargetLocation(t *testing.T) {
	ctx := context.Background()

	t.Run("routes legacy global task to active prober opsway-dk", func(t *testing.T) {
		svc := &stubProberService{
			locations: []proberPkg.LocationInfo{{ID: "opsway-dk", Name: "Denmark (opsway-dk)", Status: "online"}},
		}
		assert.Equal(t, "opsway-dk", resolveTargetLocation(ctx, svc, "global", "opsway-dk"))
	})

	t.Run("preserves global if global prober is actively registered", func(t *testing.T) {
		svc := &stubProberService{
			locations: []proberPkg.LocationInfo{{ID: "global", Name: "Global", Status: "online"}},
		}
		assert.Equal(t, "global", resolveTargetLocation(ctx, svc, "global", "opsway-dk"))
	})

	t.Run("routes default opsway-dk to global when only global prober is active", func(t *testing.T) {
		svc := &stubProberService{
			locations: []proberPkg.LocationInfo{{ID: "global", Name: "Global", Status: "online"}},
		}
		assert.Equal(t, "global", resolveTargetLocation(ctx, svc, "opsway-dk", "global"))
	})

	t.Run("preserves explicit non-legacy location", func(t *testing.T) {
		svc := &stubProberService{
			locations: []proberPkg.LocationInfo{{ID: "opsway-dk", Name: "Denmark", Status: "online"}},
		}
		assert.Equal(t, "opsway-de", resolveTargetLocation(ctx, svc, "opsway-de", "opsway-dk"))
	})

	t.Run("falls back to configured location when no active heartbeats exist for global", func(t *testing.T) {
		svc := &stubProberService{}
		assert.Equal(t, "opsway-dk", resolveTargetLocation(ctx, svc, "global", "opsway-dk"))
	})
}
