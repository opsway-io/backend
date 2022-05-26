package influxdb

import (
	"context"
	"fmt"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

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

func (r *RepositoryImpl) Write(data string) {
	r.writeClient.WriteRecord(data)

	r.writeClient.Flush()
}

func (r *RepositoryImpl) Read(bucket string) (string, error) {

	query := fmt.Sprintf(`from(bucket:"%s"))`, bucket)

	result, err := r.readClient.Query(context.Background(), query)
	if err != nil {
		return "", err
	}

	return result.TableMetadata().String(), nil
}
