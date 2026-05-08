# Pre-commit check

Before every commit, run:

```
make generate && make manifests && go mod tidy && go mod vendor && hack/align-ocp-version.sh
```

All commands must succeed and produce no file changes. If any files change, stage and include them in the commit.

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
./bin/operator-sdk-v1.40.0 run bundle $BUNDLE_IMG ...
```
Never inline TAG computation with &&.


## Deployment methods

<<<<<<< HEAD
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

=======
```
TAG=<branch>
IMG=quay.io/karampok/pf-status-relay-operator:$TAG
BUNDLE_IMG=quay.io/karampok/pf-status-relay-operator-bundle:$TAG
NS=openshift-pf-status-relay-operator

# Build and push operator image
podman build -t $IMG .
podman push $IMG

# Patch CSV with correct operator image, regenerate bundle
( cd config/manager && ./bin/kustomize-v5.3.0 edit set image controller=$IMG )
./bin/operator-sdk-v1.40.0 generate kustomize manifests -q  # refreshes base CSV from Go markers; no-op if source unchanged
./bin/kustomize-v5.3.0 build config/manifests | ./bin/operator-sdk-v1.40.0 generate bundle -q --overwrite --version 4.22.0
./bin/operator-sdk-v1.40.0 bundle validate ./bundle

# Build and push bundle image
podman build -f bundle.Dockerfile -t $BUNDLE_IMG .
podman push $BUNDLE_IMG

# Create namespace with required SCC label
oc create ns $NS --dry-run=client -o yaml | oc apply -f -
oc label ns $NS security.openshift.io/scc.podSecurityLabelSync=true --overwrite

# Deploy
./bin/operator-sdk-v1.40.0 run bundle $BUNDLE_IMG \
  -n $NS --install-mode=OwnNamespace --kubeconfig $KUBECONFIG
```

The bundle regeneration modifies `config/manager/kustomization.yaml` and `bundle/` — do not commit those changes.

Undo with:

```
./bin/operator-sdk-v1.40.0 cleanup pf-status-relay-operator -n $NS --kubeconfig $KUBECONFIG
oc delete ns $NS
```

After deploy, verify the running pod uses the expected image digest — not a stale cached version.

Undo with `operator-sdk cleanup <operator-name> -n <namespace>`.

### OLM via CatalogSource + Subscription (production)

`make catalog-build catalog-push`, then create CatalogSource + Subscription.

# Testing

## Test layout

- `e2e/` — real e2e suite, separate Go module. This is what CI runs via `operator-sdk run bundle`.

## e2e target

```
make e2e
```

Runs `go test -C e2e -tags e2e -v -timeout 10m .` against a live cluster. Requires operator deployed and `KUBECONFIG` set.

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

- Requires `webhook-server-cert` secret — on OCP this is auto-created by the service-CA operator via annotation on the webhook service.

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
