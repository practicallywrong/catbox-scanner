package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"catbox-scanner/internals/config"
	"catbox-scanner/internals/metrics"
)

type MasterServerClient struct {
	config     *config.Config
	client     *http.Client
	mu         sync.Mutex
	entryQueue []string
	endpoint   string
	enabled    bool
	metrics    *metrics.Metrics
}

func NewMasterServerClient(cfg *config.Config, metrics *metrics.Metrics) (*MasterServerClient, error) {
	if !cfg.MasterServer.Enabled {
		return nil, fmt.Errorf("master server is not enabled")
	}

	serverAddr := cfg.MasterServer.Endpoint
	client := &http.Client{
		Timeout: cfg.Scanner.RequestTimeout,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 0,
			MaxConnsPerHost:     0,
			ForceAttemptHTTP2:   true,
		},
	}

	masterClient := &MasterServerClient{
		config:   cfg,
		client:   client,
		endpoint: serverAddr,
		enabled:  cfg.MasterServer.Enabled,
		metrics:  metrics,
	}

	go masterClient.startSendingEntries()

	return masterClient, nil
}

func (msc *MasterServerClient) AddEntry(entry string) {
	msc.mu.Lock()
	defer msc.mu.Unlock()
	msc.entryQueue = append(msc.entryQueue, entry)
}

func (msc *MasterServerClient) startSendingEntries() {
	for {
		msc.mu.Lock()
		if len(msc.entryQueue) > 0 {
			entry := msc.entryQueue[0]
			msc.entryQueue = msc.entryQueue[1:]

			go msc.sendEntryToMaster(entry)
		}
		msc.mu.Unlock()
	}
}

func (msc *MasterServerClient) sendEntryToMaster(entry string) {
	if !msc.enabled {
		return
	}

	// Split entry into id and extension
	var id, ext string
	fmt.Sscanf(entry, "%[^.].%s", &id, &ext)
	if id == "" || ext == "" {
		fmt.Printf("Invalid entry format: %s\n", entry)
		return
	}

	url := fmt.Sprintf("%s/add?auth=%s", msc.endpoint, msc.config.MasterServer.AuthKey)

	payload := map[string]string{"id": id, "ext": ext}
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling entry: %v\n", err)
		return
	}

	resp, err := msc.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Error sending entry to master server: %v\n", err)
		msc.config.MasterServer.Enabled = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Master server responded with status code: %d\n", resp.StatusCode)
	}
}
