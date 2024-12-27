package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	notifyCh  chan *NotifyRequest
	client    *http.Client
}

type NotifyRequest struct {
	ID  string
	Ext string
}

func NewScanner(cfg *config.Config, metrics *metrics.Metrics, db *database.Database, pool *ants.Pool, isRunning *bool) *Scanner {
	var notifyCh chan *NotifyRequest
	if cfg.Server.ServerEnabled {
		notifyCh = make(chan *NotifyRequest, 1000)
	}

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
		notifyCh:  notifyCh,
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
			if s.config.Server.ServerEnabled {
				s.notifyCh <- &NotifyRequest{ID: id, Ext: ext}
			}
		}
	}
}

func (s *Scanner) StartScanning() {
	if s.config.Server.ServerEnabled && s.notifyCh != nil {
		go s.notifyServerLoop()
	}

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

	url := fmt.Sprintf("%s%s%s", "https://files.catbox.moe/", id, ext)

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

func (s *Scanner) notifyServerLoop() {
	for notifyReq := range s.notifyCh {
		err := s.notifyServer(notifyReq.ID, notifyReq.Ext)
		if err != nil {
			fmt.Printf("Failed to notify the server: %v\n", err)
		}
	}
}

func (s *Scanner) notifyServer(id, ext string) error {
	url := fmt.Sprintf("https://files.catbox.moe/%s%s", id, ext)
	payload := map[string]string{
		"url": url,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.Server.ServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
