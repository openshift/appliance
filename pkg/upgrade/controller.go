package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	config "github.com/openshift/api/config"
	configv1 "github.com/openshift/api/config/v1"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	core "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/appliance/pkg/upgrade/annotations"
	"github.com/openshift/appliance/pkg/upgrade/labels"
)

// ControllerBuilder contains the data and logic needed to build an upgrade controller. Don't
// create instance of this type directly, use the NewController function instead.
type ControllerBuilder struct {
	logger    logr.Logger
	namespace string
}

// Coodinator knows how to coordinate the activities needed to perform an upgrade without a
// registry. Don't create instances of this type directly, use the NewController function instead.
type Controller struct {
	logger    logr.Logger
	namespace string
	manager   ctrl.Manager
	client    clnt.Client
	cancel    context.CancelFunc
}

type controllerReconcileTask struct {
	logger    logr.Logger
	client    clnt.Client
	namespace string
	version   *configv1.ClusterVersion
	nodes     []*corev1.Node
}

// NewController creates a builder that can then be used to configure and create a coordiator.
func NewController() *ControllerBuilder {
	return &ControllerBuilder{}
}

// SetLogger sets the logger that the controller will use to write messages to the log. This is
// mandatory.
func (b *ControllerBuilder) SetLogger(value logr.Logger) *ControllerBuilder {
	b.logger = value
	return b
}

// SetNamespace sets the namespace where the controller will create the objects it needs, in
// particular the propagator daemon set. This is mandatory.
func (b *ControllerBuilder) SetNamespace(value string) *ControllerBuilder {
	b.namespace = value
	return b
}

// Build uses the configuration stored in the builder to create a new controller.
func (b *ControllerBuilder) Build() (result *Controller, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.namespace == "" {
		err = errors.New("namespace is mandatory")
		return
	}

	// Creat the scheme and register the types that we will be using:
	scheme := runtime.NewScheme()
	err = core.AddToScheme(scheme)
	if err != nil {
		return
	}
	err = config.Install(scheme)
	if err != nil {
		return
	}

	// Create the controller manager:
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return
	}
	options := ctrl.Options{
		Scheme:                 scheme,
		Logger:                 b.logger,
		Namespace:              b.namespace,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
	}
	manager, err := ctrl.NewManager(cfg, options)
	if err != nil {
		return
	}

	// Create and populate the object:
	controller := &Controller{
		logger:    b.logger,
		namespace: b.namespace,
		manager:   manager,
		client:    manager.GetClient(),
	}

	// Add the controllers:
	_, err = ctrl.NewControllerManagedBy(manager).
		For(&configv1.ClusterVersion{}).
		Build(controller)
	if err != nil {
		return
	}
	_, err = ctrl.NewControllerManagedBy(manager).
		For(&corev1.Node{}).
		Build(controller)
	if err != nil {
		return
	}

	// Return the result:
	result = controller
	return
}

// Start starts the controller and returns inmediately.
func (c *Controller) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	go func() {
		err := c.manager.Start(ctx)
		if err != nil {
			c.logger.Error(
				err,
				"Controller manager finished with error",
			)
		}
	}()
	return nil
}

// Stop stops the controller.
func (c *Controller) Stop(ctx context.Context) error {
	c.cancel()
	return nil
}

func (c *Controller) Reconcile(ctx context.Context, request ctrl.Request) (result ctrl.Result,
	err error) {
	// Fetch the relevant objects:
	version, err := c.fetchVersion(ctx)
	if err != nil {
		return
	}
	nodes, err := c.fetchNodes(ctx)
	if err != nil {
		return
	}

	// Create and execute the task:
	task := &controllerReconcileTask{
		logger:    c.logger,
		client:    c.client,
		namespace: c.namespace,
		version:   version,
		nodes:     nodes,
	}
	err = task.execute(ctx)
	if err != nil {
		return
	}

	return
}

func (c *Controller) fetchVersion(ctx context.Context) (result *configv1.ClusterVersion,
	err error) {
	version := &configv1.ClusterVersion{}
	key := clnt.ObjectKey{
		Name: "version",
	}
	err = c.client.Get(ctx, key, version)
	if err != nil {
		return
	}
	result = version
	return
}

