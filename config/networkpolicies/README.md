# Operator network policies

These policies are installed with the OpenShift manifests and OLM bundle.
They select pods labelled `control-plane: controller-manager` in the
operator installation namespace. Other pods in that namespace are not
selected by these policies.

No standalone default-deny policy is included. The policies allow:

- Outbound TCP 6443 to the Kubernetes API server. The rule permits any
  destination on that port; it does not identify API-server IP addresses.
- Outbound TCP/UDP 53 and 5353 to OpenShift DNS pods in `openshift-dns`.
  Port 5353 covers the DNS pod port behind the service port 53.
- Inbound TCP 9443 for admission webhooks. The rule permits any source on
  that port; it does not identify API-server IP addresses.
- Inbound TCP 8443 from pods in the operator installation namespace,
  `openshift-monitoring`, and `openshift-user-workload-monitoring` for
  authenticated HTTPS metrics. Same-namespace access preserves the
  existing metrics and TLS e2e probes.
  These policies do not install a ServiceMonitor or grant metrics RBAC.

The original policies used namespace-wide default denial and omitted
metrics access. This restoration omits the standalone deny policy and
adds metrics access and the OpenShift DNS target port.

The allow policies still isolate selected operator pods in both directions:
traffic without an allowance is denied unless another selecting policy
allows it. Removing the standalone deny policy does not disable isolation.

The operand DaemonSet uses host networking. These policies do not select
operand pods or isolate their host traffic. The controller does not
create these policies at runtime; deployment tooling installs them.

NetworkPolicy enforcement requires a supporting cluster network plugin.
Policies are additive: another policy selecting the same pod might allow
additional traffic. Node-originated health probes are exempt from ordinary
pod NetworkPolicy isolation. Validate DNS, reconciliation, admission,
and metrics on an OpenShift cluster before merging.
