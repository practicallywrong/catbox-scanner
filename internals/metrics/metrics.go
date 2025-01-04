package metrics

import (
	"fmt"
	"sync"
	"time"
)

type Metrics struct {
	sync.Mutex
	RequestsSent   int
	LinksFound     int
	ReqPerSec      int
	FoundPerMin    int
	RPSHistory     []int
	MaxHistorySize int
}

func NewMetrics(maxHistorySize int) *Metrics {
	return &Metrics{
		RPSHistory:     make([]int, 0, maxHistorySize),
		MaxHistorySize: maxHistorySize,
	}
}

func (m *Metrics) IncrementRequestsSent() {
	m.Lock()
	m.RequestsSent++
	m.Unlock()
}

func (m *Metrics) IncrementLinksFound() {
	m.Lock()
	m.LinksFound++
	m.Unlock()
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

func (m *Metrics) StartPrintLoop(isRunning *bool) {
	var lastRequestsSent, linksFoundLastMin int
	secTicker := time.NewTicker(1 * time.Second)
	minTicker := time.NewTicker(1 * time.Minute)
	defer secTicker.Stop()
	defer minTicker.Stop()

	for *isRunning {
		select {
		case <-secTicker.C:
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
