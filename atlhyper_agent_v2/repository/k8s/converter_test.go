package k8s

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// =============================================================================
// Helper constructors
// =============================================================================

// stableTime returns a fixed time 10 days ago for stable age assertions.
func stableTime() time.Time {
	return time.Now().Add(-10 * 24 * time.Hour)
}

func makeK8sPod(name, ns, phase string, opts ...func(*corev1.Pod)) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: corev1.PodSpec{},
		Status: corev1.PodStatus{
			Phase: corev1.PodPhase(phase),
		},
	}
	for _, fn := range opts {
		fn(pod)
	}
	return pod
}

func makeK8sNode(name string, opts ...func(*corev1.Node)) *corev1.Node {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{},
		},
		Spec: corev1.NodeSpec{},
		Status: corev1.NodeStatus{
			Capacity:    corev1.ResourceList{},
			Allocatable: corev1.ResourceList{},
		},
	}
	for _, fn := range opts {
		fn(node)
	}
	return node
}

func makeK8sDeployment(name, ns string, replicas int32, opts ...func(*appsv1.Deployment)) *appsv1.Deployment {
	r := replicas
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &r,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	for _, fn := range opts {
		fn(deploy)
	}
	return deploy
}

func makeK8sService(name, ns string, svcType corev1.ServiceType, opts ...func(*corev1.Service)) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: corev1.ServiceSpec{
			Type: svcType,
		},
	}
	for _, fn := range opts {
		fn(svc)
	}
	return svc
}

func makeK8sIngress(name, ns string, opts ...func(*networkingv1.Ingress)) *networkingv1.Ingress {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: networkingv1.IngressSpec{},
	}
	for _, fn := range opts {
		fn(ing)
	}
	return ing
}

func makeK8sStatefulSet(name, ns string, replicas int32, opts ...func(*appsv1.StatefulSet)) *appsv1.StatefulSet {
	r := replicas
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &r,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
		Status: appsv1.StatefulSetStatus{},
	}
	for _, fn := range opts {
		fn(sts)
	}
	return sts
}

func makeK8sDaemonSet(name, ns string, opts ...func(*appsv1.DaemonSet)) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}
	for _, fn := range opts {
		fn(ds)
	}
	return ds
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

// =============================================================================
// TestFormatDuration
// =============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"10 days", 10 * 24 * time.Hour, "10d"},
		{"1 day", 24 * time.Hour, "1d"},
		{"5 hours", 5 * time.Hour, "5h"},
		{"1 hour", 1 * time.Hour, "1h"},
		{"30 minutes", 30 * time.Minute, "30m"},
		{"1 minute", 1 * time.Minute, "1m"},
		{"zero", 0, "0m"},
		{"seconds only", 45 * time.Second, "0m"},
		{"day + hours returns days", 25 * time.Hour, "1d"},
		{"hours + minutes returns hours", 61 * time.Minute, "1h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// =============================================================================
// TestConvertPod
// =============================================================================

func TestConvertPod_BasicFields(t *testing.T) {
	pod := makeK8sPod("web-abc", "default", "Running", func(p *corev1.Pod) {
		p.Spec.NodeName = "node-1"
		p.Spec.RestartPolicy = corev1.RestartPolicyAlways
		p.Spec.ServiceAccountName = "my-sa"
		p.Spec.DNSPolicy = corev1.DNSClusterFirst
		p.Spec.HostNetwork = true
		p.Labels = map[string]string{"app": "web"}
		p.Annotations = map[string]string{"note": "test"}
		p.OwnerReferences = []metav1.OwnerReference{
			{Kind: "ReplicaSet", Name: "web-abc-rs"},
		}
		p.Spec.Containers = []corev1.Container{
			{
				Name:  "app",
				Image: "nginx:1.25",
				Ports: []corev1.ContainerPort{
					{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP},
				},
			},
		}
		p.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:         "app",
				Ready:        true,
				RestartCount: 3,
				Image:        "nginx:1.25-runtime",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}
		p.Status.PodIP = "10.0.0.5"
		p.Status.PodIPs = []corev1.PodIP{{IP: "10.0.0.5"}}
		p.Status.HostIP = "192.168.1.1"
		p.Status.QOSClass = corev1.PodQOSBurstable
	})

	result := ConvertPod(pod)

	// Summary
	if result.Summary.Name != "web-abc" {
		t.Errorf("Name = %q, want %q", result.Summary.Name, "web-abc")
	}
	if result.Summary.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", result.Summary.Namespace, "default")
	}
	if result.Summary.NodeName != "node-1" {
		t.Errorf("NodeName = %q, want %q", result.Summary.NodeName, "node-1")
	}
	if result.Summary.OwnerKind != "ReplicaSet" {
		t.Errorf("OwnerKind = %q, want %q", result.Summary.OwnerKind, "ReplicaSet")
	}
	if result.Summary.OwnerName != "web-abc-rs" {
		t.Errorf("OwnerName = %q, want %q", result.Summary.OwnerName, "web-abc-rs")
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}

	// Status
	if result.Status.Phase != "Running" {
		t.Errorf("Phase = %q, want %q", result.Status.Phase, "Running")
	}
	if result.Status.Ready != "1/1" {
		t.Errorf("Ready = %q, want %q", result.Status.Ready, "1/1")
	}
	if result.Status.Restarts != 3 {
		t.Errorf("Restarts = %d, want %d", result.Status.Restarts, 3)
	}
	if result.Status.QoSClass != "Burstable" {
		t.Errorf("QoSClass = %q, want %q", result.Status.QoSClass, "Burstable")
	}
	if result.Status.PodIP != "10.0.0.5" {
		t.Errorf("PodIP = %q, want %q", result.Status.PodIP, "10.0.0.5")
	}
	if result.Status.HostIP != "192.168.1.1" {
		t.Errorf("HostIP = %q, want %q", result.Status.HostIP, "192.168.1.1")
	}
	if len(result.Status.PodIPs) != 1 || result.Status.PodIPs[0] != "10.0.0.5" {
		t.Errorf("PodIPs = %v, want [10.0.0.5]", result.Status.PodIPs)
	}

	// Spec
	if result.Spec.RestartPolicy != "Always" {
		t.Errorf("RestartPolicy = %q, want %q", result.Spec.RestartPolicy, "Always")
	}
	if result.Spec.ServiceAccountName != "my-sa" {
		t.Errorf("ServiceAccountName = %q, want %q", result.Spec.ServiceAccountName, "my-sa")
	}
	if !result.Spec.HostNetwork {
		t.Errorf("HostNetwork = false, want true")
	}

	// Labels & Annotations
	if result.Labels["app"] != "web" {
		t.Errorf("Labels[app] = %q, want %q", result.Labels["app"], "web")
	}
	if result.Annotations["note"] != "test" {
		t.Errorf("Annotations[note] = %q, want %q", result.Annotations["note"], "test")
	}
}

func TestConvertPod_ContainerMerge(t *testing.T) {
	pod := makeK8sPod("merge-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{
			{Name: "app", Image: "app:spec-v1"},
			{Name: "sidecar", Image: "sidecar:spec-v1"},
		}
		p.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:         "app",
				Ready:        true,
				RestartCount: 2,
				Image:        "app:runtime-v1",
				State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			},
			{
				Name:         "sidecar",
				Ready:        false,
				RestartCount: 0,
				Image:        "sidecar:runtime-v1",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "CrashLoopBackOff",
						Message: "container crashed",
					},
				},
			},
		}
	})

	result := ConvertPod(pod)

	if len(result.Containers) != 2 {
		t.Fatalf("Containers count = %d, want 2", len(result.Containers))
	}

	// Container "app" should have runtime image from status
	app := result.Containers[0]
	if app.Name != "app" {
		t.Errorf("Container[0].Name = %q, want %q", app.Name, "app")
	}
	if app.Image != "app:runtime-v1" {
		t.Errorf("Container[0].Image = %q, want %q (runtime image from status)", app.Image, "app:runtime-v1")
	}
	if !app.Ready {
		t.Errorf("Container[0].Ready = false, want true")
	}
	if app.RestartCount != 2 {
		t.Errorf("Container[0].RestartCount = %d, want 2", app.RestartCount)
	}
	if app.State != "running" {
		t.Errorf("Container[0].State = %q, want %q", app.State, "running")
	}

	// Container "sidecar" should have waiting state
	sidecar := result.Containers[1]
	if sidecar.Name != "sidecar" {
		t.Errorf("Container[1].Name = %q, want %q", sidecar.Name, "sidecar")
	}
	if sidecar.State != "waiting" {
		t.Errorf("Container[1].State = %q, want %q", sidecar.State, "waiting")
	}
	if sidecar.StateReason != "CrashLoopBackOff" {
		t.Errorf("Container[1].StateReason = %q, want %q", sidecar.StateReason, "CrashLoopBackOff")
	}

	// Ready string: 1 ready out of 2
	if result.Status.Ready != "1/2" {
		t.Errorf("Ready = %q, want %q", result.Status.Ready, "1/2")
	}
}

