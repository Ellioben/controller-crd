package pkg

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Create a Deployment in a namespace ns.
func createDeployment(ctx context.Context, config *rest.Config, ns string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	deployment.Name = "example"
	// edit deployment spec

	client := clientset.AppsV1().Deployments(ns)
	_, err = client.Create(ctx, deployment, metav1.CreateOptions{})
	return err
}
