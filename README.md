# Volcano Device Plugin Mock

This project, `deviceplugin-mock`, is a component of the [Volcano](https://volcano.sh/) ecosystem. It is designed to mock device plugins in a Kubernetes cluster, allowing for the testing and development of schedulers and other components that rely on device resources without needing actual hardware.

## Overview

The project consists of two main components:

- **Controller**: Manages the lifecycle and configuration of the mock devices.
- **Daemon**: Runs on nodes and interacts with the Kubelet to register mock devices.

## Getting Started

### Prerequisites

- Go 1.24.0+
- Docker (or another container runtime)
- Kubernetes cluster

### Building

You can build the binaries and images using the provided `Makefile`.

#### Build Binaries

To build the controller and daemon binaries locally:

```sh
make build
```

The binaries will be placed in `_output/bin/`.

#### Build Images

To build the Docker images:

```sh
make images
```

You can customize the image registry and name using environment variables:

```sh
make images IMAGE_REGISTRY=myregistry.local IMAGE_NAME=my/mocker TAG=v1.0.0
```

### Deployment

#### Using Helm

A Helm chart is provided in `installer/charts/volcano-deviceplugin-mock`.

1. **Package the Helm chart:**

    ```sh
    make helm-package
    ```

    The chart will be saved to `_output/charts`.

2. **Install the chart:**

    ```sh
    helm install deviceplugin-mock ./installer/charts/volcano-deviceplugin-mock -n volcano-system --create-namespace
    ```

#### Manual Deployment

The `Makefile` also provides targets for managing manifests:

- **Generate Manifests:**
  
  ```sh
  make manifests
  ```

- **Generate Helm Template:**

  ```sh
  make helm-template
  ```
  This will output the rendered manifests to `_output/manifest`.

## Development

- **Run linting and checks:**

  ```sh
  make vet
  make fmt
  ```

- **Generate code (DeepCopy, Client, etc.):**

  ```sh
  make generate-all
  ```

## License

Copyright 2025 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