func TestConvertPod_SidecarReasonFallback(t *testing.T) {
	tests := []struct {
		name       string
		phase      corev1.PodPhase
		containers []corev1.ContainerStatus
		wantReason string
		wantMsg    string
	}{
		{
			name:  "prefer non-sidecar reason over sidecar",
			phase: corev1.PodPending,
			containers: []corev1.ContainerStatus{
				{
					Name: "linkerd-proxy",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "SidecarWaiting", Message: "sidecar msg"},
					},
				},
				{
					Name: "app",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "app msg"},
					},
				},
			},
			wantReason: "ImagePullBackOff",
			wantMsg:    "app msg",
		},
		{
			name:  "fallback to sidecar when no non-sidecar reason",
			phase: corev1.PodFailed,
			containers: []corev1.ContainerStatus{
				{
					Name: "linkerd-init",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "InitFailed", Message: "init msg"},
					},
				},
				{
					Name:  "app",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
			wantReason: "InitFailed",
			wantMsg:    "init msg",
		},
		{
			name:  "no reason from any container",
			phase: corev1.PodPending,
			containers: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
			wantReason: "",
			wantMsg:    "",
		},
		{
			name:  "Running phase does not extract container reason",
			phase: corev1.PodRunning,
			containers: []corev1.ContainerStatus{
				{
					Name: "app",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "SomeReason", Message: "some msg"},
					},
				},
			},
			wantReason: "",
			wantMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := makeK8sPod("test", "ns", string(tt.phase), func(p *corev1.Pod) {
				p.Spec.Containers = make([]corev1.Container, len(tt.containers))
				for i, cs := range tt.containers {
					p.Spec.Containers[i] = corev1.Container{Name: cs.Name, Image: "img"}
				}
				p.Status.ContainerStatuses = tt.containers
			})

			result := ConvertPod(pod)
			if result.Status.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Status.Reason, tt.wantReason)
			}
			if result.Status.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", result.Status.Message, tt.wantMsg)
			}
		})
	}
}

func TestConvertPod_Volumes(t *testing.T) {
	pod := makeK8sPod("vol-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{{Name: "app", Image: "img"}}
		p.Spec.Volumes = []corev1.Volume{
			{Name: "cm-vol", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "my-cm"}}}},
			{Name: "secret-vol", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "my-secret"}}},
			{Name: "empty", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			{Name: "pvc-vol", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "my-pvc"}}},
			{Name: "host", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data"}}},
			{Name: "proj", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{}}},
			{Name: "dapi", VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{}}},
			{Name: "other", VolumeSource: corev1.VolumeSource{}},
		}
	})

	result := ConvertPod(pod)

	expected := []struct {
		name, typ, source string
	}{
		{"cm-vol", "ConfigMap", "my-cm"},
		{"secret-vol", "Secret", "my-secret"},
		{"empty", "EmptyDir", ""},
		{"pvc-vol", "PVC", "my-pvc"},
		{"host", "HostPath", "/data"},
		{"proj", "Projected", ""},
		{"dapi", "DownwardAPI", ""},
		{"other", "Other", ""},
	}

	if len(result.Volumes) != len(expected) {
		t.Fatalf("Volumes count = %d, want %d", len(result.Volumes), len(expected))
	}
	for i, exp := range expected {
		vol := result.Volumes[i]
		if vol.Name != exp.name {
			t.Errorf("Volume[%d].Name = %q, want %q", i, vol.Name, exp.name)
		}
		if vol.Type != exp.typ {
			t.Errorf("Volume[%d].Type = %q, want %q", i, vol.Type, exp.typ)
		}
		if vol.Source != exp.source {
			t.Errorf("Volume[%d].Source = %q, want %q", i, vol.Source, exp.source)
		}
	}
}

func TestConvertPod_EmptyContainers(t *testing.T) {
	pod := makeK8sPod("empty", "ns", "Pending")
	result := ConvertPod(pod)

	if result.Status.Ready != "" {
		t.Errorf("Ready = %q, want empty string for no containers", result.Status.Ready)
	}
	if result.Status.Restarts != 0 {
		t.Errorf("Restarts = %d, want 0", result.Status.Restarts)
	}
	if len(result.Containers) != 0 {
		t.Errorf("Containers = %d, want 0", len(result.Containers))
	}
}

func TestConvertPod_InitContainers(t *testing.T) {
	pod := makeK8sPod("init-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.InitContainers = []corev1.Container{
			{Name: "init-db", Image: "init:v1"},
		}
		p.Spec.Containers = []corev1.Container{
			{Name: "app", Image: "app:v1"},
		}
		p.Status.InitContainerStatuses = []corev1.ContainerStatus{
			{
				Name:  "init-db",
				Ready: false,
				Image: "init:v1-runtime",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason: "Completed",
					},
				},
			},
		}
	})

	result := ConvertPod(pod)
	if len(result.InitContainers) != 1 {
		t.Fatalf("InitContainers count = %d, want 1", len(result.InitContainers))
	}
	ic := result.InitContainers[0]
	if ic.Name != "init-db" {
		t.Errorf("InitContainer.Name = %q, want %q", ic.Name, "init-db")
	}
	if ic.State != "terminated" {
		t.Errorf("InitContainer.State = %q, want %q", ic.State, "terminated")
	}
	if ic.StateReason != "Completed" {
		t.Errorf("InitContainer.StateReason = %q, want %q", ic.StateReason, "Completed")
	}
}

func TestConvertPod_Tolerations(t *testing.T) {
	tolSec := int64(300)
	pod := makeK8sPod("tol-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{{Name: "app", Image: "img"}}
		p.Spec.Tolerations = []corev1.Toleration{
			{
				Key:               "node.kubernetes.io/not-ready",
				Operator:          corev1.TolerationOpExists,
				Effect:            corev1.TaintEffectNoExecute,
				TolerationSeconds: &tolSec,
			},
		}
	})

	result := ConvertPod(pod)
	if len(result.Spec.Tolerations) != 1 {
		t.Fatalf("Tolerations count = %d, want 1", len(result.Spec.Tolerations))
	}
	tol := result.Spec.Tolerations[0]
	if tol.Key != "node.kubernetes.io/not-ready" {
		t.Errorf("Toleration.Key = %q", tol.Key)
	}
	if tol.TolerationSeconds == nil || *tol.TolerationSeconds != 300 {
		t.Errorf("Toleration.TolerationSeconds unexpected")
	}
}

func TestConvertPod_Affinity(t *testing.T) {
	pod := makeK8sPod("aff-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{{Name: "app", Image: "img"}}
		p.Spec.Affinity = &corev1.Affinity{
			NodeAffinity:    &corev1.NodeAffinity{},
			PodAntiAffinity: &corev1.PodAntiAffinity{},
		}
	})

	result := ConvertPod(pod)
	if result.Spec.Affinity == nil {
		t.Fatal("Affinity is nil, want non-nil")
	}
	if result.Spec.Affinity.NodeAffinity != "已配置" {
		t.Errorf("NodeAffinity = %q, want %q", result.Spec.Affinity.NodeAffinity, "已配置")
	}
	if result.Spec.Affinity.PodAntiAffinity != "已配置" {
		t.Errorf("PodAntiAffinity = %q, want %q", result.Spec.Affinity.PodAntiAffinity, "已配置")
	}
	if result.Spec.Affinity.PodAffinity != "" {
		t.Errorf("PodAffinity = %q, want empty (not configured)", result.Spec.Affinity.PodAffinity)
	}
}

