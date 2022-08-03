package http

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/jeremywohl/flatten"
	httpProbe "github.com/opsway-io/backend/internal/probes/http"
	"github.com/sirupsen/logrus"
)

type Service interface {
	WriteResult(monitorID string, orgID string, res *httpProbe.Result)
	GetResult(bucket string, measurement string, tag0 string, tag1 string) (string, error)
}

type ServiceImpl struct {
	writeClient api.WriteAPI
	readClient  api.QueryAPI
}

func NewService(db influxdb2.Client, org string, bucket string) (Service, error) {
	return &ServiceImpl{
		writeClient: db.WriteAPI(org, bucket),
		readClient:  db.QueryAPI(org),
	}, nil
}

func (r *ServiceImpl) WriteResult(monitorID string, orgID string, res *httpProbe.Result) {
	m := structs.Map(res)
	m, err := flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		logrus.WithError(err).Fatal(err)
	}

	p := influxdb2.NewPoint("http-probe", map[string]string{"monitorID": monitorID, "orgID": orgID}, m, time.Now())

	r.writeClient.WritePoint(p)

	r.writeClient.Flush()
}

func (r *ServiceImpl) GetResult(bucket string, measurement string, tag0 string, tag1 string) (string, error) {
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
