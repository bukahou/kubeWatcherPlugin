package cluster

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// Event K8s Event 资源模型
type Event struct {
	model_v3.CommonMeta
	Type           string               `json:"type"`
	Reason         string               `json:"reason"`
	Message        string               `json:"message"`
	Source         string               `json:"source,omitempty"`
	InvolvedObject model_v3.ResourceRef `json:"involvedObject"`
	Count          int32                `json:"count"`
	FirstTimestamp time.Time            `json:"firstTimestamp"`
	LastTimestamp  time.Time            `json:"lastTimestamp"`
}

func (e *Event) IsWarning() bool { return e.Type == "Warning" }
func (e *Event) IsNormal() bool  { return e.Type == "Normal" }

func (e *Event) IsCritical() bool {
	if e.Type != "Warning" {
		return false
	}
	switch e.Reason {
	case "Failed", "FailedScheduling", "FailedMount", "FailedAttachVolume",
		"OOMKilled", "BackOff", "CrashLoopBackOff", "Unhealthy", "NodeNotReady":
		return true
	}
	return false
}

func (e *Event) GetSeverity() string {
	if e.IsCritical() {
		return string(model_v3.HealthStatusCritical)
	}
	if e.IsWarning() {
		return string(model_v3.HealthStatusWarning)
	}
	return "info"
}

func (e *Event) MatchesResource(kind, namespace, name string) bool {
	return e.InvolvedObject.Kind == kind &&
		e.InvolvedObject.Namespace == namespace &&
		e.InvolvedObject.Name == name
}
