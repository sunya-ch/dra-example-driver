# KEP 5075: Consumable Capacity - PoC

Remarks:
* Require git clone and checkout `consumable-capacity-poc` branch from this forked repo: https://github.com/sunya-ch/dra-example-driver.git. 
  There are some additional changes required in script and kind configuration.
* All commands must be run from repo root path.

## Prepare kind cluster with PoC node image and Install PoC DRA driver

1. Prepare kind cluster

    ```sh
    KIND_IMAGE=ghcr.io/sunya-ch/kindest/node:kep-5075 ./demo/create-cluster.sh
    ```

2. Load DRA driver image to kind cluster

    ```sh
    DRIVER_IMAGE=ghcr.io/sunya-ch/dra-example-driver:kep-5075 ./demo/scripts/load-driver-image-into-kind.sh
    ```

3. Install DRA driver

    ```sh
    helm upgrade -i \
    --create-namespace \
    --namespace dra-example-driver \
    dra-example-driver \
    deployments/helm/dra-example-driver
    ```

The shared devices should present in resource slice.
- nic-1: shared device without consumable capacity
- qos-nic: shared device with consumable capacity

```bash
kubectl get resourceslices -oyaml
```

Expected devices: 

```yaml
spec:
  devices:
  - basic:
      attributes:
        driverVersion:
          version: 1.0.0
        index:
          int: 0
        model:
          string: LATEST-QOS-NIC-MODEL
        uuid:
          string: qos-nic-18db0e85-99e9-c746-8531-ffeb86328b39
      capacity:
        bandwidth:
          claimPolicy:
            range:
              minimum: "1"
          value: 10Gi
      shared: true
    name: qos-nic-0
  - basic:
      attributes:
        driverVersion:
          version: 1.0.0
        index:
          int: 0
        model:
          string: LATEST-NIC-MODEL
        uuid:
          string: nic-18db0e85-99e9-c746-8531-ffeb86328b39
      capacity:
        bandwidth:
          value: 10Gi
      shared: true
    name: nic-0
```

## Test

1. Deploy all [device classes](./device-class.yaml).

    ```bash
    kubectl create -f ./demo/kep-5075/device-class.yaml
    ```

2. The [first test](./resource-claim/shared-device.yaml) is to deploy two pods, each requests the `LATEST-NIC-MODEL` device.

    ```bash
    kubectl create -f ./demo/kep-5075/resource-claim/shared-device.yaml
    ```

    Check the resourceclaim:

    ```bash
    kubectl get resourceclaim -n shared-device1 -oyaml
    ```

    Expected results for both pods:

    ```yaml
        status:
            allocation:
            devices:
                results:
                - adminAccess: null
                device: nic-0
                driver: gpu.example.com
                pool: dra-example-driver-cluster-worker
                request: nic
                shared: true
            nodeSelector:
                nodeSelectorTerms:
                - matchFields:
                - key: metadata.name
                    operator: In
                    values:
                    - dra-example-driver-cluster-worker
    ```

3. The second test is to deploy three pods with 5Gi bandwidth request per each for the `LATEST-QOS-NIC-MODEL` device.

    ```bash
    kubectl create -f ./demo/kep-5075/resource-claim/qos-aware-shared-device.yaml
    ```

    Check the resourceclaim:

    ```bash
    kubectl get resourceclaim -n shared-device1 -oyaml
    ```

    Expected pods' status (2 pods with 5Gi allocated, while the third pod must be pending): 

    ```bash
    > kubectl get po -n shared-device2
    NAME   READY   STATUS    RESTARTS   AGE
    pod0   1/1     Running   0          2m27s
    pod1   1/1     Running   0          2m27s
    pod2   0/1     Pending   0          2m27s
    ```

    Expected resource claims's allocation results of two running pods:
    ```yaml
    status:
      allocation:
        devices:
            results:
            - adminAccess: null
            consumedCapacities:
                bandwidth: 5Gi
            device: qos-nic-0
            driver: gpu.example.com
            pool: dra-example-driver-cluster-worker
            request: nic
            shared: true
        nodeSelector:
            nodeSelectorTerms:
            - matchFields:
            - key: metadata.name
                operator: In
                values:
                - dra-example-driver-cluster-worker
    ```

    Once one pod is released, the third pod should be able to run.

    ```bash
    > kubectl get po -n shared-device2
    NAME   READY   STATUS    RESTARTS   AGE
    pod1   1/1     Running   0          5m59s
    pod2   1/1     Running   0          5m59s
    ```