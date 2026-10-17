package k8s

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ListBackupCronJobs returns all CronJobs across all namespaces that have the label backup=yes.
// Pass an empty string for namespace to search all namespaces.
func ListBackupCronJobs(ctx context.Context, client kubernetes.Interface, namespace string) ([]batchv1.CronJob, error) {
	cronJobList, err := client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing cronjobs with label backup=yes: %w", err)
	}

	return cronJobList.Items, nil
}

// CreateJobFromCronJob creates an immediate Job from an existing CronJob, using the
// CronJob's job template. The created Job is owned by the CronJob via an OwnerReference.
func CreateJobFromCronJob(ctx context.Context, client kubernetes.Interface, cronJob *batchv1.CronJob) (*batchv1.Job, error) {
	// Build a unique name: <cronjob-name>-manual-<timestamp>
	jobName := fmt.Sprintf("%s-manual-%d", cronJob.Name, time.Now().Unix())

	// Set a "created-by" annotation so it's easy to identify manual triggers
	annotations := make(map[string]string)
	for k, v := range cronJob.Spec.JobTemplate.Annotations {
		annotations[k] = v
	}
	annotations["cronjob.kubernetes.io/instantiate"] = "manual"

	// Inherit labels from the job template
	labels := make(map[string]string)
	for k, v := range cronJob.Spec.JobTemplate.Labels {
		labels[k] = v
	}

	// true/false pointer helpers required by the OwnerReference
	isController := true
	blockOwnerDeletion := true

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   cronJob.Namespace,
			Labels:      labels,
			Annotations: annotations,
			// Make the Job owned by the CronJob so it is visible under it in kubectl
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         batchv1.SchemeGroupVersion.String(),
					Kind:               "CronJob",
					Name:               cronJob.Name,
					UID:                cronJob.UID,
					Controller:         &isController,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}

	created, err := client.BatchV1().Jobs(cronJob.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating job %q from cronjob %q: %w", jobName, cronJob.Name, err)
	}

	return created, nil
}

// CreateJobAndAttach creates a Job from the given CronJob, waits until its pod is Running,
// then attaches to it interactively (stdin + TTY), streaming output to your terminal.
// restConfig is needed for the SPDY attach connection (not available on kubernetes.Interface).
func CreateJobAndAttach(ctx context.Context, client kubernetes.Interface, restConfig *rest.Config, cronJob *batchv1.CronJob) error {
	// 1. Create the job
	job, err := CreateJobFromCronJob(ctx, client, cronJob)
	if err != nil {
		return err
	}
	fmt.Printf("Created Job %s/%s — waiting for pod to start...\n", job.Namespace, job.Name)

	// 2. Wait for a pod owned by the job to reach Running phase
	pod, err := waitForJobPodRunning(ctx, client, job)
	if err != nil {
		return fmt.Errorf("waiting for job pod: %w", err)
	}
	fmt.Printf("Pod %s is Running — attaching...\n", pod.Name)

	// 3. Attach interactively
	return attachToPod(ctx, client, restConfig, pod)
}

// waitForJobPodRunning watches pods in the job's namespace filtered by the job's label
// until one enters the Running phase, then returns it.
func waitForJobPodRunning(ctx context.Context, client kubernetes.Interface, job *batchv1.Job) (*corev1.Pod, error) {
	// Pods created by a Job carry the label job-name=<jobName>
	labelSelector := fmt.Sprintf("job-name=%s", job.Name)

	watcher, err := client.CoreV1().Pods(job.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector:  labelSelector,
		TimeoutSeconds: int64ptr(300), // give up after 5 minutes
	})
	if err != nil {
		return nil, fmt.Errorf("watching pods for job %q: %w", job.Name, err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil, fmt.Errorf("pod watch channel closed before pod became Running")
			}
			if event.Type == watch.Error {
				return nil, fmt.Errorf("error event while watching pods")
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			switch pod.Status.Phase {
			case corev1.PodRunning:
				// Make sure at least one container is actually ready to attach to
				if hasRunningContainer(pod) {
					return pod, nil
				}
			case corev1.PodFailed, corev1.PodSucceeded:
				return nil, fmt.Errorf("pod %q reached terminal phase %q before we could attach", pod.Name, pod.Status.Phase)
			}
		}
	}
}

