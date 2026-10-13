package alerting

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	Repository
	rules    []entities.AlertRule
	triggers []entities.AlertTrigger
}

func (f *fakeRepo) GetEnabledRulesByTeamID(_ context.Context, _ uint) (*[]entities.AlertRule, error) {
	return &f.rules, nil
}

func (f *fakeRepo) CreateTrigger(_ context.Context, t *entities.AlertTrigger) error {
	f.triggers = append(f.triggers, *t)

	return nil
}

type fakeDispatcher struct {
	mu    sync.Mutex
	calls map[uint][]string
	wg    sync.WaitGroup
}

func (d *fakeDispatcher) Dispatch(_ context.Context, rule *entities.AlertRule, _ *entities.Incident, channels []string) {
	defer d.wg.Done()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls[rule.ID] = channels
}

func TestOnIncidentCreated(t *testing.T) {
	repo := &fakeRepo{rules: []entities.AlertRule{
		{ID: 1, Condition: "*", Channels: `["email","slack"]`},
		{ID: 2, Condition: "STATUS_CODE", Channels: `["webhook"]`},
		{ID: 3, Condition: "TLS", Channels: `["discord"]`},
		{ID: 4, Condition: "*", Channels: `[]`},
	}}
	d := &fakeDispatcher{calls: map[uint][]string{}}
	d.wg.Add(2)

	s := NewService(logrus.NewEntry(logrus.New()), repo, d)
	s.OnIncidentCreated(context.Background(), &entities.Incident{ID: 7, TeamID: 1, Title: "STATUS_CODE"})

	done := make(chan struct{})
	go func() { d.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch not called")
	}

	require.Len(t, repo.triggers, 3, "rules 1, 2 and 4 match")
	for _, tr := range repo.triggers {
		assert.Equal(t, uint(7), *tr.IncidentID)
	}
	assert.Equal(t, []string{"email", "slack"}, d.calls[1])
	assert.Equal(t, []string{"webhook"}, d.calls[2])
	assert.NotContains(t, d.calls, uint(3))
	assert.NotContains(t, d.calls, uint(4), "rules without channels are recorded but not dispatched")
}
