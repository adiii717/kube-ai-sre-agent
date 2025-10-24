package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/adiii717/kube-ai-sre-agent/pkg/config"
	"github.com/adiii717/kube-ai-sre-agent/pkg/events"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// Controller watches pods and spawns analysis jobs
type Controller struct {
	clientset     *kubernetes.Clientset
	config        *config.Config
	detector      *events.Detector
	namespace     string
	watchNamespace string
	llmAPIKey     string
	slackWebhook  string
}

// New creates a new controller
func New(clientset *kubernetes.Clientset, cfg *config.Config, namespace, watchNamespace, llmAPIKey, slackWebhook string) *Controller {
	return &Controller{
		clientset:      clientset,
		config:         cfg,
		detector:       events.NewDetector(&cfg.Events),
		namespace:      namespace,
		watchNamespace: watchNamespace,
		llmAPIKey:      llmAPIKey,
		slackWebhook:   slackWebhook,
	}
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) error {
	klog.Info("Starting kube-ai-sre-agent controller")

	// Determine which namespace to watch
	ns := c.watchNamespace
	if ns == "" {
		ns = c.namespace
	}

	// Create informer factory
	factory := informers.NewSharedInformerFactoryWithOptions(
		c.clientset,
		time.Minute,
		informers.WithNamespace(ns),
	)

	// Create pod informer
	podInformer := factory.Core().V1().Pods().Informer()

	// Add event handler
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.handlePodUpdate,
	})

	// Start informer
	factory.Start(ctx.Done())

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	klog.Info("Controller started successfully")

	// Wait until context is cancelled
	<-ctx.Done()
	return nil
}

func (c *Controller) handlePodUpdate(oldObj, newObj interface{}) {
	pod, ok := newObj.(*corev1.Pod)
	if !ok {
		return
	}

	// Detect incident
	incident := c.detector.DetectIncident(pod)
	if incident == nil {
		return
	}

	klog.Infof("Detected %s for pod %s/%s", incident.EventType, incident.Namespace, incident.PodName)

	// Spawn analysis job
	if err := c.spawnAnalysisJob(context.Background(), incident); err != nil {
		klog.Errorf("Failed to spawn analysis job: %v", err)
	}
}

func (c *Controller) spawnAnalysisJob(ctx context.Context, incident *events.PodIncident) error {
	jobName := fmt.Sprintf("analyze-%s-%d", incident.PodName, time.Now().Unix())

	// Parse resources
	cpuRequest, _ := resource.ParseQuantity(c.config.Analyzer.Resources.Requests.CPU)
	memRequest, _ := resource.ParseQuantity(c.config.Analyzer.Resources.Requests.Memory)
	cpuLimit, _ := resource.ParseQuantity(c.config.Analyzer.Resources.Limits.CPU)
	memLimit, _ := resource.ParseQuantity(c.config.Analyzer.Resources.Limits.Memory)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "kube-ai-sre-agent",
				"app.kubernetes.io/component": "analyzer",
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &c.config.Analyzer.TTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "kube-ai-sre-agent",
					RestartPolicy:      corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    int64Ptr(65532),
						FSGroup:      int64Ptr(65532),
					},
					Containers: []corev1.Container{
						{
							Name:  "analyzer",
							Image: c.config.Analyzer.Image,
							Env: []corev1.EnvVar{
								{Name: "POD_NAME", Value: incident.PodName},
								{Name: "POD_NAMESPACE", Value: incident.Namespace},
								{Name: "EVENT_TYPE", Value: string(incident.EventType)},
								{Name: "CONTAINER_NAME", Value: incident.ContainerName},
								{Name: "REASON", Value: incident.Reason},
								{Name: "MESSAGE", Value: incident.Message},
								{Name: "LLM_PROVIDER", Value: c.config.LLM.Provider},
								{Name: "LLM_API_KEY", Value: c.llmAPIKey},
								{Name: "SLACK_WEBHOOK_URL", Value: c.slackWebhook},
								{Name: "SLACK_ENABLED", Value: fmt.Sprintf("%t", c.config.Slack.Enabled)},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cpuRequest,
									corev1.ResourceMemory: memRequest,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    cpuLimit,
									corev1.ResourceMemory: memLimit,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.BatchV1().Jobs(c.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	klog.Infof("Spawned analysis job %s for pod %s/%s", jobName, incident.Namespace, incident.PodName)
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