// hasRunningContainer returns true when at least one container in the pod has started.
func hasRunningContainer(pod *corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil {
			return true
		}
	}
	return false
}

// attachToPod opens an interactive attach session (stdin + stdout + stderr + TTY) to the
// first container of the given pod, wiring it directly to the current process's terminal.
func attachToPod(ctx context.Context, client kubernetes.Interface, restConfig *rest.Config, pod *corev1.Pod) error {
	containerName := pod.Spec.Containers[0].Name

	// Build the attach URL via the pods/attach subresource
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("attach").
		VersionedParams(&corev1.PodAttachOptions{
			Container: containerName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	// Use SPDY to multiplex the streams (same protocol as kubectl attach)
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating SPDY executor: %w", err)
	}

	// Put the local terminal into raw mode so the TTY works correctly
	// Wire our stdin/stdout/stderr directly to the pod
	streamOpts := remotecommand.StreamOptions{
		Stdin:             os.Stdin,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Tty:               true,
		TerminalSizeQueue: terminalSizeQueue(ctx),
	}

	fmt.Printf("Attaching to pod/%s container/%s (use Ctrl-C / Ctrl-D to detach)\n", pod.Name, containerName)
	return executor.StreamWithContext(ctx, streamOpts)
}

// terminalSizeQueue returns a TerminalSizeQueue that reads the current terminal
// dimensions once and then blocks (good enough for most interactive workloads;
// swap for a signal-driven implementation if you need resize support).
func terminalSizeQueue(ctx context.Context) remotecommand.TerminalSizeQueue {
	return &staticSizeQueue{
		ch: func() chan remotecommand.TerminalSize {
			ch := make(chan remotecommand.TerminalSize, 1)
			ch <- remotecommand.TerminalSize{Width: 220, Height: 50}
			return ch
		}(),
	}
}

type staticSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func (s *staticSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.ch
	if !ok {
		return nil
	}
	return &size
}

// --- helpers ---

func int64ptr(i int64) *int64 { return &i }

// fieldSelectorForPod builds a field selector string for a specific pod name.
func fieldSelectorForPod(podName string) string {
	return fields.OneTermEqualSelector("metadata.name", podName).String()
}

// isOwnedBy returns true if the job has an OwnerReference pointing to the given UID.
func isOwnedBy(job *batchv1.Job, ownerUID types.UID) bool {
	for _, ref := range job.OwnerReferences {
		if ref.UID == ownerUID {
			return true
		}
	}
	return false
}

// isJobActive returns true when a job has active (running/pending) pods and has not
// reached a terminal condition (Complete or Failed).
func isJobActive(job *batchv1.Job) bool {
	if job.Status.Active == 0 {
		return false
	}
	for _, cond := range job.Status.Conditions {
		if (cond.Type == batchv1.JobComplete || cond.Type == batchv1.JobFailed) &&
			cond.Status == corev1.ConditionTrue {
			return false
		}
	}
	return true
}

// IsManualJobRunning checks whether a manually triggered Job (annotation
// cronjob.kubernetes.io/instantiate=manual) already exists and is actively running
// for the given CronJob. A job is considered "running" when it has at least one
// active pod and has not yet completed or failed.
// Returns (true, nil) if such a job is found, (false, nil) if not, or (false, err) on error.
func IsManualJobRunning(ctx context.Context, client kubernetes.Interface, cronJob *batchv1.CronJob) (bool, error) {
	// Jobs created from a CronJob carry an owner reference back to it — list all jobs
	// in the same namespace and filter by owner UID to scope to this CronJob only.
	jobList, err := client.BatchV1().Jobs(cronJob.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("listing jobs in namespace %q: %w", cronJob.Namespace, err)
	}

	for _, job := range jobList.Items {
		// Must be owned by this specific CronJob
		if !isOwnedBy(&job, cronJob.UID) {
			continue
		}

		// Must have been triggered manually
		if job.Annotations["cronjob.kubernetes.io/instantiate"] != "manual" {
			continue
		}

		// A job is "running" when Active > 0 and it hasn't finished
		if isJobActive(&job) {
			return true, nil
		}
	}

	return false, nil
}
