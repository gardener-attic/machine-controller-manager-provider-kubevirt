package core

import (
	"context"
	"errors"
	"fmt"

	clouderrors "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"

	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataVolumeManager manages Kubevirt DataVolume operations.
type DataVolumeManager interface {
	CreateVolume(ctx context.Context, name, namespace string, dvSpec cdi.DataVolumeSpec) error
	GetVolume(ctx context.Context, name, namespace string) (*cdi.DataVolume, error)
	ListDataVolumes(ctx context.Context, namespace string) (*cdi.DataVolumeList, error)
	DeleteVolume(ctx context.Context, name, namespace string) error
}

type defaultDataVolumeManager struct {
	c client.Client
}

// NewDefaultDataVolumeManager creates a new default manager with a k8s controller-runtime based on the sent kubecomfig.
func NewDefaultDataVolumeManager(kubeconfig string) (DataVolumeManager, error) {
	if kubeconfig == "" {
		return nil, errors.New("kubevirt cluster kubeconfig cannot be empty")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
	}

	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubevirt client: %v", err)
	}

	return &defaultDataVolumeManager{
		c: c,
	}, nil
}

func (d *defaultDataVolumeManager) CreateVolume(ctx context.Context, name, namespace string, dvSpec cdi.DataVolumeSpec) error {
	dataVolume := &cdi.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: dvSpec,
	}

	return d.c.Create(ctx, dataVolume)
}

func (d *defaultDataVolumeManager) GetVolume(ctx context.Context, name, namespace string) (*cdi.DataVolume, error) {
	dataVolume := &cdi.DataVolume{}
	if err := d.c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, dataVolume); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, &clouderrors.DataVolumeError{
				Name:      name,
				Namespace: namespace,
			}
		}

		return nil, fmt.Errorf("failed to get DataVolume: %s: %v", name, err)
	}

	return dataVolume, nil
}

func (d *defaultDataVolumeManager) ListDataVolumes(ctx context.Context, namespace string) (*cdi.DataVolumeList, error) {
	dvList := cdi.DataVolumeList{}
	if err := d.c.List(ctx, &dvList); err != nil {
		return nil, fmt.Errorf("failed to list DataVolumes in namespace %s: %v", namespace, err)
	}

	if len(dvList.Items) == 0 {
		klog.V(2).Infof("namespace %s has no data volumes", namespace)
		return nil, nil
	}

	return &dvList, nil
}

func (d *defaultDataVolumeManager) DeleteVolume(ctx context.Context, name, namespace string) error {
	dv, err := d.GetVolume(ctx, name, namespace)
	if err != nil {
		if clouderrors.IsDataVolumeError(err) {
			klog.Warningf("data volume %s in namespace %v is not found", name, namespace)
			return nil
		}

		return fmt.Errorf("failed to fetch data volume prior creation: %v", err)
	}

	return d.c.Delete(ctx, dv)
}
