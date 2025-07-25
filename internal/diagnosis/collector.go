package diagnosis

import (
	"NeuroController/internal/watcher/abnormal"
	"NeuroController/model"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ✅ 全局内存中的事件池（原始收集的事件）
var eventPool = make([]model.LogEvent, 0)

// 内部工具函数：将事件追加到事件池中（线程安全，仅限内部使用）
func appendToEventPool(event model.LogEvent) {

	log.Printf("📥 收到事件: %s/%s | %s | %s", event.Namespace, event.Name, event.Kind, event.ReasonCode)
	
	if event.Kind == "Pod" && event.Name == "default" {
		log.Printf("⚠️ 异常事件字段异常: Pod 名称为 'default'，可能未正确识别 → Namespace=%s, Message=%s",
			event.Namespace, event.Message)

	}
	if event.ReasonCode == "" {
		log.Printf("❌ 缺少 ReasonCode: %s/%s → %s", event.Namespace, event.Name, event.Message)
	}
	mu.Lock()
	defer mu.Unlock()
	eventPool = append(eventPool, event)
}

// ✅ 收集 Pod 异常事件
// 由 PodWatcher 调用；封装所有内部逻辑
func CollectPodAbnormalEvent(pod corev1.Pod, reason *abnormal.PodAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Pod",
		Namespace:  pod.Namespace,
		Name:       pod.Name,
		Node:       pod.Spec.NodeName,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)
}

// ✅ 收集 Node 异常事件
func CollectNodeAbnormalEvent(node corev1.Node, reason *abnormal.NodeAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Node",
		Namespace:  "--", // Node 没有命名空间
		Name:       node.Name,
		Node:       node.Name,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

}

// ✅ 收集核心 Event 资源的异常事件
func CollectEventAbnormalEvent(ev corev1.Event, reason *abnormal.EventAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       ev.InvolvedObject.Kind,
		Namespace:  ev.InvolvedObject.Namespace,
		Name:       ev.InvolvedObject.Name,
		Node:       ev.Source.Host,
		ReasonCode: reason.Code,
		Category:   "Event", // 分类用于分析
		Severity:   reason.Severity,
		Message:    ev.Message,
	}
	appendToEventPool(event)
}

// ✅ 收集 Endpoints 异常事件
func CollectEndpointAbnormalEvent(ep corev1.Endpoints, reason *abnormal.EndpointAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Endpoints",
		Namespace:  ep.Namespace,
		Name:       ep.Name,
		Node:       "--",
		ReasonCode: reason.Code,
		Category:   "Endpoint", // 用于分组和过滤
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)
}

// ✅ 收集 Deployment 异常事件
func CollectDeploymentAbnormalEvent(deploy appsv1.Deployment, reason *abnormal.DeploymentAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Deployment",
		Namespace:  deploy.Namespace,
		Name:       deploy.Name,
		Node:       "--",
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

}

// ✅ 收集 Service 异常事件
func CollectServiceAbnormalEvent(svc corev1.Service, reason *abnormal.ServiceAbnormalReason) {
	event := model.LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Service",
		Namespace:  svc.Namespace,
		Name:       svc.Name,
		Node:       "--",
		ReasonCode: reason.Code,
		Category:   "Warning", // 可选：可扩展为从 reason 中提取
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

}
