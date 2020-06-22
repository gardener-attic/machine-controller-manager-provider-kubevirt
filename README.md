# machine-controller-manager-provider-kubevirt
Out of tree (controller based) implementation for `kubevirt` as a new provider.

## About
- This is a sample repository that provides the blueprint required to implement a new (hyperscale) provider. We call the new provider as `kubevirt` for our ease.
- Each provider implements the interface defined at [MCM OOT driver](https://github.com/gardener/machine-controller-manager/blob/master/pkg/util/provider/driver/driver.go).

## Fundamental Design Principles:
Following are the basic principles kept in mind while developing the external plugin.
* Communication between this Machine Controller (MC) and Machine Controller Manager (MCM) is achieved using the Kubernetes native declarative approach.
* Machine Controller (MC) behaves as the controller used to interact with the cloud provider and manage the VMs corresponding to the machine objects.
* Machine Controller Manager (MCM) deals with higher level objects such as machine-set and machine-deployment objects.

## Support for a new provider
- Steps to be followed while implementing a new provider are mentioned [here](https://github.com/gardener/machine-controller-manager/blob/master/docs/development/cp_support_new.md)
