package main

import (
	clientset "controller-crd/pkg/generated/clientset/versioned"
	groupkindinformers_externalversions "controller-crd/pkg/generated/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var onlyOneSignalHandler = make(chan struct{})
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func main() {

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := SetupSignalHandler()
	//cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	cfg, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	//操作内嵌资源，so是clientset
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	// crd 的clientset
	groupKindClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building app clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	groupKindInformerFactory := groupkindinformers_externalversions.NewSharedInformerFactory(groupKindClient, time.Second*30)

	controller := NewController(kubeClient, groupKindClient,
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Core().V1().Services(),
		kubeInformerFactory.Networking().V1().Ingresses(),
		groupKindInformerFactory.Groupkind().V1alpha1().Foos())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	groupKindInformerFactory.Start(stopCh)
	//controller运行后，就是从队列里面开始拿数据了。
	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