func TestConvertPod_Probes(t *testing.T) {
	pod := makeK8sPod("probe-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{
			{
				Name:  "app",
				Image: "img",
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 10,
					PeriodSeconds:       30,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(3306),
						},
					},
				},
				StartupProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"cat", "/tmp/healthy"},
						},
					},
				},
			},
		}
	})

	result := ConvertPod(pod)
	c := result.Containers[0]

	if c.LivenessProbe == nil {
		t.Fatal("LivenessProbe is nil")
	}
	if c.LivenessProbe.Type != "httpGet" {
		t.Errorf("LivenessProbe.Type = %q, want %q", c.LivenessProbe.Type, "httpGet")
	}
	if c.LivenessProbe.Path != "/healthz" {
		t.Errorf("LivenessProbe.Path = %q, want %q", c.LivenessProbe.Path, "/healthz")
	}
	if c.LivenessProbe.Port != 8080 {
		t.Errorf("LivenessProbe.Port = %d, want %d", c.LivenessProbe.Port, 8080)
	}

	if c.ReadinessProbe == nil {
		t.Fatal("ReadinessProbe is nil")
	}
	if c.ReadinessProbe.Type != "tcpSocket" {
		t.Errorf("ReadinessProbe.Type = %q, want %q", c.ReadinessProbe.Type, "tcpSocket")
	}

	if c.StartupProbe == nil {
		t.Fatal("StartupProbe is nil")
	}
	if c.StartupProbe.Type != "exec" {
		t.Errorf("StartupProbe.Type = %q, want %q", c.StartupProbe.Type, "exec")
	}
	if c.StartupProbe.Command != "cat /tmp/healthy" {
		t.Errorf("StartupProbe.Command = %q, want %q", c.StartupProbe.Command, "cat /tmp/healthy")
	}
}

func TestConvertPod_EnvSources(t *testing.T) {
	pod := makeK8sPod("env-test", "ns", "Running", func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{
			{
				Name:  "app",
				Image: "img",
				Env: []corev1.EnvVar{
					{Name: "PLAIN", Value: "hello"},
					{Name: "FROM_CM", ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-cm"},
							Key:                  "key1",
						},
					}},
					{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
							Key:                  "key2",
						},
					}},
					{Name: "FROM_FIELD", ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
					}},
					{Name: "FROM_RESOURCE", ValueFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{Resource: "limits.cpu"},
					}},
				},
			},
		}
	})

	result := ConvertPod(pod)
	envs := result.Containers[0].Envs
	if len(envs) != 5 {
		t.Fatalf("Envs count = %d, want 5", len(envs))
	}
	if envs[0].Value != "hello" {
		t.Errorf("Env[0].Value = %q, want %q", envs[0].Value, "hello")
	}
	if envs[1].ValueFrom != "configmap:my-cm" {
		t.Errorf("Env[1].ValueFrom = %q, want %q", envs[1].ValueFrom, "configmap:my-cm")
	}
	if envs[2].ValueFrom != "secret:my-secret" {
		t.Errorf("Env[2].ValueFrom = %q, want %q", envs[2].ValueFrom, "secret:my-secret")
	}
	if envs[3].ValueFrom != "field:metadata.name" {
		t.Errorf("Env[3].ValueFrom = %q, want %q", envs[3].ValueFrom, "field:metadata.name")
	}
	if envs[4].ValueFrom != "resource:limits.cpu" {
		t.Errorf("Env[4].ValueFrom = %q, want %q", envs[4].ValueFrom, "resource:limits.cpu")
	}
}

// =============================================================================
// TestConvertNode
// =============================================================================

func TestConvertNode_RoleLabels(t *testing.T) {
	tests := []struct {
		name      string
		labels    map[string]string
		wantRoles []string
	}{
		{
			name: "master + control-plane",
			labels: map[string]string{
				"node-role.kubernetes.io/master":        "",
				"node-role.kubernetes.io/control-plane": "",
			},
			wantRoles: []string{"master", "control-plane"},
		},
		{
			name: "worker only",
			labels: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			wantRoles: []string{"worker"},
		},
		{
			name:      "no role labels",
			labels:    map[string]string{"hostname": "node-1"},
			wantRoles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := makeK8sNode("test-node", func(n *corev1.Node) {
				n.Labels = tt.labels
			})
			result := ConvertNode(node)

			// Roles may be in any order, so check set equality
			if len(result.Summary.Roles) != len(tt.wantRoles) {
				t.Errorf("Roles = %v, want %v", result.Summary.Roles, tt.wantRoles)
				return
			}
			roleSet := make(map[string]bool)
			for _, r := range result.Summary.Roles {
				roleSet[r] = true
			}
			for _, r := range tt.wantRoles {
				if !roleSet[r] {
					t.Errorf("Missing role %q in %v", r, result.Summary.Roles)
				}
			}
		})
	}
}

func TestConvertNode_IPv4Priority(t *testing.T) {
	node := makeK8sNode("dual-stack", func(n *corev1.Node) {
		n.Status.Addresses = []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "fd00::1"},
			{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
			{Type: corev1.NodeExternalIP, Address: "2001:db8::1"},
			{Type: corev1.NodeExternalIP, Address: "203.0.113.1"},
			{Type: corev1.NodeHostName, Address: "node-host"},
		}
	})

	result := ConvertNode(node)

	if result.Addresses.InternalIP != "10.0.0.1" {
		t.Errorf("InternalIP = %q, want %q (IPv4 preferred)", result.Addresses.InternalIP, "10.0.0.1")
	}
	if result.Addresses.ExternalIP != "203.0.113.1" {
		t.Errorf("ExternalIP = %q, want %q (IPv4 preferred)", result.Addresses.ExternalIP, "203.0.113.1")
	}
	if result.Addresses.Hostname != "node-host" {
		t.Errorf("Hostname = %q, want %q", result.Addresses.Hostname, "node-host")
	}
	if len(result.Addresses.All) != 5 {
		t.Errorf("All addresses count = %d, want 5", len(result.Addresses.All))
	}
}

func TestConvertNode_IPv4Only(t *testing.T) {
	node := makeK8sNode("ipv4-only", func(n *corev1.Node) {
		n.Status.Addresses = []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
		}
	})

	result := ConvertNode(node)
	if result.Addresses.InternalIP != "10.0.0.2" {
		t.Errorf("InternalIP = %q, want %q", result.Addresses.InternalIP, "10.0.0.2")
	}
}

func TestConvertNode_BasicFields(t *testing.T) {
	node := makeK8sNode("node-basic", func(n *corev1.Node) {
		n.Spec.PodCIDRs = []string{"10.244.0.0/24"}
		n.Spec.ProviderID = "aws:///us-east-1a/i-12345"
		n.Spec.Unschedulable = false
		n.Status.NodeInfo = corev1.NodeSystemInfo{
			OSImage:                 "Ubuntu 22.04",
			OperatingSystem:         "linux",
			Architecture:            "amd64",
			KernelVersion:           "5.15.0",
			ContainerRuntimeVersion: "containerd://1.6.0",
			KubeletVersion:          "v1.28.0",
			KubeProxyVersion:        "v1.28.0",
		}
		n.Status.Conditions = []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
		}
		n.Status.Capacity = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("8"),
			corev1.ResourceMemory: resource.MustParse("32Gi"),
			corev1.ResourcePods:   resource.MustParse("110"),
		}
		n.Status.Allocatable = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("7800m"),
			corev1.ResourceMemory: resource.MustParse("31Gi"),
			corev1.ResourcePods:   resource.MustParse("110"),
		}
	})

	result := ConvertNode(node)

	if result.Summary.Name != "node-basic" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Ready != "True" {
		t.Errorf("Ready = %q, want %q", result.Summary.Ready, "True")
	}
	if !result.Summary.Schedulable {
		t.Errorf("Schedulable = false, want true")
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
	if result.Info.OSImage != "Ubuntu 22.04" {
		t.Errorf("Info.OSImage = %q", result.Info.OSImage)
	}
	if result.Info.KubeletVersion != "v1.28.0" {
		t.Errorf("Info.KubeletVersion = %q", result.Info.KubeletVersion)
	}
	if result.Capacity.CPU != "8" {
		t.Errorf("Capacity.CPU = %q, want %q", result.Capacity.CPU, "8")
	}
	if result.Spec.ProviderID != "aws:///us-east-1a/i-12345" {
		t.Errorf("Spec.ProviderID = %q", result.Spec.ProviderID)
	}
}

func TestConvertNode_Taints(t *testing.T) {
	now := metav1.Now()
	node := makeK8sNode("tainted", func(n *corev1.Node) {
		n.Spec.Taints = []corev1.Taint{
			{
				Key:       "node-role.kubernetes.io/master",
				Effect:    corev1.TaintEffectNoSchedule,
				TimeAdded: &now,
			},
		}
	})

	result := ConvertNode(node)
	if len(result.Taints) != 1 {
		t.Fatalf("Taints count = %d, want 1", len(result.Taints))
	}
	if result.Taints[0].Key != "node-role.kubernetes.io/master" {
		t.Errorf("Taint.Key = %q", result.Taints[0].Key)
	}
	if result.Taints[0].Effect != "NoSchedule" {
		t.Errorf("Taint.Effect = %q", result.Taints[0].Effect)
	}
	if result.Taints[0].TimeAdded == nil {
		t.Errorf("Taint.TimeAdded is nil, want non-nil")
	}
}

