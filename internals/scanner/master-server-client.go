package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"catbox-scanner/internals/config"
	"catbox-scanner/internals/metrics"

	"golang.org/x/net/http2"
)

type MasterServerClient struct {
	config     *config.Config
	client     *http.Client
	mu         sync.Mutex
	entryQueue []string
	serverAddr string
	enabled    bool
	metrics    *metrics.Metrics
}

func NewMasterServerClient(cfg *config.Config, metrics *metrics.Metrics) (*MasterServerClient, error) {
	if !cfg.MasterServer.Enabled {
		return nil, fmt.Errorf("master server is not enabled")
	}

	serverAddr := cfg.MasterServer.Endpoint
	client := &http.Client{
		Transport: &http2.Transport{},
	}

	masterClient := &MasterServerClient{
		config:     cfg,
		client:     client,
		serverAddr: serverAddr,
		enabled:    cfg.MasterServer.Enabled,
		metrics:    metrics,
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

	url := fmt.Sprintf("%s/entries", msc.serverAddr)
	payload := map[string]string{"entry": entry}
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
}
