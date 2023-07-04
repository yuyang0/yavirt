package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

var (
	// DefaultLabels .
	DefaultLabels = []string{"host"}

	// MetricHeartbeatCount .
	MetricHeartbeatCount = "yavirt_heartbeat_total"
	// MetricErrorCount .
	MetricErrorCount   = "yavirt_error_total"
	MetricSvcTaskTotal = "yavirt_svc_task_total"
	MetricSvcTasks     = "yavirt_svc_task_count"

	metr *Metrics
)

func init() {
	hn := configs.Hostname()

	metr = New(hn)
	metr.RegisterCounter(MetricErrorCount, "yavirt errors", nil)               //nolint
	metr.RegisterCounter(MetricHeartbeatCount, "yavirt heartbeats", nil)       //nolint
	metr.RegisterCounter(MetricSvcTaskTotal, "yavirt service task total", nil) //nolint
	metr.RegisterGauge(MetricSvcTasks, "yavirt service tasks", nil)            //nolint
}

// Metrics .
type Metrics struct {
	host       string
	collectors map[string]prometheus.Collector
}

// New .
func New(host string) *Metrics {
	return &Metrics{
		host:       host,
		collectors: map[string]prometheus.Collector{},
	}
}

// RegisterCounter .
func (m *Metrics) RegisterCounter(name, desc string, labels []string) error {
	var col = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: desc,
		},
		utils.MergeStrings(labels, DefaultLabels),
	)

	if err := prometheus.Register(col); err != nil {
		return errors.Trace(err)
	}
	m.collectors[name] = col

	return nil
}

// RegisterGauge .
func (m *Metrics) RegisterGauge(name, desc string, labels []string) error {
	var col = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: desc,
		},
		utils.MergeStrings(labels, DefaultLabels),
	)

	if err := prometheus.Register(col); err != nil {
		return errors.Trace(err)
	}

	m.collectors[name] = col

	return nil
}

// Incr .
func (m *Metrics) Incr(name string, labels map[string]string) error {
	var collector, exists = m.collectors[name]
	if !exists {
		return errors.Errorf("collector %s not found", name)
	}

	labels = m.appendLabel(labels, "host", m.host)
	switch col := collector.(type) {
	case *prometheus.GaugeVec:
		col.With(labels).Inc()
	case *prometheus.CounterVec:
		col.With(labels).Inc()
	default:
		return errors.Errorf("collector %s is not counter or gauge", name)
	}

	return nil
}

// Decr .
func (m *Metrics) Decr(name string, labels map[string]string) error {
	var collector, exists = m.collectors[name]
	if !exists {
		return errors.Errorf("collector %s not found", name)
	}

	labels = m.appendLabel(labels, "host", m.host)
	switch col := collector.(type) {
	case *prometheus.GaugeVec:
		col.With(labels).Dec()
	default:
		return errors.Errorf("collector %s is not gauge", name)
	}

	return nil
}

// Store .
func (m *Metrics) Store(name string, value float64, labels map[string]string) error {
	var collector, exists = m.collectors[name]
	if !exists {
		return errors.Errorf("collector %s not found", name)
	}

	labels = m.appendLabel(labels, "host", m.host)
	switch col := collector.(type) {
	case *prometheus.GaugeVec:
		col.With(labels).Set(value)
	default:
		return errors.Errorf("collector %s is not gauge", name)
	}

	return nil
}

func (m *Metrics) appendLabel(labels map[string]string, key, value string) map[string]string {
	if labels != nil {
		labels[key] = value
	} else {
		labels = map[string]string{key: value}
	}
	return labels
}

// Handler .
func Handler() http.Handler {
	return promhttp.Handler()
}

// IncrError .
func IncrError() {
	Incr(MetricErrorCount, nil) //nolint
}

// IncrHeartbeat .
func IncrHeartbeat() {
	Incr(MetricHeartbeatCount, nil) //nolint
}

// Incr .
func Incr(name string, labels map[string]string) error {
	return metr.Incr(name, labels)
}

func Decr(name string, labels map[string]string) error {
	return metr.Decr(name, labels)
}

// Store .
func Store(name string, value float64, labels map[string]string) error {
	return metr.Store(name, value, labels)
}

// RegisterGauge .
func RegisterGauge(name, desc string, labels []string) error {
	return metr.RegisterGauge(name, desc, labels)
}

// RegisterCounter .
func RegisterCounter(name, desc string, labels []string) error {
	return metr.RegisterCounter(name, desc, labels)
}
