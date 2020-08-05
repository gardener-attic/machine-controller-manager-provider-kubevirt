package core

import (
	"context"
	"testing"

	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	dvTestCase = struct {
		name                string
		dataVolumeName      string
		dataVolumeNamespace string
		dataVolumeSpec      cdi.DataVolumeSpec
	}{
		name:                "kubevirt data volume test",
		dataVolumeName:      "test-data-volume",
		dataVolumeNamespace: "test-namespace",
		dataVolumeSpec: cdi.DataVolumeSpec{
			PVC: &v1.PersistentVolumeClaimSpec{
				StorageClassName: utilpointer.StringPtr("test-sc"),
				AccessModes: []v1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
			},
			Source: cdi.DataVolumeSource{
				HTTP: &cdi.DataVolumeSourceHTTP{
					URL: "https://test-cloud.img",
				},
			},
		},
	}
)

func TestDefaultDataVolumeManager_CreateVolume(t *testing.T) {
	t.Run("test data volume creation", func(t *testing.T) {
		pvcSize, err := resource.ParseQuantity("10Gi")
		if err != nil {
			t.Fatal(err)
		}

		dvTestCase.dataVolumeSpec.PVC.Resources.Requests = v1.ResourceList{v1.ResourceStorage: pvcSize}

		addKnownTypes(scheme.Scheme)
		manager := &defaultDataVolumeManager{
			c: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		if err := manager.CreateVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace, dvTestCase.dataVolumeSpec); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDefaultDataVolumeManager_GetVolume(t *testing.T) {
	t.Run("test data volume fetching", func(t *testing.T) {
		pvcSize, err := resource.ParseQuantity("10Gi")
		if err != nil {
			t.Fatal(err)
		}

		dvTestCase.dataVolumeSpec.PVC.Resources.Requests = v1.ResourceList{v1.ResourceStorage: pvcSize}

		addKnownTypes(scheme.Scheme)
		manager := &defaultDataVolumeManager{
			c: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		if err := manager.CreateVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace, dvTestCase.dataVolumeSpec); err != nil {
			t.Fatal(err)
		}

		dv, err := manager.GetVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace)
		if err != nil {
			t.Fatal(err)
		}

		if dv.Name != dvTestCase.dataVolumeName {
			t.Fatalf("failed to fetch data volume, unexcpeted name: %s", dv.Name)
		}
	})
}

func TestDefaultDataVolumeManager_ListDataVolumes(t *testing.T) {
	t.Run("list data volumes", func(t *testing.T) {
		pvcSize, err := resource.ParseQuantity("10Gi")
		if err != nil {
			t.Fatal(err)
		}

		dvTestCase.dataVolumeSpec.PVC.Resources.Requests = v1.ResourceList{v1.ResourceStorage: pvcSize}

		addKnownTypes(scheme.Scheme)
		manager := &defaultDataVolumeManager{
			c: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		if err := manager.CreateVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace, dvTestCase.dataVolumeSpec); err != nil {
			t.Fatal(err)
		}

		dvs, err := manager.ListDataVolumes(context.Background(), dvTestCase.dataVolumeNamespace)
		if err != nil {
			t.Fatal(err)
		}

		if dvs == nil || len(dvs.Items) == 0 {
			t.Fatal("unexpected data volumes count, data volume should be 1")
		}
	})
}

func TestDefaultDataVolumeManager_DeleteVolume(t *testing.T) {
	pvcSize, err := resource.ParseQuantity("10Gi")
	if err != nil {
		t.Fatal(err)
	}

	dvTestCase.dataVolumeSpec.PVC.Resources.Requests = v1.ResourceList{v1.ResourceStorage: pvcSize}

	addKnownTypes(scheme.Scheme)
	manager := &defaultDataVolumeManager{
		c: fake.NewFakeClientWithScheme(scheme.Scheme),
	}

	t.Run("test non-existing data volume delete", func(t *testing.T) {
		if err := manager.DeleteVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test data volume deletion", func(t *testing.T) {
		if err := manager.CreateVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace, dvTestCase.dataVolumeSpec); err != nil {
			t.Fatal(err)
		}

		if err := manager.DeleteVolume(context.Background(), dvTestCase.dataVolumeName, dvTestCase.dataVolumeNamespace); err != nil {
			t.Fatal(err)
		}
	})
}

func addKnownTypes(scheme *runtime.Scheme) {
	SchemeGroupVersion := schema.GroupVersion{Group: "v1alpha1", Version: "cdi.kubevirt.io"}

	scheme.AddKnownTypes(SchemeGroupVersion,
		&cdi.DataVolume{},
		&cdi.DataVolumeList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
}
