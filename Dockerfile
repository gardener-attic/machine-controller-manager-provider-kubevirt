#############      builder    #############
FROM golang:1.13.5 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-kubevirt
COPY . .

RUN .ci/build

#############      base       #############
FROM alpine:3.11.2 as base

RUN apk add --update bash curl tzdata
WORKDIR /

#############      machine-controller      #############
FROM base AS machine-controller

COPY --from=builder /go/src/github.com/gardener/machine-controller-manager-provider-kubevirt/bin/rel/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