func TestConvertNode_ReadyStatusUnknown(t *testing.T) {
	node := makeK8sNode("unknown-node")
	// No conditions at all
	result := ConvertNode(node)
	if result.Summary.Ready != "Unknown" {
		t.Errorf("Ready = %q, want %q", result.Summary.Ready, "Unknown")
	}
}

// =============================================================================
// TestConvertDeployment
// =============================================================================

func TestConvertDeployment_RolloutPhases(t *testing.T) {
	tests := []struct {
		name       string
		conditions []appsv1.DeploymentCondition
		paused     bool
		wantPhase  string
		wantBadges []string
	}{
		{
			name: "Available",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			},
			wantPhase:  "Available",
			wantBadges: []string{"Available"},
		},
		{
			name: "Progressing only",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Message: "updating"},
			},
			wantPhase:  "Progressing",
			wantBadges: []string{"Progressing"},
		},
		{
			name: "Available + Progressing keeps Available phase",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Message: "in progress"},
			},
			wantPhase:  "Available",
			wantBadges: []string{"Available", "Progressing"},
		},
		{
			name: "ProgressDeadlineExceeded",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded", Message: "deadline"},
			},
			wantPhase:  "Failed",
			wantBadges: []string{"Failed"},
		},
		{
			name: "ReplicaFailure",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue, Message: "quota exceeded"},
			},
			wantPhase:  "Failed",
			wantBadges: []string{"ReplicaFailure"},
		},
		{
			name:       "No conditions",
			conditions: nil,
			wantPhase:  "Unknown",
			wantBadges: nil,
		},
		{
			name: "Paused adds badge",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			},
			paused:     true,
			wantPhase:  "Available",
			wantBadges: []string{"Available", "Paused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploy := makeK8sDeployment("test", "ns", 3, func(d *appsv1.Deployment) {
				d.Status.Conditions = tt.conditions
				d.Spec.Paused = tt.paused
			})
			result := ConvertDeployment(deploy)

			if result.Rollout == nil {
				t.Fatal("Rollout is nil")
			}
			if result.Rollout.Phase != tt.wantPhase {
				t.Errorf("Phase = %q, want %q", result.Rollout.Phase, tt.wantPhase)
			}
			if len(result.Rollout.Badges) != len(tt.wantBadges) {
				t.Errorf("Badges = %v, want %v", result.Rollout.Badges, tt.wantBadges)
			} else {
				for i, b := range tt.wantBadges {
					if result.Rollout.Badges[i] != b {
						t.Errorf("Badge[%d] = %q, want %q", i, result.Rollout.Badges[i], b)
					}
				}
			}
		})
	}
}

func TestConvertDeployment_BasicFields(t *testing.T) {
	deploy := makeK8sDeployment("web", "prod", 3, func(d *appsv1.Deployment) {
		d.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
		d.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
			MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
		}
		d.Status.ReadyReplicas = 3
		d.Status.AvailableReplicas = 3
		d.Status.UpdatedReplicas = 3
		d.Status.Replicas = 3
		d.Labels = map[string]string{"app": "web"}
		d.Annotations = map[string]string{"note": "test"}
		d.Spec.Template.Labels = map[string]string{"pod-label": "val"}
	})

	result := ConvertDeployment(deploy)

	if result.Summary.Name != "web" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Namespace != "prod" {
		t.Errorf("Namespace = %q", result.Summary.Namespace)
	}
	if result.Summary.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", result.Summary.Replicas)
	}
	if result.Summary.Strategy != "RollingUpdate" {
		t.Errorf("Strategy = %q, want %q", result.Summary.Strategy, "RollingUpdate")
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
	if result.Labels["app"] != "web" {
		t.Errorf("Labels = %v", result.Labels)
	}

	// Strategy spec
	if result.Spec.Strategy == nil {
		t.Fatal("Spec.Strategy is nil")
	}
	if result.Spec.Strategy.RollingUpdate == nil {
		t.Fatal("Spec.Strategy.RollingUpdate is nil")
	}
	if result.Spec.Strategy.RollingUpdate.MaxUnavailable != "25%" {
		t.Errorf("MaxUnavailable = %q, want %q", result.Spec.Strategy.RollingUpdate.MaxUnavailable, "25%")
	}

	// Template labels
	if result.Template.Labels["pod-label"] != "val" {
		t.Errorf("Template.Labels = %v", result.Template.Labels)
	}
}

func TestConvertDeployment_NilReplicas(t *testing.T) {
	deploy := makeK8sDeployment("norep", "ns", 0, func(d *appsv1.Deployment) {
		d.Spec.Replicas = nil
	})

	result := ConvertDeployment(deploy)
	if result.Summary.Replicas != 0 {
		t.Errorf("Replicas = %d, want 0 (nil defaults to 0)", result.Summary.Replicas)
	}
}

// =============================================================================
// TestConvertService
// =============================================================================

func TestConvertService_Badges(t *testing.T) {
	tests := []struct {
		name       string
		svcType    corev1.ServiceType
		clusterIP  string
		extName    string
		wantBadges []string
	}{
		{
			name:       "LoadBalancer has LB badge",
			svcType:    corev1.ServiceTypeLoadBalancer,
			clusterIP:  "10.96.0.1",
			wantBadges: []string{"LB"},
		},
		{
			name:       "NodePort has NodePort badge",
			svcType:    corev1.ServiceTypeNodePort,
			clusterIP:  "10.96.0.2",
			wantBadges: []string{"NodePort"},
		},
		{
			name:       "Headless service has Headless badge",
			svcType:    corev1.ServiceTypeClusterIP,
			clusterIP:  "None",
			wantBadges: []string{"Headless"},
		},
		{
			name:       "ClusterIP with no special features",
			svcType:    corev1.ServiceTypeClusterIP,
			clusterIP:  "10.96.0.3",
			wantBadges: []string{},
		},
		{
			name:       "ExternalName badge added",
			svcType:    corev1.ServiceTypeExternalName,
			extName:    "ext.example.com",
			wantBadges: []string{"ExternalName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := makeK8sService("test-svc", "ns", tt.svcType, func(s *corev1.Service) {
				s.Spec.ClusterIP = tt.clusterIP
				s.Spec.ExternalName = tt.extName
			})

			result := ConvertService(svc)

			if len(result.Summary.Badges) != len(tt.wantBadges) {
				t.Errorf("Badges = %v, want %v", result.Summary.Badges, tt.wantBadges)
				return
			}
			for i, b := range tt.wantBadges {
				if result.Summary.Badges[i] != b {
					t.Errorf("Badge[%d] = %q, want %q", i, result.Summary.Badges[i], b)
				}
			}
		})
	}
}

func TestConvertService_BasicFields(t *testing.T) {
	svc := makeK8sService("my-svc", "prod", corev1.ServiceTypeClusterIP, func(s *corev1.Service) {
		s.Spec.ClusterIP = "10.96.0.1"
		s.Spec.Selector = map[string]string{"app": "web"}
		s.Spec.Ports = []corev1.ServicePort{
			{Name: "http", Protocol: corev1.ProtocolTCP, Port: 80, TargetPort: intstr.FromInt(8080)},
			{Name: "https", Protocol: corev1.ProtocolTCP, Port: 443, TargetPort: intstr.FromInt(8443), NodePort: 30443},
		}
		s.Labels = map[string]string{"tier": "frontend"}
	})

	result := ConvertService(svc)

	if result.Summary.Name != "my-svc" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Type != "ClusterIP" {
		t.Errorf("Type = %q, want %q", result.Summary.Type, "ClusterIP")
	}
	if result.Summary.PortsCount != 2 {
		t.Errorf("PortsCount = %d, want 2", result.Summary.PortsCount)
	}
	if !result.Summary.HasSelector {
		t.Errorf("HasSelector = false, want true")
	}
	if result.Summary.ClusterIP != "10.96.0.1" {
		t.Errorf("ClusterIP = %q", result.Summary.ClusterIP)
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}

	// Ports
	if len(result.Ports) != 2 {
		t.Fatalf("Ports count = %d, want 2", len(result.Ports))
	}
	if result.Ports[0].Name != "http" {
		t.Errorf("Port[0].Name = %q", result.Ports[0].Name)
	}
	if result.Ports[0].Port != 80 {
		t.Errorf("Port[0].Port = %d", result.Ports[0].Port)
	}
	if result.Ports[0].TargetPort != "8080" {
		t.Errorf("Port[0].TargetPort = %q", result.Ports[0].TargetPort)
	}
	if result.Ports[1].NodePort != 30443 {
		t.Errorf("Port[1].NodePort = %d, want 30443", result.Ports[1].NodePort)
	}

	// Selector
	if result.Selector["app"] != "web" {
		t.Errorf("Selector = %v", result.Selector)
	}
}

