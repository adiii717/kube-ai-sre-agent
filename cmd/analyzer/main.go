package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/adiii717/kube-ai-sre-agent/pkg/llm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Fetch pod details (describe)
	podInfo, err := getPodInfo(clientset, podNamespace, podName)
	if err != nil {
		klog.Errorf("Failed to get pod info: %v", err)
		podInfo = fmt.Sprintf("Failed to get pod info: %v", err)
	}

	// Fetch pod logs
	logs, err := fetchPodLogs(clientset, podNamespace, podName, containerName)
	if err != nil || logs == "" {
		klog.Warningf("Failed to fetch logs or logs empty: %v", err)
		logs = "(No logs available)"
	}

	// Combine pod info and logs
	context := fmt.Sprintf("Pod Information:\n%s\n\nPod Logs:\n%s", podInfo, logs)

	// Create LLM client
	llmClient, err := llm.NewClient(llm.Provider(llmProvider), llmAPIKey)
	if err != nil {
		klog.Fatalf("Failed to create LLM client: %v", err)
	}

	// Analyze with LLM
	analysis, err := llmClient.Analyze(eventType, podName, podNamespace, context)
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

func getPodInfo(clientset *kubernetes.Clientset, namespace, podName string) (string, error) {
	ctx := context.Background()

	// Get pod details
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var info string
	info += fmt.Sprintf("Status: %s\n", pod.Status.Phase)
	info += fmt.Sprintf("Reason: %s\n", pod.Status.Reason)
	info += fmt.Sprintf("Message: %s\n", pod.Status.Message)
	info += fmt.Sprintf("Host IP: %s\n", pod.Status.HostIP)
	info += fmt.Sprintf("Pod IP: %s\n", pod.Status.PodIP)
	info += fmt.Sprintf("Start Time: %v\n", pod.Status.StartTime)

	// Container statuses
	info += "\nContainer Statuses:\n"
	for _, cs := range pod.Status.ContainerStatuses {
		info += fmt.Sprintf("  - %s: Ready=%v, RestartCount=%d\n", cs.Name, cs.Ready, cs.RestartCount)

		if cs.State.Waiting != nil {
			info += fmt.Sprintf("    Waiting: %s - %s\n", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
		if cs.State.Running != nil {
			info += fmt.Sprintf("    Running since: %v\n", cs.State.Running.StartedAt)
		}
		if cs.State.Terminated != nil {
			info += fmt.Sprintf("    Terminated: ExitCode=%d, Reason=%s, Message=%s\n",
				cs.State.Terminated.ExitCode, cs.State.Terminated.Reason, cs.State.Terminated.Message)
		}

		if cs.LastTerminationState.Terminated != nil {
			info += fmt.Sprintf("    Last Termination: ExitCode=%d, Reason=%s\n",
				cs.LastTerminationState.Terminated.ExitCode, cs.LastTerminationState.Terminated.Reason)
		}
	}

	// Conditions
	info += "\nConditions:\n"
	for _, cond := range pod.Status.Conditions {
		info += fmt.Sprintf("  - %s: %s (Reason: %s)\n", cond.Type, cond.Status, cond.Reason)
	}

	return info, nil
}

func fetchPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string) (string, error) {
	// Try to get current logs first
	logs, err := getPodLogsWithOptions(clientset, namespace, podName, containerName, false)
	if err == nil && logs != "" {
		return logs, nil
	}

	// If current logs fail or empty, try previous container logs (for crashed pods)
	klog.V(2).Infof("Current logs unavailable, trying previous container logs")
	logs, err = getPodLogsWithOptions(clientset, namespace, podName, containerName, true)
	if err == nil && logs != "" {
		return fmt.Sprintf("[Previous Container Logs]\n%s", logs), nil
	}

	return "", fmt.Errorf("no logs available from current or previous container")
}

func getPodLogsWithOptions(clientset *kubernetes.Clientset, namespace, podName, containerName string, previous bool) (string, error) {
	podLogOpts := &corev1.PodLogOptions{
		TailLines: int64Ptr(100),
		Previous:  previous,
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

	buf, err := io.ReadAll(podLogs)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func sendSlackNotification(webhook, eventType, namespace, podName, analysis string) error {
	// TODO: Implement actual Slack API call
	klog.Infof("Sending to Slack: %s incident for %s/%s", eventType, namespace, podName)
	return nil
}

func int64Ptr(i int64) *int64 {
	return &i
}
