package main

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

const deploymentYAML = `
		apiVersion: apps/v1
		kind: Deployment
		metadata:
		  name: nginx-deployment
		  namespace: default
		spec:
		  selector:
			matchLabels:
			  app: nginx
		  template:
			metadata:
			  labels:
				app: nginx
			spec:
			  containers:
			  - name: nginx
				image: nginx:latest
		`

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

// 这段代码是一个使用 dynamic client 操作 Kubernetes API Server 的示例。它的主要功能是将一个 Deployment 对象的 YAML 格式的定义转换为 unstructured.Unstructured 对象，然后使用 dynamic client 创建或更新该对象。

// 这段代码的具体实现过程如下：

// 创建一个 RESTMapper，用于查找 GroupVersionResource（GVR）。
// 创建一个 dynamic client。
// 将 YAML 格式的 Deployment 定义解码为 unstructured.Unstructured 对象。
// 使用 RESTMapper 查找 GVR。
// 获取 GVR 的 REST 接口。
// 将对象序列化为 JSON。
// 使用 dynamic client 创建或更新对象。
// 需要注意的是，这段代码使用了 SSA（Server-Side Apply）机制来创建或更新对象。SSA 是 Kubernetes 1.16 引入的新特性，它可以在不破坏已有字段的情况下，对对象进行部分更新。在这段代码中，FieldManager 字段指定了该对象的所有者 ID。
func doSSA(ctx context.Context, cfg *rest.Config) error {

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Prepare the dynamic client
	dyclient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	// 3. Decode YAML manifest into unstructured.Unstructured
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(deploymentYAML), nil, obj)
	if err != nil {
		return err
	}

	// 4. Find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// 5. Obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dyclient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = dyclient.Resource(mapping.Resource)
	}

	// 6. Marshal object into JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// 7. Create or Update the object with SSA
	//     types.ApplyPatchType indicates SSA.
	//     FieldManager specifies the field owner ID.
	_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "sample-controller",
	})

	return err
}
