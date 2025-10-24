package controller

import (
	"sync"
	"time"

	"github.com/adiii717/kube-ai-sre-agent/pkg/events"
)

// IncidentTracker tracks recent incidents to prevent spam
type IncidentTracker struct {
	incidents sync.Map // map[string]time.Time
	cooldown  time.Duration
}

// NewIncidentTracker creates a new tracker with cooldown period
func NewIncidentTracker(cooldown time.Duration) *IncidentTracker {
	tracker := &IncidentTracker{
		cooldown: cooldown,
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

	// Check if incident was seen recently
	if lastSeen, exists := t.incidents.Load(key); exists {
		lastTime := lastSeen.(time.Time)
		if now.Sub(lastTime) < t.cooldown {
			return false // Too recent, skip
		}
	}

	// Record this incident
	t.incidents.Store(key, now)
	return true
}

// cleanup removes old entries periodically
func (t *IncidentTracker) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		t.incidents.Range(func(key, value interface{}) bool {
			lastTime := value.(time.Time)
			if now.Sub(lastTime) > t.cooldown*2 {
				t.incidents.Delete(key)
			}
			return true
		})
	}
}