func (c *Controller) fetchNodes(ctx context.Context) (results []*corev1.Node, err error) {
	list := &corev1.NodeList{}
	err = c.client.List(ctx, list)
	if err != nil {
		return
	}
	results = make([]*corev1.Node, len(list.Items))
	for i, item := range list.Items {
		results[i] = item.DeepCopy()
	}
	return
}

func (t *controllerReconcileTask) execute(ctx context.Context) error {
	var err error

	// Don't try to do anything if an upgrade has already been requested:
	if t.upgradeRequested() {
		t.logger.V(1).Info(
			"Upgrade has already been requested",
			"version", t.version.Spec.DesiredUpdate.Version,
			"image", t.version.Spec.DesiredUpdate.Image,
		)
		return nil
	}

	// Don't try to do anything if the bundle hasn't been specified:
	bundleFile := t.stringAnnotation(t.version, annotations.BundleFile)
	if bundleFile == "" {
		t.logger.V(1).Info("Bundle file hasn't been specified yet")
		return nil
	}

	// Classify nodes according to what actions they need:
	var needExtractor, needLoader, needNothing []*corev1.Node
	for _, node := range t.nodes {
		bundleExtracted := t.boolLabel(node, labels.BundleExtracted)
		bundleLoaded := t.boolLabel(node, labels.BundleLoaded)
		if !bundleExtracted {
			needExtractor = append(needExtractor, node)
		}
		if bundleExtracted && !bundleLoaded {
			needLoader = append(needLoader, node)
		}
		if bundleExtracted && bundleLoaded {
			needNothing = append(needNothing, node)
		}
	}

	// If there are nodes that need the bundle extracted then we need to start the bundle server
	// daemon set and the bundle extractor job for each of those nodes.
	if len(needExtractor) > 0 {
		t.logger.Info(
			"Some nodes don't have the bundle extracted yet, will start the bundle "+
				"server and the bundle extractor for those nodes",
			"nodes", t.nodeNames(needExtractor),
		)
		err = t.startBundleServer(ctx, bundleFile)
		if err != nil {
			return err
		}
		for _, node := range needExtractor {
			err = t.startBundleExtractor(ctx, node, bundleFile)
			if err != nil {
				return err
			}
		}
	}

	// If all the nodes have the bundle extracted already then we can stop the bundle server:
	if len(needExtractor) == 0 {
		t.logger.Info("All nodes have the bundle extracted, will stop the bundle server")
		err = t.stopBundleServer(ctx)
		if err != nil {
			return err
		}
	}

	// If there are nodes that need the bundle loaded then we need to start the bundle loader
	// job for them:
	if len(needLoader) > 0 {
		t.logger.Info(
			"Some nodes don't have the bundle loaded yet, will start the bundle "+
				"loader for those nodes",
			"nodes", t.nodeNames(needLoader),
		)
		for _, node := range needLoader {
			err = t.startBundleLoader(ctx, node)
			if err != nil {
				return err
			}
		}
	}

	// If none of the nodes needs an action then we can request the upgrade:
	if len(needNothing) == len(t.nodes) {
		t.logger.Info("All nodes are ready, will request the upgrade")
		err = t.requestUpgrade(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *controllerReconcileTask) startBundleServer(ctx context.Context, bundleFile string) error {
	// Create the service account:
	err := t.createPrivilegedServiceAccount(ctx, bundleServer)
	if err != nil {
		return err
	}

	// Create the service:
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      bundleServer,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Selector: map[string]string{
				labels.App: bundleServer,
			},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	err = t.client.Create(ctx, service)
	switch {
	case err == nil:
		t.logger.Info(
			"Created bundle server service",
			"service", service.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Bundle server service already exists",
			"service", service.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create bundle server service",
			"daemonset", service.Name,
		)
		return err
	}

	// Create the daemon set:
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      bundleServer,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labels.App: bundleServer,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						labels.App: bundleServer,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: bundleServer,
					Volumes: []corev1.Volume{
						t.makeHostVolume(),
					},
					Containers: []corev1.Container{{
						Name:            bundleServer,
						Image:           controllerImage,
						ImagePullPolicy: controllerImagePullPolicy,
						SecurityContext: &corev1.SecurityContext{
							Privileged: pointer.Bool(true),
							RunAsUser:  pointer.Int64(0),
						},
						VolumeMounts: []corev1.VolumeMount{
							t.makeHostMount(),
						},
						Command: []string{
							"/openshif-appliance",
							"start-upgrade-bundle-server",
							"--log-file=stdout",
							"--log-level=debug",
							fmt.Sprintf(
								"--root-dir=%s",
								controllerHostVolumeMountPath,
							),
							fmt.Sprintf(
								"--bundle-file=%s",
								bundleFile,
							),
							"--listen-addr=:8080",
						},
					}},
					Tolerations: t.makeTolerations(),
				},
			},
		},
	}
	err = t.client.Create(ctx, daemonSet)
	switch {
	case err == nil:
		t.logger.Info(
			"Created bundle server daemon set",
			"daemonset", daemonSet.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Bundle server daemon set already exists",
			"daemonset", daemonSet.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create bundle server daemon set",
			"daemonset", daemonSet.Name,
		)
		return err
	}

	return nil
}

