# Out of Pods

The `outOfPods` field simulates a node that has exceeded its capacity for allocating pods by applying a taint to the target node. This disruption is useful for testing deployment behaviors and autoscaling groups when nodes become unavailable for scheduling new pods.

## How it works

When the disruption is applied, a taint with the key `chaos.datadoghq.com/out-of-pods` is added to the target node. The taint effect can be configured:

- **NoSchedule (default)**: Prevents new pods from being scheduled on the node, but existing pods continue running normally.
- **NoExecute**: Prevents new pods from being scheduled AND evicts existing pods that don't have a matching toleration.

When the disruption ends, the taint is automatically removed from the node.

## Configuration

### Basic usage (NoSchedule)

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: out-of-pods-test
  namespace: chaos-demo
spec:
  level: node
  selector:
    node-role.kubernetes.io/worker: ""
  count: 1
  duration: 5m
  outOfPods: {}
```

This applies a `NoSchedule` taint to one worker node, preventing new pods from being scheduled on it.

### Forced mode (NoExecute)

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: out-of-pods-forced
  namespace: chaos-demo
spec:
  level: node
  selector:
    node-role.kubernetes.io/worker: ""
  count: 1
  duration: 5m
  outOfPods:
    forced: true
```

This applies a `NoExecute` taint, which will also evict existing pods that don't tolerate the taint.

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `forced` | boolean | `false` | When `true`, uses `NoExecute` taint effect (evicts existing pods). When `false`, uses `NoSchedule` (only prevents new scheduling). |

## Targeting

> :warning: The `outOfPods` disruption can **only** be applied at the **node level**. Setting `level: pod` will result in a validation error.

The selector in your disruption spec should match node labels:

```yaml
spec:
  level: node
  selector:
    node-role.kubernetes.io/worker: ""
```

## Use Cases

### Testing Horizontal Pod Autoscaler (HPA)

Simulate node capacity issues to verify that:
- HPA responds correctly when pods cannot be scheduled
- New nodes are provisioned by the cluster autoscaler
- Pods are rescheduled to healthy nodes

### Testing Deployment Rollouts

Verify deployment behavior when:
- Rolling updates cannot proceed due to scheduling constraints
- Pods are pending due to lack of schedulable nodes

### Testing Pod Disruption Budgets (PDB)

With `forced: true`, test that:
- PDBs correctly protect critical workloads
- Minimum availability is maintained during node tainting

## Compatibility

| Feature | Compatible |
|---------|------------|
| Pod level | :x: No |
| Node level | :white_check_mark: Yes |
| OnInit | :x: No |
| Pulse | :x: No |
| Combined with other disruptions | :x: No |

## Taint Details

The taint applied to nodes has the following structure:

```yaml
taints:
  - key: chaos.datadoghq.com/out-of-pods
    value: "true"
    effect: NoSchedule  # or NoExecute if forced: true
```

## Manual Cleanup

If for any reason the taint is not automatically removed, you can manually clean it up:

```bash
# View current taints on a node
kubectl describe node <node-name> | grep -A5 Taints

# Remove the out-of-pods taint
kubectl taint nodes <node-name> chaos.datadoghq.com/out-of-pods:NoSchedule-
# or for NoExecute
kubectl taint nodes <node-name> chaos.datadoghq.com/out-of-pods:NoExecute-
```

## Example: Testing Cluster Autoscaler

This example demonstrates how to use the `outOfPods` disruption to test cluster autoscaler behavior:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: test-cluster-autoscaler
  namespace: chaos-demo
spec:
  level: node
  selector:
    node-role.kubernetes.io/worker: ""
  count: 50%  # Taint half of the worker nodes
  duration: 10m
  outOfPods: {}
```

When this disruption is applied:
1. Half of the worker nodes will be tainted with `NoSchedule`
2. Any new pods will need to be scheduled on the remaining nodes
3. If capacity is insufficient, cluster autoscaler should provision new nodes
4. After 10 minutes, the taints are removed and normal scheduling resumes

