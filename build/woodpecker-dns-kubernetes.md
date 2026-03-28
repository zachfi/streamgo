# Woodpecker DNS and direct Kubernetes configuration

If pipeline steps fail with DNS issues (e.g. “query was misformatted” in clone or later steps), and setting `clone.git.dns` in the pipeline did not fix it, you can try the following.

## What you might see in the pod

Inside a pipeline pod, `/etc/resolv.conf` is typically set by Kubernetes. Example (cluster domain `cluster.znet`, kube-dns in namespace `ci`):

```
search ci.svc.cluster.znet svc.cluster.znet cluster.znet wp-hsvc-17.ci.svc.cluster.local
nameserver 10.55.0.10
nameserver fccb::a
options ndots:5
```

The nameservers are the kube-dns Service IPs; they forward public lookups (e.g. github.com) upstream. For a name like `github.com` (one dot), `ndots:5` causes the resolver to try the search list first (e.g. github.com.ci.svc.cluster.znet, …) before the bare name. If cluster DNS misbehaves on those stub lookups (e.g. "query was misformatted"), you can avoid it by using the clone step’s `dns:` to point at 8.8.8.8 / 8.8.4.4, or by setting pod-level `ndots:1` via a MutatingAdmissionWebhook so the first try is the bare hostname.

## 1. Use cluster DNS in the pipeline

If egress to public DNS (e.g. 8.8.8.8) is blocked or unreliable, point the clone step at your cluster DNS instead. In `build/woodpecker.jsonnet`, set:

```jsonnet
clone: {
  git: {
    image: 'woodpeckerci/plugin-git',
    dns: [ '10.96.0.10' ],   // replace with your cluster DNS Service IP (kube-dns/CoreDNS)
  },
};
```

Get the IP with: `kubectl get svc -n kube-system kube-dns` (or `coredns`), then use the cluster IP.

## 2. Set dnsOptions via Kubernetes (Woodpecker does not expose them)

Woodpecker’s Kubernetes backend does **not** expose `dnsPolicy`, `dnsConfig.options` (e.g. `ndots`, `timeout`), or a default pod spec in pipeline `backend_options` or agent env. To set `dnsOptions` for pipeline pods you must use **direct Kubernetes configuration** below.

**Where ndots comes from:** `ndots` is **not** configurable in CoreDNS or any ConfigMap. The kubelet writes each pod’s `/etc/resolv.conf` when the pod starts; the default `ndots:5` is applied there. Editing the CoreDNS ConfigMap in k3s (e.g. in `kube-system`) changes how CoreDNS resolves queries; it does **not** change what gets written into a pod’s resolv.conf. To change ndots you must mutate the **pod spec** so `spec.dnsConfig.options` includes `ndots: "1"` (via an admission controller).

### Option A: Kyverno (recommended; scoped to CI namespace)

If you use [Kyverno](https://kyverno.io/), add a policy that mutates Pods in your Woodpecker namespace (e.g. `ci`) to set `ndots: "1"`. A jsonnet that generates this policy is in **`build/k8s/ndots-policy.jsonnet`**. Render and apply with your other manifests:

```bash
jsonnet build/k8s/ndots-policy.jsonnet | kubectl apply -f -
```

Or import the jsonnet into your Tanka/Helm pipeline. The policy only affects Pods in the namespace you set in the jsonnet (default `ci`).

### Option B: Custom MutatingAdmissionWebhook


To set pod-level ndots without Kyverno, use a webhook. In the Corefile, you can add options in the relevant block (see [CoreDNS options](https://coredns.io/plugins/kubernetes/)). For “misformatted” or resolution issues, some clusters need tuning in CoreDNS or in the **pod** `dnsConfig` (next option). Cluster-wide options in CoreDNS apply to how CoreDNS resolves; they do not set the **pod**’s `/etc/resolv.conf` options (like `ndots`). So for pod-level `ndots`/`timeout` you need Option B.

### Option B: Pod-level dnsConfig (MutatingAdmissionWebhook)

To set `spec.dnsConfig.options` (e.g. `ndots: "1"`, `timeout: "2"`) only for Woodpecker pipeline pods, use a **MutatingAdmissionWebhook** that patches pods in the Woodpecker backend namespace (e.g. `woodpecker`).

Example patch the webhook should apply to matching pods:

```yaml
spec:
  dnsPolicy: ClusterFirst
  dnsConfig:
    options:
      - name: ndots
        value: "1"
      - name: timeout
        value: "2"
```

- Implement a small webhook that matches pods in the Woodpecker namespace (e.g. by label or namespace) and merges the above into `pod.spec`.
- Or use a generic mutating webhook solution (e.g. [pod-mutator](https://github.com/your-org/pod-mutator)-style) that injects a fixed `dnsConfig` for a given namespace.

After the webhook is in place, every pipeline pod (including the clone step) will get these DNS options without any change to the Woodpecker pipeline YAML.

## Summary

| Approach | Scope | Use when |
|----------|--------|----------|
| `clone.git.dns` in pipeline | Clone step only | Different nameservers; try cluster DNS IP if 8.8.8.8 failed. |
| Kyverno (see `build/k8s/ndots-policy.jsonnet`) | Pods in CI namespace | Set ndots:1 for all pipeline pods without a custom webhook. |
| Custom MutatingAdmissionWebhook | Pods in Woodpecker namespace | You need pod-level `dnsConfig.options` and don’t use Kyverno. |
