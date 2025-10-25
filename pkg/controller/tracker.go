package controller

import (
	"sync"
	"time"

	"github.com/adiii717/kube-ai-sre-agent/pkg/events"
	"k8s.io/klog/v2"
)

// IncidentRecord tracks an incident's history
type IncidentRecord struct {
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int
	Silenced  bool
	SilencedUntil time.Time
}

// IncidentTracker tracks recent incidents to prevent spam
type IncidentTracker struct {
	incidents       sync.Map // map[string]*IncidentRecord
	cooldown        time.Duration
	escalationEnabled bool
	escalationThreshold int
	silenceDuration time.Duration
}

// NewIncidentTracker creates a new tracker with cooldown period
func NewIncidentTracker(cooldown time.Duration, escalationEnabled bool, escalationThreshold int, silenceDuration time.Duration) *IncidentTracker {
	tracker := &IncidentTracker{
		cooldown:            cooldown,
		escalationEnabled:   escalationEnabled,
		escalationThreshold: escalationThreshold,
		silenceDuration:     silenceDuration,
	}

	// Cleanup old entries every minute
	go tracker.cleanup()

	return tracker
}

// ShouldAnalyze checks if incident should be analyzed (not seen recently)
func (t *IncidentTracker) ShouldAnalyze(incident *events.PodIncident) bool {
	// Create unique key: namespace/podname/eventtype
	key := incident.Namespace + "/" + incident.PodName + "/" + string(incident.EventType)

	now := time.Now()

	// Load or create record
	recordInterface, _ := t.incidents.LoadOrStore(key, &IncidentRecord{
		FirstSeen: now,
		LastSeen:  now,
		Count:     0,
	})
	record := recordInterface.(*IncidentRecord)

	// Check if silenced (escalated)
	if record.Silenced && now.Before(record.SilencedUntil) {
		klog.V(2).Infof("Incident %s is silenced until %v (occurred %d times)", key, record.SilencedUntil, record.Count)
		return false
	}

	// Check cooldown period
	if now.Sub(record.LastSeen) < t.cooldown {
		// Within cooldown - increment count
		record.Count++
		record.LastSeen = now

		// Check for escalation
		if t.escalationEnabled && record.Count >= t.escalationThreshold {
			record.Silenced = true
			record.SilencedUntil = now.Add(t.silenceDuration)
			klog.Warningf("Incident %s occurred %d times - silencing for %v", key, record.Count, t.silenceDuration)
		}

		return false // Skip analysis (too recent)
	}

	// Cooldown expired - reset and allow analysis
	record.LastSeen = now
	record.Count = 1
	record.Silenced = false
	t.incidents.Store(key, record)

	return true
}

// cleanup removes old entries periodically
func (t *IncidentTracker) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		t.incidents.Range(func(key, value interface{}) bool {
			record := value.(*IncidentRecord)

			// Remove if:
			// 1. Not silenced and cooldown expired (2x cooldown for safety)
			// 2. Silenced period expired and cooldown also expired
			if !record.Silenced && now.Sub(record.LastSeen) > t.cooldown*2 {
				t.incidents.Delete(key)
			} else if record.Silenced && now.After(record.SilencedUntil) && now.Sub(record.LastSeen) > t.cooldown {
				t.incidents.Delete(key)
			}

			return true
		})
	}
}
