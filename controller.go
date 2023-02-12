package main

import (
	clientset "controller-crd/pkg/generated/clientset/versioned"
	"controller-crd/pkg/generated/clientset/versioned/scheme"
	groupkindscheme "controller-crd/pkg/generated/clientset/versioned/scheme"
	"controller-crd/pkg/generated/informers/externalversions/groupkind/v1alpha1"
	alpha1 "controller-crd/pkg/generated/listers/groupkind/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	foosLister         alpha1.FooLister
	foosSynced         func() bool
}

func (c Controller) enqueueApp(obj interface{}) {

}

const controllerAgentName = "controller-crd"

func NewController(
	kubeclientset kubernetes.Interface,
	groupkindClientset clientset.Interface,
	depoymentInformer v12.DeploymentInformer,
	serviceInformer v13.ServiceInformer,
	ingressInformer v14.IngressInformer,
	groupkindInformer v1alpha1.FooInformer) *Controller {

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