func TestConvertService_EmptyType(t *testing.T) {
	svc := makeK8sService("empty-type", "ns", "")
	result := ConvertService(svc)
	if result.Summary.Type != "ClusterIP" {
		t.Errorf("Type = %q, want %q (default)", result.Summary.Type, "ClusterIP")
	}
}

func TestConvertService_LoadBalancerIngress(t *testing.T) {
	svc := makeK8sService("lb-svc", "ns", corev1.ServiceTypeLoadBalancer, func(s *corev1.Service) {
		s.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
			{IP: "1.2.3.4"},
			{Hostname: "lb.example.com"},
		}
	})

	result := ConvertService(svc)
	if len(result.Network.LoadBalancerIngress) != 2 {
		t.Fatalf("LB Ingress count = %d, want 2", len(result.Network.LoadBalancerIngress))
	}
	if result.Network.LoadBalancerIngress[0] != "1.2.3.4" {
		t.Errorf("LB Ingress[0] = %q", result.Network.LoadBalancerIngress[0])
	}
	if result.Network.LoadBalancerIngress[1] != "lb.example.com" {
		t.Errorf("LB Ingress[1] = %q", result.Network.LoadBalancerIngress[1])
	}
}

// =============================================================================
// TestConvertEvent
// =============================================================================

func TestConvertEvent_BasicFields(t *testing.T) {
	now := time.Now()
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pod-event-abc",
			Namespace:         "default",
			UID:               types.UID("event-uid-123"),
			CreationTimestamp: metav1.Time{Time: now},
		},
		Type:    "Warning",
		Reason:  "BackOff",
		Message: "Back-off restarting failed container",
		Count:   5,
		Source:  corev1.EventSource{Component: "kubelet"},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      "web-abc",
			Namespace: "default",
			UID:       types.UID("pod-uid-456"),
		},
		FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
		LastTimestamp:  metav1.Time{Time: now},
	}

	result := ConvertEvent(event)

	if result.Name != "pod-event-abc" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace = %q", result.Namespace)
	}
	if result.Kind != "Event" {
		t.Errorf("Kind = %q, want %q", result.Kind, "Event")
	}
	if result.Type != "Warning" {
		t.Errorf("Type = %q", result.Type)
	}
	if result.Reason != "BackOff" {
		t.Errorf("Reason = %q", result.Reason)
	}
	if result.Message != "Back-off restarting failed container" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want 5", result.Count)
	}
	if result.Source != "kubelet" {
		t.Errorf("Source = %q", result.Source)
	}
	if result.InvolvedObject.Kind != "Pod" {
		t.Errorf("InvolvedObject.Kind = %q", result.InvolvedObject.Kind)
	}
	if result.InvolvedObject.Name != "web-abc" {
		t.Errorf("InvolvedObject.Name = %q", result.InvolvedObject.Name)
	}
	if result.FirstTimestamp.IsZero() {
		t.Errorf("FirstTimestamp is zero")
	}
	if result.LastTimestamp.IsZero() {
		t.Errorf("LastTimestamp is zero")
	}
}

func TestConvertEvent_ZeroTimestamps(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-ts",
			Namespace: "ns",
		},
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "p"},
	}

	result := ConvertEvent(event)
	if !result.FirstTimestamp.IsZero() {
		t.Errorf("FirstTimestamp should be zero when not set")
	}
	if !result.LastTimestamp.IsZero() {
		t.Errorf("LastTimestamp should be zero when not set")
	}
}

// =============================================================================
// TestConvertNamespace
// =============================================================================

func TestConvertNamespace_BasicFields(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "production",
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"owner": "team-a"},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	result := ConvertNamespace(ns)

	if result.Summary.Name != "production" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
	if result.Status.Phase != "Active" {
		t.Errorf("Phase = %q, want %q", result.Status.Phase, "Active")
	}
	if result.Labels["env"] != "prod" {
		t.Errorf("Labels = %v", result.Labels)
	}
	if result.Annotations["owner"] != "team-a" {
		t.Errorf("Annotations = %v", result.Annotations)
	}
	// CreatedAt should be RFC3339 formatted
	if result.Summary.CreatedAt == "" {
		t.Errorf("CreatedAt is empty")
	}
}

// =============================================================================
// TestConvertConfigMap
// =============================================================================

func TestConvertConfigMap_BasicFields(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app-config",
			Namespace:         "default",
			UID:               types.UID("cm-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"app": "web"},
		},
		Data: map[string]string{
			"config.yaml":  "content-1",
			"settings.env": "content-2",
		},
	}

	result := ConvertConfigMap(cm)

	if result.Name != "app-config" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace = %q", result.Namespace)
	}
	if result.Kind != "ConfigMap" {
		t.Errorf("Kind = %q, want %q", result.Kind, "ConfigMap")
	}
	if len(result.DataKeys) != 2 {
		t.Errorf("DataKeys count = %d, want 2", len(result.DataKeys))
	}
	// DataKeys should contain only keys, not values
	keySet := make(map[string]bool)
	for _, k := range result.DataKeys {
		keySet[k] = true
	}
	if !keySet["config.yaml"] || !keySet["settings.env"] {
		t.Errorf("DataKeys = %v, want [config.yaml, settings.env]", result.DataKeys)
	}
}

func TestConvertConfigMap_EmptyData(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "empty-cm",
			Namespace: "ns",
		},
	}

	result := ConvertConfigMap(cm)
	if len(result.DataKeys) != 0 {
		t.Errorf("DataKeys count = %d, want 0", len(result.DataKeys))
	}
}

// =============================================================================
// TestConvertSecret
// =============================================================================

func TestConvertSecret_BasicFields(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "tls-cert",
			Namespace:         "default",
			UID:               types.UID("secret-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("cert-data"),
			"tls.key": []byte("key-data"),
		},
	}

	result := ConvertSecret(secret)

	if result.Name != "tls-cert" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Kind != "Secret" {
		t.Errorf("Kind = %q, want %q", result.Kind, "Secret")
	}
	if result.Type != "kubernetes.io/tls" {
		t.Errorf("Type = %q, want %q", result.Type, "kubernetes.io/tls")
	}
	if len(result.DataKeys) != 2 {
		t.Errorf("DataKeys count = %d, want 2", len(result.DataKeys))
	}
	// Keys only, no values
	keySet := make(map[string]bool)
	for _, k := range result.DataKeys {
		keySet[k] = true
	}
	if !keySet["tls.crt"] || !keySet["tls.key"] {
		t.Errorf("DataKeys = %v", result.DataKeys)
	}
}

func TestConvertSecret_EmptyData(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "empty-secret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeOpaque,
	}

	result := ConvertSecret(secret)
	if len(result.DataKeys) != 0 {
		t.Errorf("DataKeys count = %d, want 0", len(result.DataKeys))
	}
	if result.Type != "Opaque" {
		t.Errorf("Type = %q, want %q", result.Type, "Opaque")
	}
}

// =============================================================================
// TestConvertIngress
// =============================================================================

