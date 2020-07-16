# Wait

Wait is a container that blocks until Kubernetes pods with expected labels are
Ready on the cluster or a timeout is reached. If the timeout is reached, the
container will exit with a non-zero code. If the pods are ready before the
timeout, wait will exit with a code of zero.

This is intended to be used as an init container for order-dependent pods. For
example, consider the C++ gRPC driver. As soon as it runs in a pod, it attempts
to connect the servers and clients. It may exponentially wait or terminate if
these are unreachable. Wait can be set as its init container, requiring servers
and clients to be healthy before it runs.

The entrypoint runs a program that accepts flags of the form:

```
./wait -podLabels loadtest=my-test,loadtest-role=server \
    -podLabels loadtest=my-test,loadtest-role=client \
    -podLabels loadtest=my-test,loadtest-role=client \
    -timeout=5m
```

In this example, the wait container will wait until it encounters three pods
which meet the following conditions:

1. 1 pod with a "loadtest" label of "my-test" and "loadtest-role" of "server"
2. 2 pods with a "loadtest" label of "my-test" and "loadtest-role" of "client"

Wait does not support OR or XOR conditions.
