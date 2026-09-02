package monitors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

func getMetricsPredictions(monitorID uint, metrics *[]check.AggMetric) (*ForecasterPredictResponse, error) {
	if len(*metrics) == 0 {
		return &ForecasterPredictResponse{}, nil
	}

	timings := make([]ForecasterTimingMetric, len(*metrics))
	for i, c := range *metrics {
		timings[i] = ForecasterTimingMetric{
			ResponseTime:     c.Total,
			DNSLookup:        c.DNS,
			TCPConnection:    c.TCP,
			TLSHandshake:     c.TLS,
			ServerProcessing: c.Processing,
			ContentTransfer:  c.Transfer,
			CreatedAt:        c.Start,
		}
	}

	reqBody, err := json.Marshal(ChecksPredictRequest{
		MonitorID: int(monitorID),
		Timings:   timings,
	})
	if err != nil {
		return nil, err
	}

	forecasterURL := os.Getenv("FORECASTER_API_URL")
	if forecasterURL == "" {
		forecasterURL = "http://forecaster-api:8000"
	}
	url := fmt.Sprintf("%s/predict", forecasterURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		// Fallback to localhost if running outside docker network
		url = "http://localhost:8000/predict"
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecaster api returned status %d", resp.StatusCode)
	}

	var forecasterResp ForecasterPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecasterResp); err != nil {
		return nil, err
	}

	return &forecasterResp, nil
}

type ForecasterForecastRequest struct {
	MonitorID  int      `json:"monitor_id"`
	Timestamps []string `json:"timestamps"`
}

type ForecasterForecastResponse struct {
	Predictions []float64 `json:"predictions"`
	UpperBounds []float64 `json:"upper_bounds"`
	LowerBounds []float64 `json:"lower_bounds"`
}

func getMetricsForecast(monitorID uint, timestamps []string) (*ForecasterForecastResponse, error) {
	if len(timestamps) == 0 {
		return &ForecasterForecastResponse{}, nil
	}
	reqBody, err := json.Marshal(ForecasterForecastRequest{
		MonitorID:  int(monitorID),
		Timestamps: timestamps,
	})
	if err != nil {
		return nil, err
	}

	forecasterURL := os.Getenv("FORECASTER_API_URL")
	if forecasterURL == "" {
		forecasterURL = "http://forecaster-api:8000"
	}
	url := fmt.Sprintf("%s/forecast", forecasterURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		url = "http://localhost:8000/forecast"
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecaster api returned status %d", resp.StatusCode)
	}

	var forecasterResp ForecasterForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecasterResp); err != nil {
		return nil, err
	}

	return &forecasterResp, nil
}

type GetMonitorMetricsRequest struct {
	TeamID    uint    `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint    `param:"monitorId" validate:"required,numeric,gte=0"`
	Start     *string `query:"start"`
	End       *string `query:"end"`
}

type GetMonitorMetricsRespone struct {
	Metrics []GetMonitorMetricsResponseMetric `json:"metrics"`
}

type GetMonitorMetricsResponseMetric struct {
	Name string           `json:"name"`
	Data []MonitorMetrics `json:"timing"`
}
type MonitorMetrics struct {
	Start  string        `json:"start"`
	Timing time.Duration `json:"timing"`
}

func (h *Handlers) GetMonitorMetrics(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorMetricsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	metrics, err := h.CheckService.GetMonitorMetricsByMonitorID(
		ctx,
		req.MonitorID,
		req.Start,
		req.End,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	predictions, predErr := getMetricsPredictions(req.MonitorID, metrics)
	hasPredictions := predErr == nil && predictions != nil && len(predictions.Predictions) == len(*metrics)
	if predErr != nil {
		c.Log.WithError(predErr).Warn("failed to fetch predictions from forecaster")
	}

	metrics_list := []string{"DNS", "TCP", "TLS", "Processing", "Transfer", "Total", "Overhead"}

	var futureTimestamps []string
	now := time.Now()
	for i := 1; i <= 24; i++ {
		futureTimestamps = append(futureTimestamps, now.Add(time.Hour*time.Duration(i)).Format(time.RFC3339))
	}

	forecast, forecastErr := getMetricsForecast(req.MonitorID, futureTimestamps)
	hasForecast := forecastErr == nil && forecast != nil && len(forecast.Predictions) == len(futureTimestamps)
	if forecastErr != nil {
		c.Log.WithError(forecastErr).Warn("failed to fetch forecast from forecaster")
	}

	if hasPredictions || hasForecast {
		metrics_list = append(metrics_list, "Expected", "Upper Limit", "Lower Limit", "Anomaly")
	}

	metricResp := make([]GetMonitorMetricsResponseMetric, len(metrics_list))
	for i, metric := range metrics_list {
		metricResp[i] = GetMonitorMetricsResponseMetric{Name: metric, Data: []MonitorMetrics{}}
	}

	for i, c := range *metrics {
		overhead := c.Total - (c.DNS + c.TCP + c.TLS + c.Processing + c.Transfer)
		if overhead < 0 {
			overhead = 0
		}

		metricResp[0].Data = append(metricResp[0].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.DNS)})
		metricResp[1].Data = append(metricResp[1].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TCP)})
		metricResp[2].Data = append(metricResp[2].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TLS)})
		metricResp[3].Data = append(metricResp[3].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Processing)})
		metricResp[4].Data = append(metricResp[4].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Transfer)})
		metricResp[5].Data = append(metricResp[5].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Total)})
		metricResp[6].Data = append(metricResp[6].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(overhead)})

		if hasPredictions {
			metricResp[7].Data = append(metricResp[7].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(predictions.Predictions[i])})
			metricResp[8].Data = append(metricResp[8].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(predictions.UpperBounds[i])})
			metricResp[9].Data = append(metricResp[9].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(predictions.LowerBounds[i])})

			var anomalyVal float64 = -1
			if predictions.Anomalies[i] {
				anomalyVal = c.Total
			}
			metricResp[10].Data = append(metricResp[10].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(anomalyVal)})
		}
	}

	if hasForecast {
		for i, ts := range futureTimestamps {
			metricResp[5].Data = append(metricResp[5].Data, MonitorMetrics{Start: ts, Timing: time.Duration(forecast.Predictions[i])})
			metricResp[6].Data = append(metricResp[6].Data, MonitorMetrics{Start: ts, Timing: time.Duration(forecast.UpperBounds[i])})
			metricResp[7].Data = append(metricResp[7].Data, MonitorMetrics{Start: ts, Timing: time.Duration(forecast.LowerBounds[i])})
		}
	}

	return c.JSON(http.StatusOK, GetMonitorMetricsRespone{Metrics: metricResp})
}
