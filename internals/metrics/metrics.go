package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	sync.Mutex
	RequestsSent   int
	LinksFound     int
	ReqPerSec      int
	FoundPerMin    int
	RPSHistory     []int
	MaxHistorySize int

	requestsSentCounter prometheus.Counter
	linksFoundCounter   prometheus.Counter
}

func NewMetrics(maxHistorySize int) *Metrics {
	// Define Prometheus metrics
	requestsSentCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "requests_sent_total",
		Help: "Total number of requests sent",
	})
	linksFoundCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "links_found_total",
		Help: "Total number of links found",
	})
	reqPerSecGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "requests_per_second",
		Help: "Requests per second",
	})
	foundPerMinGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "found_per_minute",
		Help: "Links found per minute",
	})

	// Register metrics
	prometheus.MustRegister(requestsSentCounter)
	prometheus.MustRegister(linksFoundCounter)
	prometheus.MustRegister(reqPerSecGauge)
	prometheus.MustRegister(foundPerMinGauge)

	return &Metrics{
		RPSHistory:          make([]int, 0, maxHistorySize),
		MaxHistorySize:      maxHistorySize,
		requestsSentCounter: requestsSentCounter,
		linksFoundCounter:   linksFoundCounter,
	}
}

func (m *Metrics) IncrementRequestsSent() {
	m.Lock()
	m.RequestsSent++
	m.Unlock()
	m.requestsSentCounter.Inc()
}

func (m *Metrics) IncrementLinksFound() {
	m.Lock()
	m.LinksFound++
	m.Unlock()
	m.linksFoundCounter.Inc()
}

func (m *Metrics) calculateAverageRPS() int {
	m.Lock()
	defer m.Unlock()

	if len(m.RPSHistory) == 0 {
		return m.ReqPerSec
	}

	var totalRPS int
	for _, rps := range m.RPSHistory {
		totalRPS += rps
	}
	averageRPS := totalRPS / len(m.RPSHistory)

	return averageRPS
}

func (m *Metrics) StartPrintLoop() {
	var lastRequestsSent, linksFoundLastMin int
	secTicker := time.NewTicker(1 * time.Second)
	minTicker := time.NewTicker(1 * time.Minute)
	defer secTicker.Stop()
	defer minTicker.Stop()

	for {
		select {
		case <-secTicker.C:
			// Calculate RPS
			rps := m.RequestsSent - lastRequestsSent
			lastRequestsSent = m.RequestsSent
			m.ReqPerSec = rps

			m.Lock()
			m.RPSHistory = append(m.RPSHistory, rps)
			if len(m.RPSHistory) > m.MaxHistorySize {
				m.RPSHistory = m.RPSHistory[1:]
			}
			m.Unlock()

			avgRPS := m.calculateAverageRPS()

			fmt.Print("\033[2K\033[0G") // Clear the line
			fmt.Printf("Requests: %d | Links Found: %d | RPS: %d | Avg RPS: %d | FPM: %d",
				m.RequestsSent, m.LinksFound, m.ReqPerSec, avgRPS, m.FoundPerMin)

		case <-minTicker.C:
			m.FoundPerMin = m.LinksFound - linksFoundLastMin
			linksFoundLastMin = m.LinksFound
			fmt.Println("")
		}
	}
}

func StartPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9090", nil) // Expose the metrics on port 9090
}