func TestConvertIngress_BasicFields(t *testing.T) {
	pathTypePrefix := networkingv1.PathTypePrefix
	ingressClass := "nginx"

	ing := makeK8sIngress("web-ing", "prod", func(i *networkingv1.Ingress) {
		i.Spec.IngressClassName = &ingressClass
		i.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/api",
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "api-svc",
										Port: networkingv1.ServiceBackendPort{Number: 8080},
									},
								},
							},
							{
								Path:     "/web",
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "web-svc",
										Port: networkingv1.ServiceBackendPort{Name: "http"},
									},
								},
							},
						},
					},
				},
			},
			{
				Host: "api.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "api-svc",
										Port: networkingv1.ServiceBackendPort{Number: 8080},
									},
								},
							},
						},
					},
				},
			},
		}
		i.Spec.TLS = []networkingv1.IngressTLS{
			{Hosts: []string{"example.com"}, SecretName: "tls-secret"},
		}
		i.Labels = map[string]string{"tier": "frontend"}
	})

	result := ConvertIngress(ing)

	if result.Summary.Name != "web-ing" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Namespace != "prod" {
		t.Errorf("Namespace = %q", result.Summary.Namespace)
	}
	if result.Summary.IngressClass != "nginx" {
		t.Errorf("IngressClass = %q, want %q", result.Summary.IngressClass, "nginx")
	}
	if result.Summary.HostsCount != 2 {
		t.Errorf("HostsCount = %d, want 2", result.Summary.HostsCount)
	}
	if result.Summary.PathsCount != 3 {
		t.Errorf("PathsCount = %d, want 3", result.Summary.PathsCount)
	}
	if !result.Summary.TLSEnabled {
		t.Errorf("TLSEnabled = false, want true")
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
	if len(result.Summary.Hosts) != 2 {
		t.Errorf("Hosts = %v, want 2 entries", result.Summary.Hosts)
	}

	// Spec rules
	if len(result.Spec.Rules) != 2 {
		t.Fatalf("Rules count = %d, want 2", len(result.Spec.Rules))
	}
	if len(result.Spec.Rules[0].Paths) != 2 {
		t.Errorf("Rule[0].Paths = %d, want 2", len(result.Spec.Rules[0].Paths))
	}
	// Backend
	path0 := result.Spec.Rules[0].Paths[0]
	if path0.Backend == nil || path0.Backend.Service == nil {
		t.Fatal("Backend or Backend.Service is nil")
	}
	if path0.Backend.Service.Name != "api-svc" {
		t.Errorf("Backend.Service.Name = %q", path0.Backend.Service.Name)
	}
	if path0.Backend.Service.PortNumber != 8080 {
		t.Errorf("Backend.Service.PortNumber = %d", path0.Backend.Service.PortNumber)
	}

	// TLS
	if len(result.Spec.TLS) != 1 {
		t.Fatalf("TLS count = %d, want 1", len(result.Spec.TLS))
	}
	if result.Spec.TLS[0].SecretName != "tls-secret" {
		t.Errorf("TLS.SecretName = %q", result.Spec.TLS[0].SecretName)
	}
}

func TestConvertIngress_ClassFromAnnotation(t *testing.T) {
	ing := makeK8sIngress("legacy-ing", "ns", func(i *networkingv1.Ingress) {
		i.Annotations = map[string]string{
			"kubernetes.io/ingress.class": "traefik",
		}
	})

	result := ConvertIngress(ing)
	if result.Summary.IngressClass != "traefik" {
		t.Errorf("IngressClass = %q, want %q (from annotation)", result.Summary.IngressClass, "traefik")
	}
}

func TestConvertIngress_NoTLS(t *testing.T) {
	ing := makeK8sIngress("no-tls", "ns")
	result := ConvertIngress(ing)
	if result.Summary.TLSEnabled {
		t.Errorf("TLSEnabled = true, want false")
	}
}

func TestConvertIngress_LoadBalancerStatus(t *testing.T) {
	ing := makeK8sIngress("lb-ing", "ns", func(i *networkingv1.Ingress) {
		i.Status.LoadBalancer.Ingress = []networkingv1.IngressLoadBalancerIngress{
			{IP: "1.2.3.4"},
			{Hostname: "lb.example.com"},
		}
	})

	result := ConvertIngress(ing)
	if len(result.Status.LoadBalancer) != 2 {
		t.Fatalf("LoadBalancer count = %d, want 2", len(result.Status.LoadBalancer))
	}
	if result.Status.LoadBalancer[0] != "1.2.3.4" {
		t.Errorf("LB[0] = %q", result.Status.LoadBalancer[0])
	}
	if result.Status.LoadBalancer[1] != "lb.example.com" {
		t.Errorf("LB[1] = %q", result.Status.LoadBalancer[1])
	}
}

// =============================================================================
// TestConvertStatefulSet
// =============================================================================

func TestConvertStatefulSet_BasicFields(t *testing.T) {
	sts := makeK8sStatefulSet("redis", "cache", 3, func(s *appsv1.StatefulSet) {
		s.Spec.ServiceName = "redis-headless"
		s.Status.ReadyReplicas = 3
		s.Status.CurrentReplicas = 3
		s.Status.UpdatedReplicas = 3
		s.Status.AvailableReplicas = 3
		s.Status.CurrentRevision = "redis-abc"
		s.Status.UpdateRevision = "redis-abc"
		s.Labels = map[string]string{"app": "redis"}
	})

	result := ConvertStatefulSet(sts)

	if result.Summary.Name != "redis" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Namespace != "cache" {
		t.Errorf("Namespace = %q", result.Summary.Namespace)
	}
	if result.Summary.Replicas != 3 {
		t.Errorf("Replicas = %d", result.Summary.Replicas)
	}
	if result.Summary.ServiceName != "redis-headless" {
		t.Errorf("ServiceName = %q", result.Summary.ServiceName)
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
}

func TestConvertStatefulSet_Rollout(t *testing.T) {
	tests := []struct {
		name            string
		replicas        int32
		readyReplicas   int32
		updatedReplicas int32
		currentRevision string
		updateRevision  string
		wantPhase       string
		wantBadgeCount  int
	}{
		{
			name:            "Complete - all ready and updated",
			replicas:        3,
			readyReplicas:   3,
			updatedReplicas: 3,
			currentRevision: "rev-1",
			updateRevision:  "rev-1",
			wantPhase:       "Complete",
			wantBadgeCount:  0,
		},
		{
			name:            "Progressing - updated < replicas",
			replicas:        3,
			readyReplicas:   3,
			updatedReplicas: 1,
			currentRevision: "rev-1",
			updateRevision:  "rev-2",
			wantPhase:       "Progressing",
			wantBadgeCount:  2, // "Updating" + "NewRevision"
		},
		{
			name:            "Progressing - ready < replicas",
			replicas:        3,
			readyReplicas:   1,
			updatedReplicas: 3,
			currentRevision: "rev-1",
			updateRevision:  "rev-1",
			wantPhase:       "Progressing",
			wantBadgeCount:  1, // "Scaling"
		},
		{
			name:            "NewRevision badge when revisions differ",
			replicas:        3,
			readyReplicas:   3,
			updatedReplicas: 3,
			currentRevision: "rev-1",
			updateRevision:  "rev-2",
			wantPhase:       "Complete",
			wantBadgeCount:  1, // "NewRevision"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := makeK8sStatefulSet("test", "ns", tt.replicas, func(s *appsv1.StatefulSet) {
				s.Status.ReadyReplicas = tt.readyReplicas
				s.Status.UpdatedReplicas = tt.updatedReplicas
				s.Status.CurrentRevision = tt.currentRevision
				s.Status.UpdateRevision = tt.updateRevision
			})

			result := ConvertStatefulSet(sts)

			if result.Rollout == nil {
				t.Fatal("Rollout is nil")
			}
			if result.Rollout.Phase != tt.wantPhase {
				t.Errorf("Phase = %q, want %q", result.Rollout.Phase, tt.wantPhase)
			}
			if len(result.Rollout.Badges) != tt.wantBadgeCount {
				t.Errorf("Badges = %v (count %d), want count %d", result.Rollout.Badges, len(result.Rollout.Badges), tt.wantBadgeCount)
			}
		})
	}
}

func TestConvertStatefulSet_VolumeClaimTemplates(t *testing.T) {
	sts := makeK8sStatefulSet("db", "ns", 1, func(s *appsv1.StatefulSet) {
		storageClass := "fast-ssd"
		s.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &storageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
		}
	})

	result := ConvertStatefulSet(sts)
	if len(result.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("VCT count = %d, want 1", len(result.Spec.VolumeClaimTemplates))
	}
	vct := result.Spec.VolumeClaimTemplates[0]
	if vct.Name != "data" {
		t.Errorf("VCT.Name = %q", vct.Name)
	}
	if vct.StorageClass != "fast-ssd" {
		t.Errorf("VCT.StorageClass = %q", vct.StorageClass)
	}
	if vct.Storage != "10Gi" {
		t.Errorf("VCT.Storage = %q", vct.Storage)
	}
	if len(vct.AccessModes) != 1 || vct.AccessModes[0] != "ReadWriteOnce" {
		t.Errorf("VCT.AccessModes = %v", vct.AccessModes)
	}
}

// =============================================================================
// TestConvertDaemonSet
// =============================================================================

