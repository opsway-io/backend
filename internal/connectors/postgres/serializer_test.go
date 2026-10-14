package postgres

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

type dummyModel struct {
	Frequency time.Duration `gorm:"serializer:timeDurationSeconds"`
}

func TestTimeDurationSecondsSerializer(t *testing.T) {
	s, ok := schema.GetSerializer("timeDurationSeconds")
	require.True(t, ok, "timeDurationSeconds serializer should be registered")

	sch, err := schema.Parse(&dummyModel{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)
	field := sch.FieldsByName["Frequency"]
	require.NotNil(t, field)

	t.Run("Value serialization", func(t *testing.T) {
		val, err := s.Value(context.Background(), field, reflect.Value{}, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(60), val)

		// 30 seconds
		val30, err := s.Value(context.Background(), field, reflect.Value{}, 30*time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(30), val30)
	})

	t.Run("Scan normal seconds", func(t *testing.T) {
		m := dummyModel{}
		err := s.Scan(context.Background(), field, reflect.ValueOf(&m).Elem(), int64(60))
		require.NoError(t, err)
		assert.Equal(t, time.Minute, m.Frequency)
	})

	t.Run("Scan legacy nanoseconds without overflow", func(t *testing.T) {
		m := dummyModel{}
		// 60,000,000,000 ns should be normalized to 60s (1 minute)
		err := s.Scan(context.Background(), field, reflect.ValueOf(&m).Elem(), int64(60000000000))
		require.NoError(t, err)
		assert.Equal(t, time.Minute, m.Frequency)
		assert.Equal(t, uint64(60), uint64(m.Frequency/time.Second))
	})
}
