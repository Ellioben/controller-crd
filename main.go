package main

import (
	clientset "controller-crd/pkg/generated/clientset/versioned"
	"controller-crd/pkg/generated/informers/externalversions"
	"k8s.io/client-go/informers"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"
)

func main() {

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
	groupKindInformerFactory := externalversions.NewSharedInformerFactory(groupKindClient, time.Second*30)

	controller := NewController(kubeClient, groupKindClient,
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Core().V1().Services(),
		kubeInformerFactory.Networking().V1().Ingresses(),
		groupKindInformerFactory.Groupkind().V1alpha1().Foos())

}