func TestConvertDaemonSet_BasicFields(t *testing.T) {
	ds := makeK8sDaemonSet("fluentd", "kube-system", func(d *appsv1.DaemonSet) {
		d.Status.DesiredNumberScheduled = 5
		d.Status.CurrentNumberScheduled = 5
		d.Status.NumberReady = 5
		d.Status.NumberAvailable = 5
		d.Status.UpdatedNumberScheduled = 5
		d.Labels = map[string]string{"app": "fluentd"}
	})

	result := ConvertDaemonSet(ds)

	if result.Summary.Name != "fluentd" {
		t.Errorf("Name = %q", result.Summary.Name)
	}
	if result.Summary.Namespace != "kube-system" {
		t.Errorf("Namespace = %q", result.Summary.Namespace)
	}
	if result.Summary.DesiredNumberScheduled != 5 {
		t.Errorf("Desired = %d, want 5", result.Summary.DesiredNumberScheduled)
	}
	if result.Summary.NumberReady != 5 {
		t.Errorf("Ready = %d, want 5", result.Summary.NumberReady)
	}
	if result.Summary.Age != "10d" {
		t.Errorf("Age = %q, want %q", result.Summary.Age, "10d")
	}
}

func TestConvertDaemonSet_Rollout(t *testing.T) {
	tests := []struct {
		name             string
		desired          int32
		updated          int32
		ready            int32
		misscheduled     int32
		unavailable      int32
		wantPhase        string
		wantBadgeContain []string
	}{
		{
			name:             "Complete",
			desired:          3,
			updated:          3,
			ready:            3,
			wantPhase:        "Complete",
			wantBadgeContain: []string{},
		},
		{
			name:             "Progressing - updating",
			desired:          3,
			updated:          1,
			ready:            3,
			wantPhase:        "Progressing",
			wantBadgeContain: []string{"Updating"},
		},
		{
			name:             "Progressing - not ready",
			desired:          3,
			updated:          3,
			ready:            1,
			wantPhase:        "Progressing",
			wantBadgeContain: []string{"Scaling"},
		},
		{
			name:             "Misscheduled badge",
			desired:          3,
			updated:          3,
			ready:            3,
			misscheduled:     1,
			wantPhase:        "Complete",
			wantBadgeContain: []string{"Misscheduled"},
		},
		{
			name:             "Unavailable badge",
			desired:          3,
			updated:          3,
			ready:            3,
			unavailable:      1,
			wantPhase:        "Complete",
			wantBadgeContain: []string{"Unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := makeK8sDaemonSet("test", "ns", func(d *appsv1.DaemonSet) {
				d.Status.DesiredNumberScheduled = tt.desired
				d.Status.UpdatedNumberScheduled = tt.updated
				d.Status.NumberReady = tt.ready
				d.Status.NumberMisscheduled = tt.misscheduled
				d.Status.NumberUnavailable = tt.unavailable
			})

			result := ConvertDaemonSet(ds)

			if result.Rollout == nil {
				t.Fatal("Rollout is nil")
			}
			if result.Rollout.Phase != tt.wantPhase {
				t.Errorf("Phase = %q, want %q", result.Rollout.Phase, tt.wantPhase)
			}
			for _, wantBadge := range tt.wantBadgeContain {
				found := false
				for _, b := range result.Rollout.Badges {
					if b == wantBadge {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Badges %v does not contain %q", result.Rollout.Badges, wantBadge)
				}
			}
		})
	}
}

// =============================================================================
// TestConvertReplicaSet
// =============================================================================

func TestConvertReplicaSet_BasicFields(t *testing.T) {
	replicas := int32(3)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-abc-rs",
			Namespace:         "default",
			UID:               types.UID("rs-uid-123"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"app": "web"},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Deployment", Name: "web-abc"},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "web", "pod-template-hash": "abc"},
			},
		},
		Status: appsv1.ReplicaSetStatus{
			ReadyReplicas:     3,
			AvailableReplicas: 3,
		},
	}

	result := ConvertReplicaSet(rs)

	if result.Name != "web-abc-rs" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace = %q", result.Namespace)
	}
	if result.Kind != "ReplicaSet" {
		t.Errorf("Kind = %q", result.Kind)
	}
	if result.Replicas != 3 {
		t.Errorf("Replicas = %d", result.Replicas)
	}
	if result.ReadyReplicas != 3 {
		t.Errorf("ReadyReplicas = %d", result.ReadyReplicas)
	}
	if result.AvailableReplicas != 3 {
		t.Errorf("AvailableReplicas = %d", result.AvailableReplicas)
	}
	if result.OwnerKind != "Deployment" {
		t.Errorf("OwnerKind = %q", result.OwnerKind)
	}
	if result.OwnerName != "web-abc" {
		t.Errorf("OwnerName = %q", result.OwnerName)
	}
	if result.Selector["app"] != "web" {
		t.Errorf("Selector = %v", result.Selector)
	}
}

func TestConvertReplicaSet_NoOwner(t *testing.T) {
	replicas := int32(1)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-rs",
			Namespace: "ns",
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
	}

	result := ConvertReplicaSet(rs)
	if result.OwnerKind != "" {
		t.Errorf("OwnerKind = %q, want empty", result.OwnerKind)
	}
	if result.OwnerName != "" {
		t.Errorf("OwnerName = %q, want empty", result.OwnerName)
	}
}

// =============================================================================
// TestConvertJob
// =============================================================================

func TestConvertJob_BasicFields(t *testing.T) {
	now := time.Now()
	completionTime := metav1.Time{Time: now.Add(-5 * time.Minute)}
	startTime := metav1.Time{Time: now.Add(-10 * time.Minute)}
	completions := int32(1)
	parallelism := int32(1)
	backoffLimit := int32(6)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "data-migrate",
			Namespace:         "default",
			UID:               types.UID("job-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"job": "migrate"},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "scheduled-migrate"},
			},
		},
		Spec: batchv1.JobSpec{
			Completions:  &completions,
			Parallelism:  &parallelism,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "worker", Image: "migrate:v1"}},
				},
			},
		},
		Status: batchv1.JobStatus{
			Active:         0,
			Succeeded:      1,
			Failed:         0,
			StartTime:      &startTime,
			CompletionTime: &completionTime,
		},
	}

	result := ConvertJob(job)

	if result.Name != "data-migrate" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Kind != "Job" {
		t.Errorf("Kind = %q", result.Kind)
	}
	if result.Active != 0 {
		t.Errorf("Active = %d", result.Active)
	}
	if result.Succeeded != 1 {
		t.Errorf("Succeeded = %d", result.Succeeded)
	}
	if !result.Complete {
		t.Errorf("Complete = false, want true")
	}
	if result.Completions == nil || *result.Completions != 1 {
		t.Errorf("Completions unexpected")
	}
	if result.Parallelism == nil || *result.Parallelism != 1 {
		t.Errorf("Parallelism unexpected")
	}
	if result.BackoffLimit == nil || *result.BackoffLimit != 6 {
		t.Errorf("BackoffLimit unexpected")
	}
	if result.StartTime == nil {
		t.Errorf("StartTime is nil")
	}
	if result.FinishTime == nil {
		t.Errorf("FinishTime is nil")
	}
	if result.OwnerKind != "CronJob" {
		t.Errorf("OwnerKind = %q", result.OwnerKind)
	}
	if result.OwnerName != "scheduled-migrate" {
		t.Errorf("OwnerName = %q", result.OwnerName)
	}
}

func TestConvertJob_NotComplete(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-job",
			Namespace: "ns",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
		Status: batchv1.JobStatus{
			Active: 1,
		},
	}

	result := ConvertJob(job)
	if result.Complete {
		t.Errorf("Complete = true, want false")
	}
	if result.Active != 1 {
		t.Errorf("Active = %d, want 1", result.Active)
	}
	if result.StartTime != nil {
		t.Errorf("StartTime should be nil")
	}
}

// =============================================================================
// TestConvertCronJob
// =============================================================================

func TestConvertCronJob_BasicFields(t *testing.T) {
	lastSchedule := metav1.Time{Time: time.Now().Add(-1 * time.Hour)}
	lastSuccess := metav1.Time{Time: time.Now().Add(-2 * time.Hour)}
	successLimit := int32(3)
	failedLimit := int32(1)

	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cleanup",
			Namespace:         "ops",
			UID:               types.UID("cj-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"job": "cleanup"},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   "0 * * * *",
			Suspend:                    boolPtr(false),
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &successLimit,
			FailedJobsHistoryLimit:     &failedLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "cleaner", Image: "cleaner:v1"}},
						},
					},
				},
			},
		},
		Status: batchv1.CronJobStatus{
			Active:             []corev1.ObjectReference{{Name: "cleanup-123"}},
			LastScheduleTime:   &lastSchedule,
			LastSuccessfulTime: &lastSuccess,
		},
	}

	result := ConvertCronJob(cj)

	if result.Name != "cleanup" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Kind != "CronJob" {
		t.Errorf("Kind = %q", result.Kind)
	}
	if result.Schedule != "0 * * * *" {
		t.Errorf("Schedule = %q", result.Schedule)
	}
	if result.Suspend {
		t.Errorf("Suspend = true, want false")
	}
	if result.ConcurrencyPolicy != "Forbid" {
		t.Errorf("ConcurrencyPolicy = %q", result.ConcurrencyPolicy)
	}
	if result.ActiveJobs != 1 {
		t.Errorf("ActiveJobs = %d, want 1", result.ActiveJobs)
	}
	if result.SuccessfulJobsHistoryLimit == nil || *result.SuccessfulJobsHistoryLimit != 3 {
		t.Errorf("SuccessfulJobsHistoryLimit unexpected")
	}
	if result.FailedJobsHistoryLimit == nil || *result.FailedJobsHistoryLimit != 1 {
		t.Errorf("FailedJobsHistoryLimit unexpected")
	}
	if result.LastScheduleTime == nil {
		t.Errorf("LastScheduleTime is nil")
	}
	if result.LastSuccessfulTime == nil {
		t.Errorf("LastSuccessfulTime is nil")
	}
}

