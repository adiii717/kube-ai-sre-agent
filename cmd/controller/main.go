package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/adiii717/kube-ai-sre-agent/pkg/config"
	"github.com/adiii717/kube-ai-sre-agent/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	var kubeconfig string
	var configPath string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not provided)")
	flag.StringVar(&configPath, "config", "/etc/config/config.yaml", "Path to configuration file")
	klog.InitFlags(nil)
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		klog.Fatalf("Failed to load config: %v", err)
	}

	// Get environment variables
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	watchNamespace := os.Getenv("WATCH_NAMESPACE")
	llmAPIKey := os.Getenv("LLM_API_KEY")
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")

	cooldownMinutes := 5 // default
	if envCooldown := os.Getenv("COOLDOWN_MINUTES"); envCooldown != "" {
		if minutes, err := strconv.Atoi(envCooldown); err == nil {
			cooldownMinutes = minutes
		}
	}
	cooldown := time.Duration(cooldownMinutes) * time.Minute

	// Escalation config
	escalationEnabled := true
	if env := os.Getenv("ESCALATION_ENABLED"); env != "" {
		escalationEnabled, _ = strconv.ParseBool(env)
	}

	escalationThreshold := 10 // default
	if env := os.Getenv("ESCALATION_THRESHOLD"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			escalationThreshold = val
		}
	}

	silenceDurationMinutes := 60 // default 1 hour
	if env := os.Getenv("SILENCE_DURATION_MINUTES"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			silenceDurationMinutes = val
		}
	}
	silenceDuration := time.Duration(silenceDurationMinutes) * time.Minute

	// Create Kubernetes client
	var restConfig *rest.Config
	if kubeconfig != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		restConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create controller
	ctrl := controller.New(clientset, cfg, namespace, watchNamespace, llmAPIKey, slackWebhook, cooldown, escalationEnabled, escalationThreshold, silenceDuration)

	// Setup signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run controller
	if err := ctrl.Run(ctx); err != nil {
		klog.Fatalf("Controller error: %v", err)
	}

	klog.Info("Controller stopped")
}