func (t *controllerReconcileTask) stopBundleServer(ctx context.Context) error {
	// Delete the service:
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      bundleServer,
		},
	}
	err := t.client.Delete(ctx, service)
	switch {
	case err == nil:
		t.logger.Info(
			"Deleted bundle server service",
			"service", service.Name,
		)
	case apierrors.IsNotFound(err):
		t.logger.V(2).Info(
			"Bundle server service doesn't exist",
			"service", service.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create bundle server service",
			"daemonset", service.Name,
		)
		return err
	}

	// Delete the daemon set:
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      bundleServer,
		},
	}
	err = t.client.Delete(ctx, daemonSet)
	switch {
	case err == nil:
		t.logger.Info(
			"Deleted bundle server daemon set",
			"daemonset", daemonSet.Name,
		)
	case apierrors.IsNotFound(err):
		t.logger.V(2).Info(
			"Bundle server daemon set doesn't exist",
			"daemonset", daemonSet.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to deleted bundle server daemon set",
			"daemonset", daemonSet.Name,
		)
		return err
	}

	// Delete the service account:
	err = t.deletePrivilegedServiceAccount(ctx, bundleServer)
	if err != nil {
		return err
	}

	return nil
}

func (t *controllerReconcileTask) startBundleExtractor(ctx context.Context, node *corev1.Node,
	bundleFile string) error {
	// Create the service account:
	err := t.createPrivilegedServiceAccount(ctx, bundleExtractor)
	if err != nil {
		return err
	}

	// Create the extractor job:
	extractorJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      fmt.Sprintf("%s-%s", bundleExtractor, node.Name),
			Labels: map[string]string{
				labels.Job: bundleExtractor,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName:           node.Name,
					ServiceAccountName: bundleExtractor,
					Volumes: []corev1.Volume{
						t.makeHostVolume(),
					},
					Containers: []corev1.Container{{
						Name:            bundleExtractor,
						Image:           controllerImage,
						ImagePullPolicy: controllerImagePullPolicy,
						SecurityContext: &corev1.SecurityContext{
							Privileged: pointer.Bool(true),
							RunAsUser:  pointer.Int64(0),
						},
						VolumeMounts: []corev1.VolumeMount{
							t.makeHostMount(),
						},
						Command: []string{
							"/openshift-appliance",
							"start-upgrade-bundle-extractor",
							"--log-file=stdout",
							"--log-level=debug",
							fmt.Sprintf(
								"--node-name=%s",
								node.Name,
							),
							fmt.Sprintf(
								"--root-dir=%s",
								controllerHostVolumeMountPath,
							),
							fmt.Sprintf(
								"--bundle-file=%s",
								bundleFile,
							),
							"--bundle-dir=/var/lib/upgrade",
							fmt.Sprintf(
								"--bundle-server=bundle-server.%s.svc.cluster.local:8080",
								t.namespace,
							),
						},
					}},
					Tolerations:   t.makeTolerations(),
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
	err = t.client.Create(ctx, extractorJob)
	switch {
	case err == nil:
		t.logger.Info(
			"Created bundle extractor",
			"node", node.Name,
			"job", extractorJob.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Bundle extractor already exists",
			"node", node.Name,
			"name", extractorJob.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create bundle extractor",
			"node", node.Name,
			"job", extractorJob.Name,
		)
		return err
	}
	return nil
}

func (t *controllerReconcileTask) startBundleLoader(ctx context.Context, node *corev1.Node) error {
	// Create the service account:
	err := t.createPrivilegedServiceAccount(ctx, bundleLoader)
	if err != nil {
		return err
	}

	// Create the loader job:
	loaderJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      fmt.Sprintf("%s-%s", bundleLoader, node.Name),
			Labels: map[string]string{
				labels.Job: bundleLoader,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName:           node.Name,
					ServiceAccountName: bundleLoader,
					Volumes: []corev1.Volume{
						t.makeHostVolume(),
					},
					HostNetwork: true,
					Containers: []corev1.Container{{
						Name:            bundleLoader,
						Image:           controllerImage,
						ImagePullPolicy: controllerImagePullPolicy,
						SecurityContext: &corev1.SecurityContext{
							Privileged: pointer.Bool(true),
							RunAsUser:  pointer.Int64(0),
						},
						VolumeMounts: []corev1.VolumeMount{
							t.makeHostMount(),
						},
						Command: []string{
							"/openshift-appliance",
							"start-upgrade-bundle-loader",
							"--log-file=stdout",
							"--log-level=debug",
							fmt.Sprintf(
								"--node-name=%s",
								node.Name,
							),
							fmt.Sprintf(
								"--root-dir=%s",
								controllerHostVolumeMountPath,
							),
							"--bundle-dir=/var/lib/upgrade",
						},
					}},
					Tolerations:   t.makeTolerations(),
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
	err = t.client.Create(ctx, loaderJob)
	switch {
	case err == nil:
		t.logger.Info(
			"Created bundle loader",
			"node", node.Name,
			"job", loaderJob.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Bundle loader already exists",
			"node", node.Name,
			"name", loaderJob.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create bundle loader",
			"node", node.Name,
			"job", loaderJob.Name,
		)
		return err
	}

	return nil
}

func (t *controllerReconcileTask) makeHostVolume() corev1.Volume {
	directory := corev1.HostPathDirectory
	return corev1.Volume{
		Name: controllerHostVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Type: &directory,
				Path: controllerHostVolumePath,
			},
		},
	}
}

