package agent

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/apis/resaware/v1alpha1"
)

type configKind int

type configIf struct {
	kind configKind
	cfg  *rest.Config
	cli  *dynamic.DynamicClient
}

func newConfigIf(kind configKind) *configIf {
	return &configIf{
		kind: kind,
	}
}

func (cif *configIf) SetKubeClient(cfg *rest.Config) error {
	cif.cfg = cfg
	cli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client kuebclient: %v", err)
	}
	cif.cli = cli
	return nil
}

func (cif *configIf) CreateWatch(ctx context.Context, ns, name string, gvr schema.GroupVersionResource) (watch.Interface, error) {
	selector := metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	}

	return cif.cli.Resource(gvr).Namespace(ns).Watch(ctx, selector)
}

func (cif *configIf) GetConfigCrd(ctx context.Context, ns, name string, gvr schema.GroupVersionResource) (*unstructured.Unstructured, error) {
	un, err := cif.cli.Resource(gvr).Namespace(ns).Get(ctx, name, v1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("get configed crd error: %v", err)
	}

	return un, err
}

func (cif *configIf) UpdateConfigCrd(ctx context.Context, ns string, un *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	updated, err := cif.cli.Resource(schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "oenumas",
	}).Namespace(ns).Update(context.Background(), un, v1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update config crd error%v", err)
	}
	return updated, nil
}

func (cif *configIf) CreateConfigCrd(ctx context.Context, name, ns string) error {
	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "oenumas",
	}

	_, err := cif.cli.Resource(gvr).Namespace(ns).Get(ctx, name, v1.GetOptions{})
	if err == nil {
		return fmt.Errorf("the target crd: oenuma already exists.")
	} else {
		klog.Info("the target crd not exists, automatically create crd: oenuma")
	}

	crd := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "resource.sop.huawei.com/v1alpha1",
			"kind":       "Oenuma",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": ns,
			},
			"spec": map[string]interface{}{
				"name":         "testapp1",
				"replicas":     1,
				"updateenable": "enable",
			},
		},
	}

	_, err = cif.cli.Resource(gvr).Namespace(ns).Create(ctx, crd, v1.CreateOptions{})
	if err != nil {
		klog.Error(err)
		return fmt.Errorf("unable to automatically create crd: oenuma, please try to creating it manually.")
	}
	return nil
}
