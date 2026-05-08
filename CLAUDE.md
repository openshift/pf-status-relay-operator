# Dev workflow

## Environment

```
export IMG=quay.io/<youruser>/pf-status-relay-operator:<use-gitbranch-trimmed>
export BUNDLE_IMG=quay.io/<youruser>/pf-status-relay-operator-bundle:<use-gitbranch-trimmed>
export KUBECONFIG=/path/to/kubeconfig
```
When suggesting commands involving TAG or BUNDLE_IMG, set TAG as a separate variable first:
```
TAG=xyz
BUNDLE_IMG=quay.io/karampok/pf-status-relay-operator-bundle:$TAG
./bin/operator-sdk run bundle $BUNDLE_IMG ...
```
Never inline TAG computation with &&.


## Deployment methods

### 1. Raw manifests (OpenShift only)

Uses OpenShift service-CA for webhook certs — does not work on vanilla k8s.

Undo with `make undeploy && make uninstall`.

### 1b. Raw manifests on kind (k8s-certmanager-prom overlay)

quay.io/openshift images are private — build locally and load into kind:

```
make docker-build IMG=<img>
kind load docker-image <img> --name <cluster-name>
make deploy OVERLAY=config/overlays/k8s-certmanager-prom IMG=<img>
```

Install cert-manager and prometheus-operator before deploying.

Undo with `make undeploy OVERLAY=config/overlays/k8s-certmanager-prom && make uninstall`.

### 2. operator-sdk run bundle (OLM-based, works on kind)

On kind: load images into kind nodes before running the bundle command.

This is what CI does — the e2e workflow calls `operator-sdk run bundle`
with the bundle image built by ci-operator.

Undo with `operator-sdk cleanup <operator-name> -n <namespace>`.

### 3. OLM via CatalogSource + Subscription (production)

`make catalog-build catalog-push`, then create CatalogSource + Subscription.

# Testing

## Test layout

Two separate e2e locations — different purposes:

- `test/e2e/` — legacy kubebuilder scaffold, to be removed. Procedure preserved below.
- `e2e/` — real e2e suite, separate Go module. This is what CI runs via `operator-sdk run bundle`.

## Smoke test (replaces test/e2e)

Verifies operator pod reaches Running on a kind cluster. Steps in order:

1. Install prometheus-operator `v0.72.0` and cert-manager `v1.14.4`; wait for cert-manager webhook ready
2. Create namespace `pf-status-relay-operator-system`
3. Build operator image and load into kind (respects `KIND_CLUSTER` env var, default: `kind`)
4. `make install` — installs CRDs
5. `make deploy IMG=<img>` — deploys operator
6. Assert exactly 1 pod with label `control-plane=controller-manager` in namespace `pf-status-relay-operator-system` reaches phase `Running`

Teardown: delete prometheus-operator, cert-manager, namespace.

# Architecture

## Controller

- DaemonSet name: `pf-status-relay-ds-{monitor-name}`
- ServiceAccount: `pf-status-relay-operator-pf-status-relay`
- Container env vars: `PF_STATUS_RELAY_INTERFACES` (comma-separated), `PF_STATUS_RELAY_POLLING_INTERVAL`
- Image source: `PF_STATUS_RELAY_IMAGE` env var on operator pod

## Conflict detection (`api/v1alpha1/validate.go`)

- Two monitors conflict if overlapping node selectors AND shared interfaces
- Selectors overlap if they share ANY key-value pair
- `nil` nodeSelector overlaps with everything
- Degraded monitors skipped during validation

## Webhook

- Requires `webhook-server-cert` secret — pods fail without cert-manager installed

# Notes


username from (podman/docker login --get-login quay.io)



## CI

**Config:** https://github.com/openshift/release/tree/main/ci-operator/config/openshift/pf-status-relay-operator/

**Generated jobs:** https://github.com/openshift/release/tree/main/ci-operator/jobs/openshift/pf-status-relay-operator/

**Step registry:**
- `optional-operators-ci-operator-sdk-aws` — e2e workflow
- `openshift-ci-security` — security scan
- `go-verify-deps` — dep check

**Docs:**
- https://docs.ci.openshift.org/
- https://steps.ci.openshift.org/ci-operator-reference
