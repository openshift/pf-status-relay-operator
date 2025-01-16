set -e

export VERSION="4.19.0"

sed -i 's/quay.io\/openshift\/origin-pf-status-relay:.*$/quay.io\/openshift\/origin-pf-status-relay:'$VERSION'/g' config/manager/env_patch.yaml
sed -i 's/quay.io\/openshift\/origin-kube-rbac-proxy:.*$/quay.io\/openshift\/origin-kube-rbac-proxy:'$VERSION'/g' config/default/manager_auth_proxy_patch.yaml
make bundle

cp bundle/manifests/* manifests/stable

sed -i 's/quay.io\/openshift\/origin-kube-rbac-proxy:.*$/quay.io\/openshift\/origin-kube-rbac-proxy:'$VERSION'/g' manifests/stable/image-references
sed -i 's/quay.io\/openshift\/origin-pf-status-relay:.*$/quay.io\/openshift\/origin-pf-status-relay:'$VERSION'/g' manifests/stable/image-references
sed -i 's/quay.io\/openshift\/origin-pf-status-relay-operator:.*$/quay.io\/openshift\/origin-pf-status-relay-operator:'$VERSION'/g' manifests/stable/image-references
sed -i 's/pf-status-relay-operator\..*$/pf-status-relay-operator\.v'$VERSION'/g' manifests/pf-status-relay-operator.package.yaml
