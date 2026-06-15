package httpserver

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"GoFaux/internal/mock"
)

const maxTrafficEvents = 500

type TrafficEvent struct {
	ID             int64             `json:"id"`
	Time           time.Time         `json:"time"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Query          string            `json:"query,omitempty"`
	Status         int               `json:"status"`
	Matched        bool              `json:"matched"`
	MockID         string            `json:"mock_id,omitempty"`
	MockName       string            `json:"mock_name,omitempty"`
	MockEndpoint   string            `json:"mock_endpoint,omitempty"`
	DurationMS     int64             `json:"duration_ms"`
	RequestBytes   int               `json:"request_bytes"`
	ResponseBytes  int               `json:"response_bytes"`
	Body           string            `json:"body,omitempty"`
	BodyTruncated  bool              `json:"body_truncated,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	RemoteAddr     string            `json:"remote_addr,omitempty"`
	ContentType    string            `json:"content_type,omitempty"`
	UserAgent      string            `json:"user_agent,omitempty"`
	PathParameters map[string]string `json:"path_parameters,omitempty"`
}

type TrafficSummary struct {
	Total        int            `json:"total"`
	Matched      int            `json:"matched"`
	Missed       int            `json:"missed"`
	Recent       []TrafficEvent `json:"recent"`
	ByMethod     []MetricCount  `json:"by_method"`
	ByStatus     []MetricCount  `json:"by_status"`
	TopPaths     []MetricCount  `json:"top_paths"`
	Timeline     []MetricCount  `json:"timeline"`
	AverageMS    int64          `json:"average_ms"`
	RequestBytes int            `json:"request_bytes"`
}

type MetricCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func (s *Server) recordTraffic(event TrafficEvent) {
	s.trafficMu.Lock()
	defer s.trafficMu.Unlock()
	s.nextTrafficID++
	event.ID = s.nextTrafficID
	s.traffic = append([]TrafficEvent{event}, s.traffic...)
	if len(s.traffic) > maxTrafficEvents {
		s.traffic = s.traffic[:maxTrafficEvents]
	}
}

func (s *Server) trafficSnapshot(limit int) []TrafficEvent {
	s.trafficMu.RLock()
	defer s.trafficMu.RUnlock()
	if limit <= 0 || limit > len(s.traffic) {
		limit = len(s.traffic)
	}
	out := make([]TrafficEvent, limit)
	copy(out, s.traffic[:limit])
	return out
}

func (s *Server) clearTraffic() {
	s.trafficMu.Lock()
	defer s.trafficMu.Unlock()
	s.traffic = nil
	s.nextTrafficID = 0
}

func (s *Server) trafficSummary() TrafficSummary {
	events := s.trafficSnapshot(maxTrafficEvents)
	summary := TrafficSummary{Total: len(events), Recent: events}
	byMethod := map[string]int{}
	byStatus := map[string]int{}
	byPath := map[string]int{}
	timeline := map[string]int{}
	var totalMS int64

	for _, event := range events {
		if event.Matched {
			summary.Matched++
		} else {
			summary.Missed++
		}
		byMethod[event.Method]++
		byStatus[strconv.Itoa(event.Status)]++
		byPath[event.Path]++
		bucket := event.Time.Local().Format("15:04")
		timeline[bucket]++
		totalMS += event.DurationMS
		summary.RequestBytes += event.RequestBytes
	}
	if len(events) > 0 {
		summary.AverageMS = totalMS / int64(len(events))
	}
	summary.ByMethod = sortedCounts(byMethod, 0)
	summary.ByStatus = sortedCounts(byStatus, 0)
	summary.TopPaths = sortedCounts(byPath, 8)
	summary.Timeline = sortedTimeline(timeline)
	return summary
}

func sortedCounts(values map[string]int, limit int) []MetricCount {
	counts := make([]MetricCount, 0, len(values))
	for label, count := range values {
		counts = append(counts, MetricCount{Label: label, Count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count == counts[j].Count {
			return counts[i].Label < counts[j].Label
		}
		return counts[i].Count > counts[j].Count
	})
	if limit > 0 && len(counts) > limit {
		return counts[:limit]
	}
	return counts
}

func sortedTimeline(values map[string]int) []MetricCount {
	counts := make([]MetricCount, 0, len(values))
	for label, count := range values {
		counts = append(counts, MetricCount{Label: label, Count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Label < counts[j].Label
	})
	if len(counts) > 24 {
		return counts[len(counts)-24:]
	}
	return counts
}

func trafficEventFromRequest(r *http.Request, status int, matched bool, def mock.Definition, match mock.Match, started time.Time, requestBody string, requestBodyBytes int, bodyTruncated bool, responseBytes int) TrafficEvent {
	event := TrafficEvent{
		Time:           time.Now(),
		Method:         r.Method,
		Path:           r.URL.Path,
		Query:          r.URL.RawQuery,
		Status:         status,
		Matched:        matched,
		DurationMS:     time.Since(started).Milliseconds(),
		RequestBytes:   requestBodyBytes,
		ResponseBytes:  responseBytes,
		Body:           requestBody,
		BodyTruncated:  bodyTruncated,
		Headers:        requestHeadersForLog(r),
		RemoteAddr:     r.RemoteAddr,
		ContentType:    r.Header.Get("Content-Type"),
		UserAgent:      r.Header.Get("User-Agent"),
		PathParameters: match.PathParams,
	}
	if matched {
		event.MockID = def.ID
		event.MockName = def.Name
		event.MockEndpoint = def.Endpoint
	}
	return event
}

func requestHeadersForLog(r *http.Request) map[string]string {
	if len(r.Header) == 0 {
		return nil
	}
	headers := map[string]string{}
	for key, values := range r.Header {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}
	return headers
}
