package scanner

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"catbox-scanner/internals/config"
	"catbox-scanner/internals/database"
	"catbox-scanner/internals/metrics"
	"catbox-scanner/internals/utils"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/dnscache"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
const id_len = 6

type Scanner struct {
	metrics   *metrics.Metrics
	db        *database.Database
	pool      *ants.Pool
	isRunning *bool
	config    *config.Config
	client    *http.Client
}

func NewScanner(cfg *config.Config, metrics *metrics.Metrics, db *database.Database, pool *ants.Pool, isRunning *bool) *Scanner {
	r := &dnscache.Resolver{}
	client := &http.Client{
		Timeout: cfg.Scanner.RequestTimeout,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 0,
			MaxConnsPerHost:     0,
			ForceAttemptHTTP2:   true,
			//Dns caching
			DialContext: func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := r.LookupHost(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					var dialer net.Dialer
					conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
					if err == nil {
						break
					}
				}
				return
			},
		},
	}

	return &Scanner{
		metrics:   metrics,
		db:        db,
		pool:      pool,
		isRunning: isRunning,
		config:    cfg,
		client:    client,
	}
}

func (s *Scanner) scanWorker(id string) {
	for _, ext := range s.config.Scanner.Exts {
		exists, err := s.checkFileExists(id, ext)
		if err != nil {
			continue
		}

		if exists {
			s.db.SaveValidLink(id, ext)
			s.metrics.IncrementLinksFound()
		}
	}
}

func (s *Scanner) StartScanning() {
	for *s.isRunning && !s.pool.IsClosed() {
		id := utils.GenerateRandomID(id_len, charset)
		err := s.pool.Submit(func() {
			s.scanWorker(id)
		})

		if err != nil {
			// pool is closed
			break
		}
	}
}

func (s *Scanner) checkFileExists(id, ext string) (bool, error) {
	if id == "" || ext == "" {
		return false, fmt.Errorf("id or extension is empty")
	}

	url := fmt.Sprintf("%s%s.%s", "https://files.catbox.moe/", id, ext)

	if s.client == nil {
		return false, fmt.Errorf("http client is not initialized")
	}

	resp, err := s.client.Head(url)
	if err != nil {
		return false, fmt.Errorf("file check failed for URL: %s, error: %v", url, err)
	}
	defer resp.Body.Close()

	s.metrics.IncrementRequestsSent()
	return resp.StatusCode == http.StatusOK, nil
}
