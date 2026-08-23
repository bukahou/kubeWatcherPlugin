package cluster

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// Deployment K8s Deployment 资源模型
type Deployment struct {
	Summary     DeploymentSummary  `json:"summary"`
	Spec        DeploymentSpec     `json:"spec"`
	Template    PodTemplate        `json:"template"`
	Status      DeploymentStatus   `json:"status"`
	Rollout     *DeploymentRollout `json:"rollout,omitempty"`
	ReplicaSets []ReplicaSetBrief  `json:"replicaSets,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
}

// DeploymentSummary 摘要信息
type DeploymentSummary struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Strategy    string    `json:"strategy"`
	Replicas    int32     `json:"replicas"`
	Updated     int32     `json:"updated"`
	Ready       int32     `json:"ready"`
	Available   int32     `json:"available"`
	Unavailable int32     `json:"unavailable,omitempty"`
	Paused      bool      `json:"paused,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Age         string    `json:"age"`
	Selector    string    `json:"selector,omitempty"`
}

// DeploymentSpec 规格
type DeploymentSpec struct {
	Replicas                *int32              `json:"replicas,omitempty"`
	Selector                *LabelSelector      `json:"selector,omitempty"`
	Strategy                *DeploymentStrategy `json:"strategy,omitempty"`
	MinReadySeconds         int32               `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit    *int32              `json:"revisionHistoryLimit,omitempty"`
	ProgressDeadlineSeconds *int32              `json:"progressDeadlineSeconds,omitempty"`
}

// DeploymentStrategy 更新策略
type DeploymentStrategy struct {
	Type          string                 `json:"type"`
	RollingUpdate *RollingUpdateStrategy `json:"rollingUpdate,omitempty"`
}

// RollingUpdateStrategy 滚动更新策略
type RollingUpdateStrategy struct {
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	MaxSurge       string `json:"maxSurge,omitempty"`
}

// DeploymentStatus 状态
type DeploymentStatus struct {
	ObservedGeneration  int64                 `json:"observedGeneration,omitempty"`
	Replicas            int32                 `json:"replicas"`
	UpdatedReplicas     int32                 `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32                 `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32                 `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32                 `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32                `json:"collisionCount,omitempty"`
	Conditions          []DeploymentCondition `json:"conditions,omitempty"`
}

// DeploymentCondition 状态条件
type DeploymentCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// DeploymentRollout 滚动更新状态
type DeploymentRollout struct {
	Phase   string   `json:"phase"`
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

// ReplicaSetBrief ReplicaSet 简要信息
type ReplicaSetBrief struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Revision  string    `json:"revision,omitempty"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	Available int32     `json:"available"`
	Image     string    `json:"image,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}

// ReplicaSet K8s ReplicaSet 完整模型
type ReplicaSet struct {
	model_v3.CommonMeta
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	Selector          map[string]string `json:"selector,omitempty"`
}

func (d *Deployment) GetName() string      { return d.Summary.Name }
func (d *Deployment) GetNamespace() string { return d.Summary.Namespace }
func (d *Deployment) IsHealthy() bool {
	return d.Summary.Ready == d.Summary.Replicas && d.Summary.Replicas > 0
}
func (d *Deployment) IsUpdating() bool { return d.Summary.Updated < d.Summary.Replicas }
func (d *Deployment) IsPaused() bool   { return d.Summary.Paused }
