module github.com/gardener/machine-controller-manager-provider-kubevirt

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/gardener/machine-controller-manager v0.33.0
	github.com/golang/mock v1.4.4-0.20200731163441-8734ec565a4d
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/spf13/pflag v1.0.5
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/component-base v0.18.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
	kubevirt.io/client-go v0.33.0
	kubevirt.io/containerized-data-importer v1.10.6
	sigs.k8s.io/controller-runtime v0.5.5
)

replace (
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.17.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.9
	k8s.io/apiserver => k8s.io/apiserver v0.17.9
	k8s.io/client-go => k8s.io/client-go v0.17.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.9
	k8s.io/code-generator => k8s.io/code-generator v0.17.9
)
