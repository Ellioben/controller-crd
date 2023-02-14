package main

import (
	"context"
	groupkindv1alpha1 "controller-crd/pkg/apis/groupkind/v1alpha1"
	clientset "controller-crd/pkg/generated/clientset/versioned"
	"controller-crd/pkg/generated/clientset/versioned/scheme"
	groupkindscheme "controller-crd/pkg/generated/clientset/versioned/scheme"
	groupkindinformer "controller-crd/pkg/generated/informers/externalversions/groupkind/v1alpha1"
	groupkindlister "controller-crd/pkg/generated/listers/groupkind/v1alpha1"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v12 "k8s.io/client-go/informers/apps/v1"
	v13 "k8s.io/client-go/informers/core/v1"
	v14 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v15 "k8s.io/client-go/listers/apps/v1"
	v16 "k8s.io/client-go/listers/core/v1"
	v17 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

//常量
const (
	// SuccessSynced is used as part of the Event 'reason' when a App is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a App fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by App"
	// MessageResourceSynced is the message used for an Event fired when a App
	// is synced successfully
	MessageResourceSynced = "App synced successfully"
)

// Controller is the controller implementation for App resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// appclientset is a clientset for our own API group
	groupkindclientset clientset.Interface
	deploymentsSynced  cache.InformerSynced
	serviceSynced      cache.InformerSynced
	ingressSynced      cache.InformerSynced
	appsSynced         cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder           record.EventRecorder
	groupkindClientset clientset.Interface
	deploymentsLister  v15.DeploymentLister
	serviceLister      v16.ServiceLister
	ingressLister      v17.IngressLister
	foosLister         groupkindlister.FooLister
	foosSynced         func() bool
}

const controllerAgentName = "controller-crd"

func NewController(
	kubeclientset kubernetes.Interface,
	groupkindClientset clientset.Interface,
	depoymentInformer v12.DeploymentInformer,
	serviceInformer v13.ServiceInformer,
	ingressInformer v14.IngressInformer,
	groupkindInformer groupkindinformer.FooInformer) *Controller {

	// Create event broadcaster
	// Add app-controller types to the default Kubernetes Scheme so Events can be
	// logged for app-controller types.
	utilruntime.Must(groupkindscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:      kubeclientset,
		groupkindClientset: groupkindClientset,
		deploymentsLister:  depoymentInformer.Lister(),
		deploymentsSynced:  depoymentInformer.Informer().HasSynced,
		serviceLister:      serviceInformer.Lister(),
		serviceSynced:      serviceInformer.Informer().HasSynced,
		ingressLister:      ingressInformer.Lister(),
		ingressSynced:      ingressInformer.Informer().HasSynced,
		foosLister:         groupkindInformer.Lister(),
		foosSynced:         groupkindInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Apps"),
		recorder:           recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when App resources change
	groupkindInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueApp,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueApp(new)
		},
	})

	return controller
}

//
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting App controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	//wait 这些资源再list里都同步完成
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.appsSynced, c.serviceSynced, c.ingressSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process App resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	//持续循环这个方法，监听work
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// App resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the App resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the App resource with this namespace/name
	foo, err := c.foosLister.Foos(namespace).Get(name)
	if err != nil {
		// The App resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("app '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	deployment, err := c.deploymentsLister.Deployments(foo.Namespace).Get(foo.Spec.Deployment.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(foo.Namespace).Create(context.TODO(), newDeployment(foo), metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}

	service, err := c.serviceLister.Services(foo.Namespace).Get(foo.Spec.Service.Name)
	if errors.IsNotFound(err) {
		service, err = c.kubeclientset.CoreV1().Services(foo.Namespace).Create(context.TODO(), newService(foo), metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}

	ingress, err := c.ingressLister.Ingresses(foo.Namespace).Get(foo.Spec.Ingress.Name)
	if errors.IsNotFound(err) {
		ingress, err = c.kubeclientset.NetworkingV1().Ingresses(foo.Namespace).Create(context.TODO(), newIngress(foo), metav1.CreateOptions{})
	}

	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this App resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(deployment, foo) {
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(foo, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}
	if !metav1.IsControlledBy(service, foo) {
		msg := fmt.Sprintf(MessageResourceExists, service.Name)
		c.recorder.Event(foo, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}
	if !metav1.IsControlledBy(ingress, foo) {
		msg := fmt.Sprintf(MessageResourceExists, ingress.Name)
		c.recorder.Event(foo, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}

	c.recorder.Event(foo, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueApp takes a App resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than App.
func (c *Controller) enqueueApp(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// newDeployment creates a new Deployment for a App resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the App resource that 'owns' it.
func newDeployment(foo *groupkindv1alpha1.Foo) *appsv1.Deployment {
	labels := map[string]string{
		"foo":        "kindgroup",
		"controller": foo.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Spec.Deployment.Name,
			Namespace: foo.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(foo, groupkindv1alpha1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &foo.Spec.Deployment.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  foo.Spec.Deployment.Name,
							Image: foo.Spec.Deployment.Image,
						},
					},
				},
			},
		},
	}
}

func newService(foo *groupkindv1alpha1.Foo) *corev1.Service {
	labels := map[string]string{
		"app":        "app-deployment",
		"controller": foo.Name,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Spec.Deployment.Name,
			Namespace: foo.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(foo, groupkindv1alpha1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
				},
			},
		},
	}
}

func newIngress(foo *groupkindv1alpha1.Foo) *v1.Ingress {
	pathType := v1.PathTypePrefix
	return &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Spec.Deployment.Name,
			Namespace: foo.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(foo, groupkindv1alpha1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: foo.Spec.Service.Name,
											Port: v1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