func (t *controllerReconcileTask) makeHostMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      controllerHostVolumeName,
		MountPath: controllerHostVolumeMountPath,
	}
}

func (t *controllerReconcileTask) makeTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		{
			Key:      "node-role.kubernetes.io/master",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
}

func (t *controllerReconcileTask) createPrivilegedServiceAccount(ctx context.Context,
	name string) error {
	// Create the service account:
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name,
		},
	}
	err := t.client.Create(ctx, serviceAccount)
	switch {
	case err == nil:
		t.logger.Info(
			"Created service account",
			"name", serviceAccount.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Service account already exists",
			"name", serviceAccount.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create service account",
			"name", serviceAccount.Name,
		)
		return err
	}

	// Create the admin role binding:
	adminBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-cluster-admin", t.namespace, name),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: t.namespace,
			Name:      name,
		}},
	}
	err = t.client.Create(ctx, adminBinding)
	switch {
	case err == nil:
		t.logger.Info(
			"Created admin cluster role binding",
			"name", adminBinding.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Admin cluster role binding already exists",
			"name", adminBinding.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create admin cluster role binding",
			"name", adminBinding.Name,
		)
		return err
	}

	// Create the privileged role binding:
	privilegedBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      fmt.Sprintf("%s-privileged", name),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:openshift:scc:privileged",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: t.namespace,
			Name:      name,
		}},
	}
	err = t.client.Create(ctx, privilegedBinding)
	switch {
	case err == nil:
		t.logger.Info(
			"Created privileged role binding",
			"name", privilegedBinding.Name,
		)
	case apierrors.IsAlreadyExists(err):
		t.logger.V(2).Info(
			"Privileged role binding already exists",
			"name", privilegedBinding.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to create privileged role binding",
			"name", privilegedBinding.Name,
		)
		return err
	}

	return nil
}

