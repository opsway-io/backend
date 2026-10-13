package check

// MonitorPerformance holds aggregated response time statistics (milliseconds) for a monitor.
type MonitorPerformance struct {
	MonitorID           uint
	AverageResponseTime float32
	P99                 float32
	P95                 float32
}
