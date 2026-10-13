package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureAsyncInsertDSN(t *testing.T) {
	// DSN without query params
	dsn1 := "clickhouse+native://default:@localhost:9000/opsway"
	res1 := ensureAsyncInsertDSN(dsn1)
	assert.Equal(t, "clickhouse+native://default:@localhost:9000/opsway?async_insert=1&wait_for_async_insert=0", res1)

	// DSN with existing query params but no async_insert
	dsn2 := "clickhouse+native://default:@localhost:9000/opsway?dial_timeout=10s"
	res2 := ensureAsyncInsertDSN(dsn2)
	assert.Equal(t, "clickhouse+native://default:@localhost:9000/opsway?dial_timeout=10s&async_insert=1&wait_for_async_insert=0", res2)

	// DSN with existing async_insert
	dsn3 := "clickhouse+native://default:@localhost:9000/opsway?async_insert=1&wait_for_async_insert=1"
	res3 := ensureAsyncInsertDSN(dsn3)
	assert.Equal(t, dsn3, res3)
}