func (t *controllerReconcileTask) deletePrivilegedServiceAccount(ctx context.Context,
	name string) error {
	// Create the service account:
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name,
		},
	}
	err := t.client.Delete(ctx, serviceAccount)
	switch {
	case err == nil:
		t.logger.Info(
			"Deleted service account",
			"name", serviceAccount.Name,
		)
	case apierrors.IsNotFound(err):
		t.logger.V(2).Info(
			"Service account doesn't exist",
			"name", serviceAccount.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to delete service account",
			"name", serviceAccount.Name,
		)
		return err
	}

	// Delete the admin role binding:
	adminBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-cluster-admin", t.namespace, name),
		},
	}
	err = t.client.Delete(ctx, adminBinding)
	switch {
	case err == nil:
		t.logger.Info(
			"Deleted admin cluster role binding",
			"name", adminBinding.Name,
		)
	case apierrors.IsNotFound(err):
		t.logger.V(2).Info(
			"Admin cluster role binding doesn't exist",
			"name", adminBinding.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to delete admin cluster role binding",
			"name", adminBinding.Name,
		)
		return err
	}

	// Delete the privileged role binding:
	privilegedBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      fmt.Sprintf("%s-privileged", name),
		},
	}
	err = t.client.Delete(ctx, privilegedBinding)
	switch {
	case err == nil:
		t.logger.Info(
			"Deleted privileged role binding",
			"name", privilegedBinding.Name,
		)
	case apierrors.IsNotFound(err):
		t.logger.V(2).Info(
			"Privileged role binding doesn't exist",
			"name", privilegedBinding.Name,
		)
	default:
		t.logger.Error(
			err,
			"Failed to delete privileged role binding",
			"name", privilegedBinding.Name,
		)
		return err
	}

	return nil
}

func (t *controllerReconcileTask) upgradeRequested() bool {
	desiredUpdate := t.version.Spec.DesiredUpdate
	return desiredUpdate != nil && (desiredUpdate.Version != "" || desiredUpdate.Image != "")
}

func (t *controllerReconcileTask) requestUpgrade(ctx context.Context) error {
	var err error

	// Get the bundle metadata from the first node that has it:
	if len(t.nodes) == 0 {
		return errors.New("there are no nodes")
	}
	var metadata *Metadata
	for _, node := range t.nodes {
		metadata, err = t.readMetadata(node)
		if err != nil {
			return err
		}
		if metadata != nil {
			break
		}
	}
	if metadata == nil {
		return errors.New("no node has metadata")
	}

	// Request the upgrade:
	versionUpdate := t.version.DeepCopy()
	versionUpdate.Spec.DesiredUpdate = &configv1.Update{
		Image: metadata.Release,
		Force: true,
	}
	versionPatch := clnt.MergeFrom(t.version)
	err = t.client.Patch(ctx, versionUpdate, versionPatch)
	if err != nil {
		return err
	}
	t.logger.Info(
		"Requested upgrade",
		"version", metadata.Version,
		"image", metadata.Release,
	)

	return nil
}

func (t *controllerReconcileTask) readMetadata(node *corev1.Node) (metadata *Metadata, err error) {
	value := t.stringAnnotation(node, annotations.BundleMetadata)
	if value == "" {
		return
	}
	err = json.Unmarshal([]byte(value), &metadata)
	if err != nil {
		return
	}
	t.logger.Info(
		"Read metadata",
		"node", node.Name,
		"version", metadata.Version,
		"arch", metadata.Arch,
		"release", metadata.Release,
	)
	return
}

func (c *controllerReconcileTask) boolLabel(object clnt.Object, label string) bool {
	values := object.GetLabels()
	if values == nil {
		return false
	}
	value, ok := values[label]
	if !ok {
		return false
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		c.logger.Error(
			err,
			"Invalid value for boolean label, will return false",
			"label", label,
			"value", value,
		)
		return false
	}
	return result
}

func (t *controllerReconcileTask) stringAnnotation(object clnt.Object, name string) string {
	values := object.GetAnnotations()
	if values == nil {
		return ""
	}
	value, ok := values[name]
	if !ok {
		return ""
	}
	return value
}

func (t *controllerReconcileTask) nodeNames(nodes []*corev1.Node) []string {
	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name
	}
	slices.Sort(names)
	return names
}

const (
	controllerHostVolumeName      = "host"
	controllerHostVolumePath      = "/"
	controllerHostVolumeMountPath = "/host"

	controllerImage           = "quay.io/edge-infrastructure/openshift-appliance:latest"
	controllerImagePullPolicy = corev1.PullIfNotPresent

	bundleCleaner   = "bundle-cleaner"
	bundleExtractor = "bundle-extractor"
	bundleLoader    = "bundle-loader"
	bundleServer    = "bundle-server"
)
