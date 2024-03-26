# Maintenance

## How to update supported Kubernetes

Cat-gate supports the three latest Kubernetes versions.
If a new Kubernetes is released, please update the following files.

- Update Kubernetes version in `Makefile` and `e2e/Makefile`.
- Update controller-runtime version in `Makefile`.
- Update kubectl version in `aqua.yaml`.
- Update `k8s.io/*` and `sigs.k8s.io/controller-runtime` packages version in `go.mod`.
- Update Kubernetes version in `cluster.yaml`.

If Kubernetes or controller-runtime API has changed, please fix the relevant source code.

## How to update dependencies

- Update tool version in `Makefile` and `e2e/Makefile`.
- Update `aqua.yaml`.
- Update `Dockerfile`.
- Update GitHub Actions workflow files in `.github/workflows/ci.yaml` and `.github/workflows/release.yaml`.