func TestConvertCronJob_Suspended(t *testing.T) {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "suspended",
			Namespace: "ns",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			Suspend:  boolPtr(true),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
			},
		},
	}

	result := ConvertCronJob(cj)
	if !result.Suspend {
		t.Errorf("Suspend = false, want true")
	}
}

// =============================================================================
// TestConvertPersistentVolume
// =============================================================================

func TestConvertPersistentVolume_BasicFields(t *testing.T) {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pv-001",
			UID:               types.UID("pv-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"storage": "fast"},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("100Gi"),
			},
			StorageClassName:              "fast-ssd",
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       "ebs.csi.aws.com",
					VolumeHandle: "vol-12345",
				},
			},
			ClaimRef: &corev1.ObjectReference{
				Name:      "data-pvc",
				Namespace: "default",
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}

	result := ConvertPersistentVolume(pv)

	if result.Name != "pv-001" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Kind != "PersistentVolume" {
		t.Errorf("Kind = %q", result.Kind)
	}
	if result.Capacity != "100Gi" {
		t.Errorf("Capacity = %q", result.Capacity)
	}
	if result.Phase != "Bound" {
		t.Errorf("Phase = %q", result.Phase)
	}
	if result.StorageClass != "fast-ssd" {
		t.Errorf("StorageClass = %q", result.StorageClass)
	}
	if result.ReclaimPolicy != "Retain" {
		t.Errorf("ReclaimPolicy = %q", result.ReclaimPolicy)
	}
	if result.VolumeSourceType != "CSI" {
		t.Errorf("VolumeSourceType = %q, want %q", result.VolumeSourceType, "CSI")
	}
	if result.ClaimRefName != "data-pvc" {
		t.Errorf("ClaimRefName = %q", result.ClaimRefName)
	}
	if result.ClaimRefNS != "default" {
		t.Errorf("ClaimRefNS = %q", result.ClaimRefNS)
	}
	if len(result.AccessModes) != 2 {
		t.Errorf("AccessModes = %v", result.AccessModes)
	}
}

func TestConvertPersistentVolume_NoClaimRef(t *testing.T) {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "unbound-pv"},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("50Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{Server: "nfs.example.com", Path: "/data"},
			},
		},
		Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeAvailable},
	}

	result := ConvertPersistentVolume(pv)
	if result.ClaimRefName != "" {
		t.Errorf("ClaimRefName = %q, want empty", result.ClaimRefName)
	}
	if result.VolumeSourceType != "NFS" {
		t.Errorf("VolumeSourceType = %q, want %q", result.VolumeSourceType, "NFS")
	}
	if result.Phase != "Available" {
		t.Errorf("Phase = %q, want %q", result.Phase, "Available")
	}
}

func TestDetectVolumeSourceType(t *testing.T) {
	tests := []struct {
		name   string
		source corev1.PersistentVolumeSource
		want   string
	}{
		{"HostPath", corev1.PersistentVolumeSource{HostPath: &corev1.HostPathVolumeSource{}}, "HostPath"},
		{"NFS", corev1.PersistentVolumeSource{NFS: &corev1.NFSVolumeSource{}}, "NFS"},
		{"CSI", corev1.PersistentVolumeSource{CSI: &corev1.CSIPersistentVolumeSource{}}, "CSI"},
		{"Local", corev1.PersistentVolumeSource{Local: &corev1.LocalVolumeSource{}}, "Local"},
		{"Unknown", corev1.PersistentVolumeSource{}, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectVolumeSourceType(tt.source)
			if got != tt.want {
				t.Errorf("detectVolumeSourceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// TestConvertPersistentVolumeClaim
// =============================================================================

func TestConvertPersistentVolumeClaim_BasicFields(t *testing.T) {
	storageClass := "standard"
	volumeMode := corev1.PersistentVolumeFilesystem

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "data-pvc",
			Namespace:         "default",
			UID:               types.UID("pvc-uid"),
			CreationTimestamp: metav1.Time{Time: stableTime()},
			Labels:            map[string]string{"app": "db"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			VolumeName:       "pv-001",
			VolumeMode:       &volumeMode,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("50Gi"),
				},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("50Gi"),
			},
		},
	}

	result := ConvertPersistentVolumeClaim(pvc)

	if result.Name != "data-pvc" {
		t.Errorf("Name = %q", result.Name)
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace = %q", result.Namespace)
	}
	if result.Kind != "PersistentVolumeClaim" {
		t.Errorf("Kind = %q", result.Kind)
	}
	if result.Phase != "Bound" {
		t.Errorf("Phase = %q", result.Phase)
	}
	if result.VolumeName != "pv-001" {
		t.Errorf("VolumeName = %q", result.VolumeName)
	}
	if result.StorageClass != "standard" {
		t.Errorf("StorageClass = %q", result.StorageClass)
	}
	if result.RequestedCapacity != "50Gi" {
		t.Errorf("RequestedCapacity = %q", result.RequestedCapacity)
	}
	if result.ActualCapacity != "50Gi" {
		t.Errorf("ActualCapacity = %q", result.ActualCapacity)
	}
	if result.VolumeMode != "Filesystem" {
		t.Errorf("VolumeMode = %q, want %q", result.VolumeMode, "Filesystem")
	}
	if len(result.AccessModes) != 1 || result.AccessModes[0] != "ReadWriteOnce" {
		t.Errorf("AccessModes = %v", result.AccessModes)
	}
}

func TestConvertPersistentVolumeClaim_Pending(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-pvc",
			Namespace: "ns",
		},
		Spec: corev1.PersistentVolumeClaimSpec{},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimPending,
		},
	}

	result := ConvertPersistentVolumeClaim(pvc)
	if result.Phase != "Pending" {
		t.Errorf("Phase = %q, want %q", result.Phase, "Pending")
	}
	if result.VolumeName != "" {
		t.Errorf("VolumeName = %q, want empty", result.VolumeName)
	}
	if result.StorageClass != "" {
		t.Errorf("StorageClass = %q, want empty", result.StorageClass)
	}
}

// =============================================================================
// TestBuildCommonMeta
// =============================================================================

func TestBuildCommonMeta(t *testing.T) {
	now := time.Now()
	labels := map[string]string{"app": "test"}

	meta := buildCommonMeta("uid-123", "resource-name", "ns", "Pod", labels, now)

	if meta.UID != "uid-123" {
		t.Errorf("UID = %q", meta.UID)
	}
	if meta.Name != "resource-name" {
		t.Errorf("Name = %q", meta.Name)
	}
	if meta.Namespace != "ns" {
		t.Errorf("Namespace = %q", meta.Namespace)
	}
	if meta.Kind != "Pod" {
		t.Errorf("Kind = %q", meta.Kind)
	}
	if meta.Labels["app"] != "test" {
		t.Errorf("Labels = %v", meta.Labels)
	}
	if meta.CreatedAt != now {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestBuildCommonMeta_NilLabels(t *testing.T) {
	meta := buildCommonMeta("uid", "name", "ns", "ConfigMap", nil, time.Now())
	if meta.Labels != nil {
		t.Errorf("Labels = %v, want nil", meta.Labels)
	}
}

// =============================================================================
// TestIsPodReady
// =============================================================================

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name       string
		conditions []corev1.PodCondition
		want       bool
	}{
		{
			name: "ready",
			conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			want: true,
		},
		{
			name: "not ready",
			conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
			want: false,
		},
		{
			name:       "no conditions",
			conditions: nil,
			want:       false,
		},
		{
			name: "other conditions only",
			conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				Status: corev1.PodStatus{Conditions: tt.conditions},
			}
			got := isPodReady(pod)
			if got != tt.want {
				t.Errorf("isPodReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
