package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// =============================================================================
// TestExecute_Scale
// =============================================================================

func TestExecute_Scale_Success(t *testing.T) {
	var calledNs, calledName string
	var calledReplicas int32

	genericRepo := &mock.GenericRepository{
		ScaleDeploymentFn: func(ctx context.Context, namespace, name string, replicas int32) error {
			calledNs = namespace
			calledName = name
			calledReplicas = replicas
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-scale-1",
		Action:    command.ActionScale,
		Namespace: "default",
		Name:      "nginx",
		Params:    map[string]any{"replicas": float64(3)},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledNs != "default" {
		t.Errorf("expected namespace 'default', got '%s'", calledNs)
	}
	if calledName != "nginx" {
		t.Errorf("expected name 'nginx', got '%s'", calledName)
	}
	if calledReplicas != 3 {
		t.Errorf("expected replicas 3, got %d", calledReplicas)
	}
}

func TestExecute_Scale_InvalidParams(t *testing.T) {
	genericRepo := &mock.GenericRepository{
		ScaleDeploymentFn: func(ctx context.Context, namespace, name string, replicas int32) error {
			// replicas will be 0 (zero value) since "invalid" can't parse to int32
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	// Params that can't be marshaled properly -- use a channel which json.Marshal can't handle
	cmd := &command.Command{
		ID:     "cmd-scale-2",
		Action: command.ActionScale,
		Params: map[string]any{"replicas": make(chan int)},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for invalid params")
	}
}

func TestExecute_Scale_RepoError(t *testing.T) {
	genericRepo := &mock.GenericRepository{
		ScaleDeploymentFn: func(ctx context.Context, namespace, name string, replicas int32) error {
			return fmt.Errorf("deployment not found")
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-scale-3",
		Action:    command.ActionScale,
		Namespace: "default",
		Name:      "nginx",
		Params:    map[string]any{"replicas": float64(3)},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for repo error")
	}
	if !strings.Contains(result.Error, "deployment not found") {
		t.Errorf("expected error to contain 'deployment not found', got '%s'", result.Error)
	}
}

// =============================================================================
// TestExecute_Restart
// =============================================================================

func TestExecute_Restart_Success(t *testing.T) {
	var calledNs, calledName string

	genericRepo := &mock.GenericRepository{
		RestartDeploymentFn: func(ctx context.Context, namespace, name string) error {
			calledNs = namespace
			calledName = name
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-restart-1",
		Action:    command.ActionRestart,
		Namespace: "production",
		Name:      "api-server",
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledNs != "production" {
		t.Errorf("expected namespace 'production', got '%s'", calledNs)
	}
	if calledName != "api-server" {
		t.Errorf("expected name 'api-server', got '%s'", calledName)
	}
}

// =============================================================================
// TestExecute_UpdateImage
// =============================================================================

func TestExecute_UpdateImage_Success(t *testing.T) {
	var calledContainer, calledImage string

	genericRepo := &mock.GenericRepository{
		UpdateDeploymentImageFn: func(ctx context.Context, namespace, name, container, image string) error {
			calledContainer = container
			calledImage = image
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-update-1",
		Action:    command.ActionUpdateImage,
		Namespace: "default",
		Name:      "nginx",
		Params:    map[string]any{"container": "web", "image": "nginx:latest"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledContainer != "web" {
		t.Errorf("expected container 'web', got '%s'", calledContainer)
	}
	if calledImage != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got '%s'", calledImage)
	}
}

func TestExecute_UpdateImage_MissingImage(t *testing.T) {
	genericRepo := &mock.GenericRepository{}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-update-2",
		Action:    command.ActionUpdateImage,
		Namespace: "default",
		Name:      "nginx",
		Params:    map[string]any{"container": "web"},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing image")
	}
	if !strings.Contains(result.Error, "image is required") {
		t.Errorf("expected error to contain 'image is required', got '%s'", result.Error)
	}
}

// =============================================================================
// TestExecute_GetLogs
// =============================================================================

func TestExecute_GetLogs_Success(t *testing.T) {
	var calledOpts model.LogOptions

	podRepo := &mock.PodRepository{
		GetFn: func(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
			return &cluster.Pod{
				Containers: []cluster.PodContainerDetail{
					{Name: "app"},
				},
			}, nil
		},
		GetLogsFn: func(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
			calledOpts = opts
			return "line1\nline2\nline3", nil
		},
	}

	svc := &commandService{
		podRepo:     podRepo,
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-logs-1",
		Action:    command.ActionGetLogs,
		Namespace: "default",
		Name:      "nginx-pod",
		Params:    map[string]any{"container": "app", "tailLines": float64(50)},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "line1") {
		t.Errorf("expected output to contain 'line1', got '%s'", result.Output)
	}
	if calledOpts.Container != "app" {
		t.Errorf("expected container 'app', got '%s'", calledOpts.Container)
	}
	if calledOpts.TailLines != 50 {
		t.Errorf("expected tailLines 50, got %d", calledOpts.TailLines)
	}
}

func TestExecute_GetLogs_AutoSelectContainer(t *testing.T) {
	var calledOpts model.LogOptions

	podRepo := &mock.PodRepository{
		GetFn: func(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
			return &cluster.Pod{
				Containers: []cluster.PodContainerDetail{
					{Name: "app"},
					{Name: "linkerd-proxy"},
				},
			}, nil
		},
		GetLogsFn: func(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
			calledOpts = opts
			return "auto-selected logs", nil
		},
	}

	svc := &commandService{
		podRepo:     podRepo,
		genericRepo: &mock.GenericRepository{},
	}

	// Container empty -- should auto-select "app" (non-sidecar)
	cmd := &command.Command{
		ID:        "cmd-logs-2",
		Action:    command.ActionGetLogs,
		Namespace: "default",
		Name:      "nginx-pod",
		Params:    map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledOpts.Container != "app" {
		t.Errorf("expected auto-selected container 'app', got '%s'", calledOpts.Container)
	}
}

func TestExecute_GetLogs_TailLinesClamp(t *testing.T) {
	tests := []struct {
		name         string
		inputTail    float64
		expectedTail int64
	}{
		{"ZeroDefaults100", 0, 100},
		{"NegativeDefaults100", -1, 100},
		{"Over200ClampedTo200", 500, 200},
		{"NormalValueKept", 150, 150},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calledOpts model.LogOptions

			podRepo := &mock.PodRepository{
				GetFn: func(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
					return &cluster.Pod{
						Containers: []cluster.PodContainerDetail{{Name: "app"}},
					}, nil
				},
				GetLogsFn: func(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
					calledOpts = opts
					return "logs", nil
				},
			}

			svc := &commandService{
				podRepo:     podRepo,
				genericRepo: &mock.GenericRepository{},
			}

			cmd := &command.Command{
				ID:        "cmd-logs-clamp",
				Action:    command.ActionGetLogs,
				Namespace: "default",
				Name:      "nginx-pod",
				Params:    map[string]any{"tailLines": tc.inputTail},
			}

			result := svc.Execute(context.Background(), cmd)

			if !result.Success {
				t.Fatalf("expected success, got error: %s", result.Error)
			}
			if calledOpts.TailLines != tc.expectedTail {
				t.Errorf("expected tailLines %d, got %d", tc.expectedTail, calledOpts.TailLines)
			}
		})
	}
}

func TestExecute_GetLogs_FallbackToSidecar(t *testing.T) {
	var calledOpts model.LogOptions

	podRepo := &mock.PodRepository{
		GetFn: func(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
			return &cluster.Pod{
				Containers: []cluster.PodContainerDetail{
					{Name: "linkerd-proxy"},
				},
			}, nil
		},
		GetLogsFn: func(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
			calledOpts = opts
			return "sidecar logs", nil
		},
	}

	svc := &commandService{
		podRepo:     podRepo,
		genericRepo: &mock.GenericRepository{},
	}

	// Pod only has sidecar containers, should fallback to first container
	cmd := &command.Command{
		ID:        "cmd-logs-fallback",
		Action:    command.ActionGetLogs,
		Namespace: "default",
		Name:      "proxy-pod",
		Params:    map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledOpts.Container != "linkerd-proxy" {
		t.Errorf("expected fallback container 'linkerd-proxy', got '%s'", calledOpts.Container)
	}
}

// =============================================================================
// TestExecute_Dynamic
// =============================================================================

func TestExecute_Dynamic_AISource_SummarizeList(t *testing.T) {
	genericRepo := &mock.GenericRepository{
		ExecuteFn: func(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error) {
			return &model.DynamicResponse{
				StatusCode: 200,
				Body:       []byte(`{"kind":"PodList","items":[{"metadata":{"name":"nginx","namespace":"default","creationTimestamp":"2025-01-01T00:00:00Z"},"status":{"phase":"Running","containerStatuses":[]}}]}`),
			}, nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-dynamic-ai",
		Action:    command.ActionDynamic,
		Namespace: "default",
		Source:    "ai",
		Params: map[string]any{
			"command": "list",
			"kind":    "Pod",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	// AI source + list -> summarizeList produces table format with "Pod (N):"
	if !strings.Contains(result.Output, "Pod (1):") {
		t.Errorf("expected summarized table with 'Pod (1):', got '%s'", result.Output)
	}
	if !strings.Contains(result.Output, "nginx") {
		t.Errorf("expected output to contain 'nginx', got '%s'", result.Output)
	}
}

func TestExecute_Dynamic_NonAISource(t *testing.T) {
	bodyJSON := `{"kind":"PodList","items":[{"metadata":{"name":"nginx","namespace":"default","creationTimestamp":"2025-01-01T00:00:00Z","managedFields":[{"manager":"kubectl"}]},"status":{"phase":"Running"}}]}`

	genericRepo := &mock.GenericRepository{
		ExecuteFn: func(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error) {
			return &model.DynamicResponse{
				StatusCode: 200,
				Body:       []byte(bodyJSON),
			}, nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-dynamic-web",
		Action:    command.ActionDynamic,
		Namespace: "default",
		Source:    "web",
		Params: map[string]any{
			"command": "list",
			"kind":    "Pod",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Non-AI source returns raw JSON (with managedFields stripped)
	if strings.Contains(result.Output, "managedFields") {
		t.Error("expected managedFields to be stripped from output")
	}

	// Should still be valid JSON (not table format)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.Output), &parsed); err != nil {
		t.Errorf("expected valid JSON output, got parse error: %v", err)
	}
}

func TestExecute_Dynamic_HTTP4xxError(t *testing.T) {
	genericRepo := &mock.GenericRepository{
		ExecuteFn: func(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error) {
			return &model.DynamicResponse{
				StatusCode: 404,
				Body:       []byte(`{"message":"not found"}`),
			}, nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-dynamic-404",
		Action:    command.ActionDynamic,
		Namespace: "default",
		Params: map[string]any{
			"command": "get",
			"kind":    "Pod",
		},
		Name: "nonexistent",
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for HTTP 404")
	}
	if !strings.Contains(result.Error, "404") {
		t.Errorf("expected error to contain '404', got '%s'", result.Error)
	}
}

func TestExecute_Dynamic_MissingCommand(t *testing.T) {
	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-dynamic-nocommand",
		Action:    command.ActionDynamic,
		Namespace: "default",
		Params: map[string]any{
			"kind": "Pod",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing command")
	}
	if !strings.Contains(result.Error, "command is required") {
		t.Errorf("expected error to contain 'command is required', got '%s'", result.Error)
	}
}

func TestExecute_Dynamic_MissingKind(t *testing.T) {
	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-dynamic-nokind",
		Action:    command.ActionDynamic,
		Namespace: "default",
		Params: map[string]any{
			"command": "list",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing kind")
	}
	if !strings.Contains(result.Error, "kind is required") {
		t.Errorf("expected error to contain 'kind is required', got '%s'", result.Error)
	}
}

// =============================================================================
// TestExecute_Delete
// =============================================================================

func TestExecute_Delete_Pod(t *testing.T) {
	var deletePodCalled bool
	var deleteCalled bool

	genericRepo := &mock.GenericRepository{
		DeletePodFn: func(ctx context.Context, namespace, name string, opts model.DeleteOptions) error {
			deletePodCalled = true
			return nil
		},
		DeleteFn: func(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error {
			deleteCalled = true
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-delete-pod",
		Action:    command.ActionDelete,
		Kind:      "Pod",
		Namespace: "default",
		Name:      "nginx-pod",
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !deletePodCalled {
		t.Error("expected DeletePodFn to be called for Pod kind")
	}
	if deleteCalled {
		t.Error("expected DeleteFn NOT to be called for Pod kind")
	}
}

func TestExecute_Delete_Generic(t *testing.T) {
	var deletePodCalled bool
	var calledKind string

	genericRepo := &mock.GenericRepository{
		DeletePodFn: func(ctx context.Context, namespace, name string, opts model.DeleteOptions) error {
			deletePodCalled = true
			return nil
		},
		DeleteFn: func(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error {
			calledKind = kind
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:        "cmd-delete-deploy",
		Action:    command.ActionDelete,
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "nginx",
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if deletePodCalled {
		t.Error("expected DeletePodFn NOT to be called for Deployment kind")
	}
	if calledKind != "Deployment" {
		t.Errorf("expected DeleteFn called with kind 'Deployment', got '%s'", calledKind)
	}
}

// =============================================================================
// TestExecute_Cordon_Uncordon
// =============================================================================

func TestExecute_Cordon_Success(t *testing.T) {
	var calledName string

	genericRepo := &mock.GenericRepository{
		CordonNodeFn: func(ctx context.Context, name string) error {
			calledName = name
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:     "cmd-cordon-1",
		Action: command.ActionCordon,
		Name:   "node-1",
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledName != "node-1" {
		t.Errorf("expected node name 'node-1', got '%s'", calledName)
	}
}

func TestExecute_Uncordon_Success(t *testing.T) {
	var calledName string

	genericRepo := &mock.GenericRepository{
		UncordonNodeFn: func(ctx context.Context, name string) error {
			calledName = name
			return nil
		},
	}

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: genericRepo,
	}

	cmd := &command.Command{
		ID:     "cmd-uncordon-1",
		Action: command.ActionUncordon,
		Name:   "node-2",
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if calledName != "node-2" {
		t.Errorf("expected node name 'node-2', got '%s'", calledName)
	}
}

// =============================================================================
// TestExecute_UnknownAction
// =============================================================================

func TestExecute_UnknownAction(t *testing.T) {
	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:     "cmd-unknown",
		Action: "unknown",
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for unknown action")
	}
	if !strings.Contains(result.Error, "unknown action") {
		t.Errorf("expected error to contain 'unknown action', got '%s'", result.Error)
	}
}

// =============================================================================
// TestExecute_ResultFormat
// =============================================================================

func TestExecute_ResultFormat_CommandID(t *testing.T) {
	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-format-123",
		Action:    command.ActionRestart,
		Namespace: "default",
		Name:      "nginx",
	}

	result := svc.Execute(context.Background(), cmd)

	if result.CommandID != "cmd-format-123" {
		t.Errorf("expected CommandID 'cmd-format-123', got '%s'", result.CommandID)
	}
}

func TestExecute_ResultFormat_ExecTime(t *testing.T) {
	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-format-time",
		Action:    command.ActionRestart,
		Namespace: "default",
		Name:      "nginx",
	}

	result := svc.Execute(context.Background(), cmd)

	if result.ExecTime <= 0 {
		t.Errorf("expected ExecTime > 0, got %v", result.ExecTime)
	}
}

func TestExecute_ResultFormat_ExecutedAt(t *testing.T) {
	before := time.Now()

	svc := &commandService{
		podRepo:     &mock.PodRepository{},
		genericRepo: &mock.GenericRepository{},
	}

	cmd := &command.Command{
		ID:        "cmd-format-at",
		Action:    command.ActionRestart,
		Namespace: "default",
		Name:      "nginx",
	}

	result := svc.Execute(context.Background(), cmd)

	after := time.Now()

	if result.ExecutedAt.Before(before) || result.ExecutedAt.After(after) {
		t.Errorf("expected ExecutedAt between %v and %v, got %v", before, after, result.ExecutedAt)
	}
}
