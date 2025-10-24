package events

import (
	"github.com/adiii717/kube-ai-sre-agent/pkg/config"
	corev1 "k8s.io/api/core/v1"
)

// EventType represents the type of Kubernetes event
type EventType string

const (
	CrashLoopBackOff  EventType = "CrashLoopBackOff"
	ImagePullBackOff  EventType = "ImagePullBackOff"
	HealthCheckFailure EventType = "HealthCheckFailure"
	OOMKilled         EventType = "OOMKilled"
)

// PodIncident represents a pod incident that needs analysis
type PodIncident struct {
	PodName      string
	Namespace    string
	EventType    EventType
	Reason       string
	Message      string
	ContainerName string
}

// Detector detects and filters pod incidents
type Detector struct {
	config *config.EventsConfig
}

// NewDetector creates a new event detector
func NewDetector(cfg *config.EventsConfig) *Detector {
	return &Detector{
		config: cfg,
	}
}

// ShouldProcess determines if an event should be processed
func (d *Detector) ShouldProcess(eventType EventType) bool {
	switch eventType {
	case CrashLoopBackOff:
		return d.config.CrashLoopBackOff
	case ImagePullBackOff:
		return d.config.ImagePullBackOff
	case HealthCheckFailure:
		return d.config.HealthCheckFailure
	case OOMKilled:
		return d.config.OOMKilled
	default:
		return false
	}
}

// DetectIncident analyzes a pod and returns incidents if any
func (d *Detector) DetectIncident(pod *corev1.Pod) *PodIncident {
	// Check pod status
	if pod.Status.Phase == corev1.PodFailed {
		return &PodIncident{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			EventType: CrashLoopBackOff,
			Reason:    pod.Status.Reason,
			Message:   pod.Status.Message,
		}
	}

	// Check container statuses
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if waiting := containerStatus.State.Waiting; waiting != nil {
			incident := d.detectFromWaiting(pod, containerStatus.Name, waiting)
			if incident != nil {
				return incident
			}
		}

		if terminated := containerStatus.State.Terminated; terminated != nil {
			incident := d.detectFromTerminated(pod, containerStatus.Name, terminated)
			if incident != nil {
				return incident
			}
		}
	}

	return nil
}

func (d *Detector) detectFromWaiting(pod *corev1.Pod, containerName string, waiting *corev1.ContainerStateWaiting) *PodIncident {
	switch waiting.Reason {
	case "CrashLoopBackOff":
		if d.config.CrashLoopBackOff {
			return &PodIncident{
				PodName:       pod.Name,
				Namespace:     pod.Namespace,
				EventType:     CrashLoopBackOff,
				Reason:        waiting.Reason,
				Message:       waiting.Message,
				ContainerName: containerName,
			}
		}
	case "ImagePullBackOff", "ErrImagePull":
		if d.config.ImagePullBackOff {
			return &PodIncident{
				PodName:       pod.Name,
				Namespace:     pod.Namespace,
				EventType:     ImagePullBackOff,
				Reason:        waiting.Reason,
				Message:       waiting.Message,
				ContainerName: containerName,
			}
		}
	}
	return nil
}

func (d *Detector) detectFromTerminated(pod *corev1.Pod, containerName string, terminated *corev1.ContainerStateTerminated) *PodIncident {
	if terminated.Reason == "OOMKilled" && d.config.OOMKilled {
		return &PodIncident{
			PodName:       pod.Name,
			Namespace:     pod.Namespace,
			EventType:     OOMKilled,
			Reason:        terminated.Reason,
			Message:       terminated.Message,
			ContainerName: containerName,
		}
	}
	return nil
}
