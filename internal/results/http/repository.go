package result

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Repository interface {
	Write(data map[string]interface{})
	Read(bucket string) (string, error)
}

type RepositoryImpl struct {
	writeClient api.WriteAPI
	readClient  api.QueryAPI
}

func NewRepository(db influxdb2.Client, org string, bucket string) (*RepositoryImpl, error) {
	return &RepositoryImpl{
		writeClient: db.WriteAPI(org, bucket),
		readClient:  db.QueryAPI(org),
	}, nil
}

func (r *RepositoryImpl) Write(monitorID string, orgID string, data map[string]interface{}) {
	p := influxdb2.NewPoint("http-probe", map[string]string{"monitorID": monitorID, "orgID": orgID}, data, time.Now())

	r.writeClient.WritePoint(p)

	r.writeClient.Flush()
}

func (r *RepositoryImpl) Read(bucket string, measurement string, tag0 string, tag1 string) (string, error) {
	query := fmt.Sprintf(
		`from (bucket: %s) 
		|> range(start: -1h) 
		|> filter(fn: (r) => r._measurement == %s and r.tag == %s and r._tag == %s )`,
		bucket, measurement, tag0, tag1,
	)

	result, err := r.readClient.Query(context.Background(), query)
	if err != nil {
		return "", err
	}

	return result.TableMetadata().String(), nil
}
