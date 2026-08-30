// per-user traffic usage metering: hot-path atomic counters accumulated in
// Server.id2Usage, flushed to the control plane (UsageAPI) by a ticker.
package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type usageCounter struct {
	up   uint64 // 客户端 → 公网（用户上行）
	down uint64 // 公网 → 客户端（用户下行）
}

// usageOf returns the shared per-id counter slot, creating it on first use.
func (s *Server) usageOf(id string) *usageCounter {
	if v, ok := s.id2Usage.Load(id); ok {
		return v.(*usageCounter)
	}
	v, _ := s.id2Usage.LoadOrCreate(id, func() interface{} { return &usageCounter{} })
	return v.(*usageCounter)
}

type usageReport struct {
	ID   string `json:"id"`
	Up   uint64 `json:"up"`
	Down uint64 `json:"down"`
}

// startUsageReporter periodically snapshots (swap-to-zero) all counters and
// POSTs them to the control plane. Failed reports re-add their counters so
// nothing is lost on transient outages.
func (s *Server) startUsageReporter() {
	if s.config.UsageAPI == "" {
		return
	}
	go func() {
		defer func() {
			if e := recover(); e != nil {
				s.Logger.Error().Msgf("usage reporter recovered: %#v", e)
			}
		}()
		client := &http.Client{Timeout: 10 * time.Second}
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if atomic.LoadUint32(&s.closing) == 1 {
				return
			}
			s.flushUsage(client)
		}
	}()
}

func (s *Server) flushUsage(client *http.Client) {
	reports := make([]usageReport, 0, 8)
	s.id2Usage.Range(func(key, value interface{}) bool {
		c := value.(*usageCounter)
		up := atomic.SwapUint64(&c.up, 0)
		down := atomic.SwapUint64(&c.down, 0)
		if up == 0 && down == 0 {
			return true
		}
		reports = append(reports, usageReport{ID: key.(string), Up: up, Down: down})
		return true
	})
	if len(reports) == 0 {
		return
	}

	body, err := json.Marshal(map[string]any{"reports": reports})
	if err != nil {
		s.reAddUsage(reports)
		return
	}
	req, err := http.NewRequest(http.MethodPost, s.config.UsageAPI, bytes.NewReader(body))
	if err != nil {
		s.reAddUsage(reports)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if s.config.UsageToken != "" {
		req.Header.Set("X-Node-Token", s.config.UsageToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.Warn().Err(err).Msg("usage report failed, will retry next tick")
		s.reAddUsage(reports)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.Logger.Warn().Int("status", resp.StatusCode).Msg("usage report rejected, will retry next tick")
		s.reAddUsage(reports)
	}
}

// reAddUsage puts swapped-out counters back after a failed report.
func (s *Server) reAddUsage(reports []usageReport) {
	for _, r := range reports {
		c := s.usageOf(r.ID)
		atomic.AddUint64(&c.up, r.Up)
		atomic.AddUint64(&c.down, r.Down)
	}
}

// flushUsageOnExit is called from Shutdown to not lose the final window.
func (s *Server) flushUsageOnExit() {
	if s.config.UsageAPI == "" {
		return
	}
	s.flushUsage(&http.Client{Timeout: 5 * time.Second})
}
