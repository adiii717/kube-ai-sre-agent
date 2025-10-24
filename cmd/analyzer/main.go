package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/adiii717/kube-ai-sre-agent/pkg/llm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)

	// Get environment variables
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	eventType := os.Getenv("EVENT_TYPE")
	containerName := os.Getenv("CONTAINER_NAME")
	llmProvider := os.Getenv("LLM_PROVIDER")
	llmAPIKey := os.Getenv("LLM_API_KEY")
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
	slackEnabled, _ := strconv.ParseBool(os.Getenv("SLACK_ENABLED"))

	klog.Infof("Analyzing incident: %s for pod %s/%s", eventType, podNamespace, podName)

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Fetch pod logs
	logs, err := fetchPodLogs(clientset, podNamespace, podName, containerName)
	if err != nil {
		klog.Errorf("Failed to fetch logs: %v", err)
		logs = fmt.Sprintf("Failed to fetch logs: %v", err)
	}

	// Create LLM client
	llmClient, err := llm.NewClient(llm.Provider(llmProvider), llmAPIKey)
	if err != nil {
		klog.Fatalf("Failed to create LLM client: %v", err)
	}

	// Analyze with LLM
	analysis, err := llmClient.Analyze(eventType, podName, podNamespace, logs)
	if err != nil {
		klog.Fatalf("Failed to analyze: %v", err)
	}

	klog.Infof("Analysis:\n%s", analysis)

	// Send to Slack if enabled
	if slackEnabled && slackWebhook != "" {
		if err := sendSlackNotification(slackWebhook, eventType, podNamespace, podName, analysis); err != nil {
			klog.Errorf("Failed to send Slack notification: %v", err)
		} else {
			klog.Info("Slack notification sent successfully")
		}
	}

	klog.Info("Analysis complete")
}

func fetchPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string) (string, error) {
	podLogOpts := &corev1.PodLogOptions{
		TailLines: int64Ptr(100),
	}

	if containerName != "" {
		podLogOpts.Container = containerName
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := make([]byte, 2000)
	numBytes, err := podLogs.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	return string(buf[:numBytes]), nil
}

func sendSlackNotification(webhook, eventType, namespace, podName, analysis string) error {
	// TODO: Implement actual Slack API call
	klog.Infof("Sending to Slack: %s incident for %s/%s", eventType, namespace, podName)
	return nil
}

func int64Ptr(i int64) *int64 {
	return &i
}
