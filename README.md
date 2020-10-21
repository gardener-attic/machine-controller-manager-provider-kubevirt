# machine-controller-manager-provider-kubevirt

This project contains the external [Machine Controller Manager](https://github.com/gardener/machine-controller-manager) plugin (driver) implementation for the [KubeVirt](https://kubevirt.io) provider. It is intended to be used in combination with the [Gardener Extension for KubeVirt provider](https://github.com/gardener/gardener-extension-provider-kubevirt).

For more information about Gardener integration with KubeVirt see [this gardener.cloud blog post](https://gardener.cloud/blog/2020-10/00/). 

## Prerequisites

* A provider cluster with [KubeVirt](https://kubevirt.io) and [CDI](https://github.com/kubevirt/containerized-data-importer) installed, and a user with read and write permissions on KubeVirt, CDI, and Kubernetes core resources in a certain namespace of this cluster. 
* To take advantage of networking features, the provider cluster should also contain [Multus](https://intel.github.io/multus-cni/doc/quickstart.html).

## Supported KubeVirt versions

This plugin has been tested with KubeVirt v0.32.0 and CDI v1.23.5.

## How to start using or developing this extension locally

You can run the extension locally on your machine by executing `make start`.

Static code checks and tests can be executed by running `make verify`. We are using Go modules for Golang package dependency management and [Ginkgo](https://github.com/onsi/ginkgo)/[Gomega](https://github.com/onsi/gomega) for testing.

## Feedback and Support

Feedback and contributions are always welcome. Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extension-provider-kubevirt/issues) or join our [Slack channel #gardener](https://kubernetes.slack.com/messages/gardener) (please invite yourself to the Kubernetes workspace [here](http://slack.k8s.io)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* ["Gardener Project Update" blog on kubernetes.io](https://kubernetes.io/blog/2019/12/02/gardener-project-update/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)
* [GEP-4 (New `core.gardener.cloud/v1alpha1` API)](https://github.com/gardener/gardener/blob/master/docs/proposals/04-new-core-gardener-cloud-apis.md)
* [Extensibility API documentation](https://github.com/gardener/gardener/tree/master/docs/extensions)
* [Gardener Extensions Golang library](https://godoc.org/github.com/gardener/gardener/extensions/pkg)
* [Gardener API Reference](https://gardener.cloud/api-reference/)
