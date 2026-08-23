package cluster

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// ============================================================
// Service 模型
// ============================================================

// Service K8s Service 资源模型
type Service struct {
	Summary     ServiceSummary    `json:"summary"`
	Spec        ServiceSpec       `json:"spec"`
	Ports       []ServicePort     `json:"ports,omitempty"`
	Selector    map[string]string `json:"selector,omitempty"`
	Network     ServiceNetwork    `json:"network"`
	Backends    *ServiceBackends  `json:"backends,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ServiceSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	PortsCount   int       `json:"portsCount"`
	HasSelector  bool      `json:"hasSelector"`
	Badges       []string  `json:"badges,omitempty"`
	ClusterIP    string    `json:"clusterIP,omitempty"`
	ExternalName string    `json:"externalName,omitempty"`
}

type ServiceSpec struct {
	Type                          string   `json:"type"`
	SessionAffinity               string   `json:"sessionAffinity,omitempty"`
	SessionAffinityTimeoutSeconds *int32   `json:"sessionAffinityTimeoutSeconds,omitempty"`
	ExternalTrafficPolicy         string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy         string   `json:"internalTrafficPolicy,omitempty"`
	IPFamilies                    []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy                string   `json:"ipFamilyPolicy,omitempty"`
	ClusterIPs                    []string `json:"clusterIPs,omitempty"`
	ExternalIPs                   []string `json:"externalIPs,omitempty"`
	LoadBalancerClass             string   `json:"loadBalancerClass,omitempty"`
	LoadBalancerSourceRanges      []string `json:"loadBalancerSourceRanges,omitempty"`
	PublishNotReadyAddresses      bool     `json:"publishNotReadyAddresses,omitempty"`
	AllocateLoadBalancerNodePorts *bool    `json:"allocateLoadBalancerNodePorts,omitempty"`
	HealthCheckNodePort           int32    `json:"healthCheckNodePort,omitempty"`
	ExternalName                  string   `json:"externalName,omitempty"`
}

type ServicePort struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type ServiceNetwork struct {
	ClusterIPs            []string `json:"clusterIPs,omitempty"`
	ExternalIPs           []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress   []string `json:"loadBalancerIngress,omitempty"`
	IPFamilies            []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy        string   `json:"ipFamilyPolicy,omitempty"`
	ExternalTrafficPolicy string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy string   `json:"internalTrafficPolicy,omitempty"`
}

type ServiceBackends struct {
	Summary   BackendSummary    `json:"summary"`
	Ports     []EndpointPort    `json:"ports,omitempty"`
	Endpoints []BackendEndpoint `json:"endpoints,omitempty"`
}

type BackendSummary struct {
	Ready    int       `json:"ready"`
	NotReady int       `json:"notReady"`
	Total    int       `json:"total"`
	Slices   int       `json:"slices,omitempty"`
	Updated  time.Time `json:"updated,omitempty"`
}

type EndpointPort struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type BackendEndpoint struct {
	Address   string           `json:"address"`
	Ready     bool             `json:"ready"`
	NodeName  string           `json:"nodeName,omitempty"`
	Zone      string           `json:"zone,omitempty"`
	TargetRef *model_v3.K8sRef `json:"targetRef,omitempty"`
}

func (s *Service) GetName() string      { return s.Summary.Name }
func (s *Service) GetNamespace() string { return s.Summary.Namespace }
func (s *Service) GetType() string      { return s.Summary.Type }
func (s *Service) IsClusterIP() bool    { return s.Summary.Type == "ClusterIP" }
func (s *Service) IsNodePort() bool     { return s.Summary.Type == "NodePort" }
func (s *Service) IsLoadBalancer() bool { return s.Summary.Type == "LoadBalancer" }
func (s *Service) IsExternalName() bool { return s.Summary.Type == "ExternalName" }
func (s *Service) IsHeadless() bool     { return s.Summary.ClusterIP == "None" }
func (s *Service) HasEndpoints() bool   { return s.Backends != nil && s.Backends.Summary.Total > 0 }

// ============================================================
// Ingress 模型
// ============================================================

// Ingress K8s Ingress 资源模型
type Ingress struct {
	Summary     IngressSummary    `json:"summary"`
	Spec        IngressSpec       `json:"spec"`
	Status      IngressStatus     `json:"status"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	IngressClass string    `json:"ingressClass,omitempty"`
	HostsCount   int       `json:"hostsCount"`
	PathsCount   int       `json:"pathsCount"`
	TLSEnabled   bool      `json:"tlsEnabled"`
	Hosts        []string  `json:"hosts,omitempty"`
}

type IngressSpec struct {
	IngressClassName string          `json:"ingressClassName,omitempty"`
	DefaultBackend   *IngressBackend `json:"defaultBackend,omitempty"`
	Rules            []IngressRule   `json:"rules,omitempty"`
	TLS              []IngressTLS    `json:"tls,omitempty"`
}

type IngressStatus struct {
	LoadBalancer []string `json:"loadBalancer,omitempty"`
}

type IngressBackend struct {
	Type     string                 `json:"type"`
	Service  *IngressServiceBackend `json:"service,omitempty"`
	Resource *IngressResourceRef    `json:"resource,omitempty"`
}

type IngressServiceBackend struct {
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

type IngressResourceRef struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type IngressRule struct {
	Host  string        `json:"host,omitempty"`
	Paths []IngressPath `json:"paths"`
}

type IngressPath struct {
	Path     string          `json:"path"`
	PathType string          `json:"pathType"`
	Backend  *IngressBackend `json:"backend,omitempty"`
}

type IngressTLS struct {
	Hosts      []string `json:"hosts,omitempty"`
	SecretName string   `json:"secretName,omitempty"`
}

func (i *Ingress) GetName() string      { return i.Summary.Name }
func (i *Ingress) GetNamespace() string { return i.Summary.Namespace }
func (i *Ingress) GetHosts() []string   { return i.Summary.Hosts }
func (i *Ingress) HasTLS() bool         { return i.Summary.TLSEnabled }
