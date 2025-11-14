# Advanced Go Projects - Detailed Descriptions

This document provides comprehensive descriptions, architectural deep-dives, and implementation details for advanced-level Go projects that demonstrate mastery of distributed systems, language implementation, and high-performance computing.

---

## Table of Contents
1. [Distributed Database with Raft Consensus](#1-distributed-database-with-raft-consensus)
2. [Container Orchestration Platform](#2-container-orchestration-platform)
3. [High-Performance API Gateway with Dynamic Routing](#3-high-performance-api-gateway-with-dynamic-routing)
4. [Custom Language Compiler and VM](#4-custom-language-compiler-and-vm)
5. [Distributed Tracing and Observability Platform](#5-distributed-tracing-and-observability-platform)
6. [Real-Time Stream Processing Engine](#6-real-time-stream-processing-engine)
7. [Service Mesh Implementation](#7-service-mesh-implementation)
8. [Blockchain and Smart Contract Platform](#8-blockchain-and-smart-contract-platform)
9. [Machine Learning Model Serving Platform](#9-machine-learning-model-serving-platform)
10. [Distributed File System](#10-distributed-file-system)

---

## 1. Distributed Database with Raft Consensus

### Comprehensive Description

Building a distributed database from scratch is one of the most challenging and rewarding projects in computer science. It requires deep understanding of consensus algorithms, distributed systems theory, storage engines, and fault tolerance. This project will push your skills to the limit and teach you principles used in systems like etcd, Consul, and CockroachDB.

### Deep Architectural Dive

#### The Raft Algorithm in Detail

Raft is a consensus algorithm designed for understandability. Unlike Paxos, Raft decomposes consensus into three independent sub-problems:

**1. Leader Election**

The cluster operates in terms—monotonically increasing integers. Each term begins with an election:

```
Term 1: [Leader A]
Term 2: [Leader B] (A failed)
Term 3: [Leader B] (no leader change)
Term 4: [Leader C] (B failed)
```

State transitions:
```
Follower → (timeout) → Candidate → (majority votes) → Leader
Leader → (discovers higher term) → Follower
Candidate → (discovers leader/higher term) → Follower
```

Election timeout is randomized (typically 150-300ms) to prevent split votes. When a follower doesn't receive heartbeat within election timeout:
1. Increment current term
2. Vote for itself
3. Send RequestVote RPC to all peers
4. If receives majority votes, become leader
5. If receives heartbeat from valid leader, revert to follower
6. If election timeout elapses with no winner, start new election

**Safety properties**:
- **Election Safety**: At most one leader per term
- **Leader Append-Only**: Leader never overwrites or deletes log entries
- **Log Matching**: If two logs contain an entry with same index and term, all preceding entries are identical

**2. Log Replication**

Once elected, the leader serves client requests. Each request is appended to the log and replicated:

```
Client → Leader: Set x=5
Leader log: [1: Set x=5] (uncommitted)
Leader → Followers: AppendEntries([1: Set x=5])
Followers → Leader: Success
Leader: Commit index = 1 (majority confirmed)
Leader → Followers: AppendEntries(commitIndex=1)
Followers: Apply Set x=5 to state machine
```

The AppendEntries RPC includes:
- `prevLogIndex`: Index of log entry immediately preceding new ones
- `prevLogTerm`: Term of prevLogIndex entry
- `entries[]`: Log entries to store
- `leaderCommit`: Leader's commit index

Followers verify log consistency:
```go
if log[prevLogIndex].term != prevLogTerm {
    return false // Log inconsistent
}
```

If consistency check fails, leader decrements `nextIndex` for that follower and retries, eventually finding point where logs match.

**3. Safety**

Raft ensures that committed entries are durable using the **Leader Completeness** property:

*If a log entry is committed in a given term, that entry will be present in the logs of all leaders for all higher-numbered terms.*

This is achieved through election restriction: candidate's log must be "at least as up-to-date" as any other log in the majority:

```go
func (r *Raft) isLogUpToDate(candidateLastLogIndex, candidateLastLogTerm int64) bool {
    lastLogTerm := r.log[len(r.log)-1].Term
    lastLogIndex := int64(len(r.log) - 1)

    // Candidate's log is more up-to-date if:
    // 1. Last term is higher, OR
    // 2. Last term is same AND log is at least as long
    if candidateLastLogTerm > lastLogTerm {
        return true
    }
    if candidateLastLogTerm == lastLogTerm && candidateLastLogIndex >= lastLogIndex {
        return true
    }
    return false
}
```

#### Storage Engine Deep Dive

A production-grade storage engine requires sophisticated data structures:

**LSM Tree Architecture**:
```
Write Path:
  Write → WAL → MemTable → (flush) → L0 SSTable

Read Path:
  Read → MemTable → L0 SSTables → L1 SSTables → ... → Ln SSTables

Compaction:
  L0: 4 SSTables (overlapping) → Compact → L1: 1 SSTable
  L1 → L2 when size threshold exceeded
```

**Components**:

1. **Write-Ahead Log (WAL)**:
```go
type WAL struct {
    file       *os.File
    encoder    *gob.Encoder
    syncPolicy SyncPolicy
}

func (w *WAL) Append(entry LogEntry) error {
    if err := w.encoder.Encode(entry); err != nil {
        return err
    }

    if w.syncPolicy == SyncEveryWrite {
        return w.file.Sync() // fsync to disk
    }
    return nil
}
```

2. **MemTable** (in-memory sorted map):
```go
type MemTable struct {
    data    *skiplist.SkipList // Or Red-Black tree
    size    int64
    maxSize int64
}

func (m *MemTable) Put(key, value []byte) error {
    if m.size >= m.maxSize {
        return ErrMemTableFull
    }
    m.data.Insert(key, value)
    m.size += int64(len(key) + len(value))
    return nil
}
```

3. **SSTable** (Sorted String Table):
```
SSTable Structure:
+------------------+
| Data Block 1     | ← key-value pairs
| Data Block 2     |
| ...              |
| Data Block N     |
+------------------+
| Filter Block     | ← Bloom filter
+------------------+
| Index Block      | ← offsets to data blocks
+------------------+
| Footer           | ← metadata
+------------------+
```

```go
type SSTable struct {
    fileHandle  *os.File
    indexBlock  *IndexBlock
    filterBlock *BloomFilter
    cache       *BlockCache
}

func (sst *SSTable) Get(key []byte) ([]byte, error) {
    // Check bloom filter first
    if !sst.filterBlock.MayContain(key) {
        return nil, ErrNotFound
    }

    // Find data block using index
    blockIndex := sst.indexBlock.FindBlock(key)

    // Read block (with caching)
    block := sst.cache.Get(blockIndex)
    if block == nil {
        block = sst.readBlock(blockIndex)
        sst.cache.Put(blockIndex, block)
    }

    // Binary search within block
    return block.Search(key)
}
```

4. **Compaction**:
```go
func (e *Engine) compact(level int) error {
    // Select files to compact
    files := e.selectCompactionFiles(level)

    // Merge sorted files
    iterator := NewMergingIterator(files)

    // Write new SSTable
    writer := NewSSTableWriter(level + 1)
    for iterator.Next() {
        // Skip deleted keys
        if iterator.Value() != tombstone {
            writer.Append(iterator.Key(), iterator.Value())
        }
    }

    // Atomically swap old files with new
    return e.installCompactionResults(files, writer.Finalize())
}
```

**Optimization**: Bloom filters reduce disk reads by ~10x for non-existent keys:
```go
type BloomFilter struct {
    bits       []byte
    numHashes  int
    numBits    int
}

func (bf *BloomFilter) Add(key []byte) {
    for i := 0; i < bf.numHashes; i++ {
        hash := bf.hash(key, i)
        bf.bits[hash/8] |= (1 << (hash % 8))
    }
}

func (bf *BloomFilter) MayContain(key []byte) bool {
    for i := 0; i < bf.numHashes; i++ {
        hash := bf.hash(key, i)
        if bf.bits[hash/8]&(1<<(hash%8)) == 0 {
            return false
        }
    }
    return true // Maybe present
}
```

#### Snapshot and Recovery

Snapshots prevent unbounded log growth:

```go
type Snapshot struct {
    LastIncludedIndex int64
    LastIncludedTerm  int64
    Data              []byte // Serialized state machine
}

func (r *Raft) CreateSnapshot() (*Snapshot, error) {
    if r.commitIndex <= r.lastSnapshotIndex {
        return nil, nil // Nothing to snapshot
    }

    // Serialize state machine
    data, err := r.stateMachine.Serialize()
    if err != nil {
        return nil, err
    }

    snapshot := &Snapshot{
        LastIncludedIndex: r.commitIndex,
        LastIncludedTerm:  r.log[r.commitIndex].Term,
        Data:              data,
    }

    // Persist snapshot
    if err := r.persistSnapshot(snapshot); err != nil {
        return nil, err
    }

    // Truncate log
    r.log = r.log[r.commitIndex+1:]
    r.lastSnapshotIndex = snapshot.LastIncludedIndex
    r.lastSnapshotTerm = snapshot.LastIncludedTerm

    return snapshot, nil
}
```

InstallSnapshot RPC for catching up lagging followers:
```go
func (r *Raft) InstallSnapshot(req *InstallSnapshotRequest) error {
    if req.Term < r.currentTerm {
        return fmt.Errorf("stale term")
    }

    if req.LastIncludedIndex <= r.lastSnapshotIndex {
        return nil // Already have this snapshot
    }

    // Save snapshot to disk
    if err := r.persistSnapshot(&req.Snapshot); err != nil {
        return err
    }

    // Discard logs before snapshot
    if req.LastIncludedIndex >= int64(len(r.log)) {
        r.log = []LogEntry{}
    } else {
        r.log = r.log[req.LastIncludedIndex:]
    }

    // Update state machine
    r.stateMachine.Restore(req.Snapshot.Data)

    r.lastSnapshotIndex = req.LastIncludedIndex
    r.lastSnapshotTerm = req.LastIncludedTerm
    r.commitIndex = req.LastIncludedIndex
    r.lastApplied = req.LastIncludedIndex

    return nil
}
```

#### Linearizable Reads

Challenge: Reads from leader might return stale data if leader is partitioned.

**Solution 1: Read Index**
```go
func (r *Raft) LinearizableRead() ([]byte, error) {
    readIndex := r.commitIndex

    // Confirm leadership by broadcasting heartbeat
    if !r.confirmLeadership() {
        return nil, ErrNotLeader
    }

    // Wait until commit index >= read index
    r.waitForApply(readIndex)

    return r.stateMachine.Read(key)
}

func (r *Raft) confirmLeadership() bool {
    responses := 0
    for _, peer := range r.peers {
        if r.sendHeartbeat(peer) {
            responses++
        }
    }
    return responses >= r.majority()
}
```

**Solution 2: Lease-Based Reads**
```go
type LeaderLease struct {
    startTime time.Time
    duration  time.Duration
}

func (r *Raft) LeaseRead(key []byte) ([]byte, error) {
    if time.Now().Before(r.lease.startTime.Add(r.lease.duration)) {
        return r.stateMachine.Read(key), nil // Safe to read
    }
    return nil, ErrLeaseExpired
}
```

Leader extends lease on each successful heartbeat. Safe because:
- Election timeout > Lease duration
- If leader is partitioned, lease expires before new leader elected

#### Multi-Raft for Sharding

Single Raft group doesn't scale beyond thousands of QPS. Solution: **Multi-Raft**

```
Key Space: [a-z]
Shard 1: [a-m] → Raft Group 1 (Node 1, 2, 3)
Shard 2: [n-z] → Raft Group 2 (Node 2, 3, 4)
```

```go
type MultiRaftNode struct {
    raftGroups map[uint64]*Raft // shard ID → Raft instance
    router     *ShardRouter
}

func (n *MultiRaftNode) Put(key, value []byte) error {
    shardID := n.router.GetShard(key)
    raft := n.raftGroups[shardID]
    return raft.Propose(PutCommand{Key: key, Value: value})
}
```

**Shard Splitting**:
```go
func (n *MultiRaftNode) SplitShard(oldShardID uint64, splitKey []byte) error {
    // 1. Create new shard with same replicas
    newShardID := generateShardID()
    newRaft := n.createRaftGroup(newShardID, n.getReplicasFor(oldShardID))

    // 2. Copy data [splitKey, ∞) to new shard
    snapshot := n.raftGroups[oldShardID].CreateSnapshot()
    newRaftSnapshot := extractRange(snapshot, splitKey, nil)
    newRaft.InstallSnapshot(newRaftSnapshot)

    // 3. Delete data [splitKey, ∞) from old shard
    n.raftGroups[oldShardID].Propose(DeleteRangeCommand{Start: splitKey})

    // 4. Update routing table
    n.router.UpdateShard(oldShardID, nil, splitKey)
    n.router.AddShard(newShardID, splitKey, nil)

    return nil
}
```

### Real-World Challenges and Solutions

#### Challenge 1: Network Partitions
**Problem**: Split-brain, data divergence

**Solution**:
- Quorum-based decisions (majority voting)
- Higher term always wins
- Log matching property ensures consistency

**Testing**: Use Jepsen to inject partitions

#### Challenge 2: Clock Skew
**Problem**: Distributed systems can't rely on synchronized clocks

**Solution**:
- Never compare timestamps across nodes
- Use logical clocks (Lamport timestamps, Vector clocks)
- Raft uses terms (logical time) instead of wall-clock time

#### Challenge 3: Cascading Failures
**Problem**: One slow node causes entire cluster to slow down

**Solution**:
- Timeouts at every layer
- Circuit breakers for failed nodes
- Isolate slow followers (don't wait indefinitely)

```go
func (r *Raft) replicateToFollower(follower string, entries []LogEntry) error {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    done := make(chan error, 1)
    go func() {
        done <- r.sendAppendEntries(follower, entries)
    }()

    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        return ErrTimeout
    }
}
```

#### Challenge 4: Large Value Handling
**Problem**: Raft log can't handle multi-GB values efficiently

**Solution**:
- Store large values separately in blob storage
- Replicate only metadata through Raft
- Use reference counting for garbage collection

```go
type LargeValueRef struct {
    BlobID     string
    Size       int64
    Checksum   []byte
}

func (db *Database) PutLarge(key []byte, value []byte) error {
    // Store blob in object storage
    blobID := generateBlobID()
    if err := db.blobStore.Put(blobID, value); err != nil {
        return err
    }

    // Replicate reference through Raft
    ref := LargeValueRef{
        BlobID:   blobID,
        Size:     int64(len(value)),
        Checksum: sha256.Sum256(value),
    }

    return db.raft.Propose(PutCommand{Key: key, Value: encodeRef(ref)})
}
```

### Performance Optimization Techniques

1. **Batching**: Group multiple client requests into one Raft proposal
2. **Pipelining**: Send AppendEntries without waiting for response (track in-flight)
3. **Parallel Applies**: Apply committed entries to state machine in parallel
4. **Async Disk Writes**: Group fsync calls
5. **Zero-Copy**: Use `io.Copy` and `sendfile` for large transfers

```go
type Batcher struct {
    requests chan *Request
    batch    []*Request
    maxSize  int
    timeout  time.Duration
}

func (b *Batcher) Run() {
    ticker := time.NewTicker(b.timeout)
    defer ticker.Stop()

    for {
        select {
        case req := <-b.requests:
            b.batch = append(b.batch, req)
            if len(b.batch) >= b.maxSize {
                b.flush()
            }
        case <-ticker.C:
            if len(b.batch) > 0 {
                b.flush()
            }
        }
    }
}

func (b *Batcher) flush() {
    // Single Raft proposal for entire batch
    b.raft.ProposeBatch(b.batch)
    b.batch = b.batch[:0]
}
```

### Testing Strategy

**Unit Tests**:
- State transitions (Follower → Candidate → Leader)
- Log replication correctness
- Election safety

**Integration Tests**:
- 3-node cluster setup
- Leader election after failures
- Log consistency across nodes

**Chaos Tests** (Jepsen-style):
```go
func TestNetworkPartition(t *testing.T) {
    cluster := NewTestCluster(5)

    // Partition: [1,2,3] | [4,5]
    cluster.PartitionNodes([]int{1,2,3}, []int{4,5})

    // Write to majority partition
    cluster.Node(1).Put("key", "value")

    // Heal partition
    cluster.HealPartition()

    // Verify all nodes have same value
    for i := 1; i <= 5; i++ {
        assert.Equal(t, "value", cluster.Node(i).Get("key"))
    }
}
```

**Performance Tests**:
- Throughput: ops/second
- Latency: P50, P95, P99
- Scalability: Add nodes and measure

---

## 2. Container Orchestration Platform

### Comprehensive Description

Container orchestration is the backbone of modern cloud-native infrastructure. Building a simplified Kubernetes teaches you about distributed scheduling, declarative APIs, reconciliation loops, and managing complex system state across many machines.

### Architecture Deep Dive

#### Control Plane vs Data Plane

**Control Plane** (brain):
- API Server: REST/gRPC interface
- Scheduler: Decides pod→node placement
- Controller Manager: Runs reconciliation loops
- etcd: Stores all cluster state

**Data Plane** (muscle):
- Kubelet: Node agent, manages pod lifecycle
- Container Runtime: Actually runs containers (containerd, Docker)
- Kube-Proxy: Network routing for services
- CNI Plugin: Sets up pod networking

```
┌─────────────────────────────────────────────────┐
│              Control Plane                      │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐│
│  │   API    │  │ Scheduler │  │ Controller   ││
│  │  Server  │  │           │  │   Manager    ││
│  └────┬─────┘  └─────┬─────┘  └──────┬───────┘│
│       │              │                │        │
│       └──────────────┴────────────────┘        │
│                      │                         │
│                ┌─────▼─────┐                   │
│                │    etcd   │                   │
│                └───────────┘                   │
└─────────────────────────────────────────────────┘
         │                          │
    ┌────▼────┐                ┌───▼─────┐
    │ Node 1  │                │ Node 2  │
    │┌────────┐│                │┌────────┐
    ││Kubelet ││                ││Kubelet ││
    │└────────┘│                │└────────┘│
    │┌────────┐│                │┌────────┐│
    ││Container│               ││Container││
    ││Runtime │                ││Runtime  ││
    │└────────┘│                │└────────┘│
    └─────────┘                 └─────────┘
```

#### Declarative API Design

Kubernetes uses a declarative model: users specify **desired state**, system converges to it.

**Resource Definition**:
```go
type Pod struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       PodSpec   `json:"spec"`
    Status     PodStatus `json:"status"`
}

type ObjectMeta struct {
    Name              string            `json:"name"`
    Namespace         string            `json:"namespace"`
    UID               types.UID         `json:"uid"`
    ResourceVersion   string            `json:"resourceVersion"`
    Generation        int64             `json:"generation"`
    Labels            map[string]string `json:"labels"`
    Annotations       map[string]string `json:"annotations"`
}
```

**API Machinery**:
```go
type RESTStorage interface {
    Create(ctx context.Context, obj runtime.Object) (runtime.Object, error)
    Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, error)
    Get(ctx context.Context, name string) (runtime.Object, error)
    Delete(ctx context.Context, name string) (runtime.Object, error)
    List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error)
    Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error)
}

// etcd-backed storage
type PodStorage struct {
    etcdClient *clientv3.Client
}

func (s *PodStorage) Create(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
    pod := obj.(*Pod)

    // Generate UID
    pod.UID = types.UID(uuid.New().String())
    pod.ResourceVersion = "1"

    // Validation
    if errs := validation.ValidatePod(pod); len(errs) > 0 {
        return nil, errors.NewInvalid(schema.GroupKind{Kind: "Pod"}, pod.Name, errs)
    }

    // Set defaults
    scheme.Default(pod)

    // Store in etcd
    key := fmt.Sprintf("/registry/pods/%s/%s", pod.Namespace, pod.Name)
    data, err := json.Marshal(pod)
    if err != nil {
        return nil, err
    }

    _, err = s.etcdClient.Put(ctx, key, string(data))
    return pod, err
}
```

**Watch API** (for real-time updates):
```go
func (s *PodStorage) Watch(ctx context.Context, opts *internalversion.ListOptions) (watch.Interface, error) {
    watcher := &podWatcher{
        result: make(chan watch.Event, 100),
        done:   make(chan struct{}),
    }

    // Watch etcd for changes
    prefix := fmt.Sprintf("/registry/pods/%s/", opts.Namespace)
    watchChan := s.etcdClient.Watch(ctx, prefix, clientv3.WithPrefix())

    go func() {
        defer close(watcher.result)
        for {
            select {
            case <-ctx.Done():
                return
            case <-watcher.done:
                return
            case wresp := <-watchChan:
                for _, ev := range wresp.Events {
                    pod := &Pod{}
                    json.Unmarshal(ev.Kv.Value, pod)

                    var eventType watch.EventType
                    switch ev.Type {
                    case clientv3.EventTypePut:
                        if ev.IsCreate() {
                            eventType = watch.Added
                        } else {
                            eventType = watch.Modified
                        }
                    case clientv3.EventTypeDelete:
                        eventType = watch.Deleted
                    }

                    watcher.result <- watch.Event{
                        Type:   eventType,
                        Object: pod,
                    }
                }
            }
        }
    }()

    return watcher, nil
}
```

#### Scheduler Deep Dive

The scheduler is responsible for pod→node assignment.

**Scheduling Algorithm**:
```
1. Filter (Predicates): Remove infeasible nodes
2. Score (Priorities): Rank remaining nodes
3. Select: Pick highest-scoring node
4. Bind: Update pod.Spec.NodeName in etcd
```

**Filter Phase**:
```go
type PredicateFunc func(*Pod, *NodeInfo) bool

var predicates = []PredicateFunc{
    PodFitsResources,
    PodFitsHostPorts,
    MatchNodeSelector,
    CheckNodeAffinity,
    CheckTaints,
}

func PodFitsResources(pod *Pod, node *NodeInfo) bool {
    requested := calculatePodResources(pod)
    available := node.Allocatable

    if requested.CPU > available.CPU-node.Requested.CPU {
        return false
    }
    if requested.Memory > available.Memory-node.Requested.Memory {
        return false
    }

    return true
}

func MatchNodeSelector(pod *Pod, node *NodeInfo) bool {
    for key, value := range pod.Spec.NodeSelector {
        if node.Labels[key] != value {
            return false
        }
    }
    return true
}
```

**Score Phase**:
```go
type PriorityFunc func(*Pod, *NodeInfo) int64

var priorities = []PriorityFunc{
    LeastRequestedPriority,      // Prefer nodes with more free resources
    BalancedResourceAllocation,  // Balance CPU and memory usage
    SpreadPriority,              // Spread pods across nodes
}

func LeastRequestedPriority(pod *Pod, node *NodeInfo) int64 {
    requested := calculatePodResources(pod)

    cpuScore := (node.Allocatable.CPU - node.Requested.CPU - requested.CPU) * 100 / node.Allocatable.CPU
    memScore := (node.Allocatable.Memory - node.Requested.Memory - requested.Memory) * 100 / node.Allocatable.Memory

    return (cpuScore + memScore) / 2
}

func SpreadPriority(pod *Pod, node *NodeInfo) int64 {
    // Prefer nodes with fewer pods from same service
    selector := labels.SelectorFromSet(pod.Labels)
    matchingPods := 0
    for _, existingPod := range node.Pods {
        if selector.Matches(labels.Set(existingPod.Labels)) {
            matchingPods++
        }
    }

    return 100 - int64(matchingPods*10) // Penalty for colocation
}
```

**Affinity/Anti-Affinity**:
```go
type PodAffinity struct {
    RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm
    PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm
}

type PodAffinityTerm struct {
    LabelSelector *metav1.LabelSelector
    TopologyKey   string // e.g., "kubernetes.io/hostname"
}

func CheckPodAffinity(pod *Pod, node *NodeInfo, allPods []*Pod) bool {
    for _, term := range pod.Spec.Affinity.PodAffinity.RequiredDuringScheduling {
        // Find pods matching label selector
        matchingPods := filterPods(allPods, term.LabelSelector)

        // Check if any matching pod is in same topology domain
        nodeTopology := node.Labels[term.TopologyKey]
        found := false
        for _, matchingPod := range matchingPods {
            podNode := getNodeForPod(matchingPod)
            if podNode.Labels[term.TopologyKey] == nodeTopology {
                found = true
                break
            }
        }

        if !found {
            return false // Required affinity not satisfied
        }
    }
    return true
}
```

#### Controller Pattern

Controllers implement the "observe, analyze, act" loop:

```go
type Controller interface {
    Run(stopCh <-chan struct{})
}

type DeploymentController struct {
    deploymentLister listers.DeploymentLister
    replicaSetLister listers.ReplicaSetLister
    podLister        listers.PodLister
    client           kubernetes.Interface
    queue            workqueue.RateLimitingInterface
}

func (c *DeploymentController) Run(stopCh <-chan struct{}) {
    defer c.queue.ShutDown()

    // Start informers (watch for changes)
    go c.deploymentInformer.Run(stopCh)
    go c.replicaSetInformer.Run(stopCh)

    // Wait for caches to sync
    cache.WaitForCacheSync(stopCh,
        c.deploymentInformer.HasSynced,
        c.replicaSetInformer.HasSynced)

    // Start worker goroutines
    for i := 0; i < 5; i++ {
        go wait.Until(c.worker, time.Second, stopCh)
    }

    <-stopCh
}

func (c *DeploymentController) worker() {
    for c.processNextItem() {
    }
}

func (c *DeploymentController) processNextItem() bool {
    key, shutdown := c.queue.Get()
    if shutdown {
        return false
    }
    defer c.queue.Done(key)

    err := c.syncDeployment(key.(string))
    if err != nil {
        // Retry with exponential backoff
        c.queue.AddRateLimited(key)
        return true
    }

    c.queue.Forget(key)
    return true
}

func (c *DeploymentController) syncDeployment(key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }

    deployment, err := c.deploymentLister.Deployments(namespace).Get(name)
    if errors.IsNotFound(err) {
        // Deployment deleted, clean up
        return c.cleanup(namespace, name)
    }
    if err != nil {
        return err
    }

    // Reconcile: make reality match desired state
    return c.reconcile(deployment)
}
```

**Reconciliation Logic**:
```go
func (c *DeploymentController) reconcile(deployment *Deployment) error {
    // 1. Get all ReplicaSets owned by this Deployment
    rsList, err := c.getReplicaSetsForDeployment(deployment)
    if err != nil {
        return err
    }

    // 2. Compute desired ReplicaSet
    desiredRS := c.computeDesiredReplicaSet(deployment)

    // 3. Find current (newest) ReplicaSet
    currentRS := findNewestReplicaSet(rsList)

    // 4. Decide what action to take
    switch {
    case currentRS == nil:
        // No ReplicaSet exists, create one
        return c.createReplicaSet(desiredRS)

    case !replicaSetMatches(currentRS, desiredRS):
        // Template changed, rolling update needed
        return c.performRollingUpdate(deployment, currentRS, desiredRS)

    case currentRS.Spec.Replicas != deployment.Spec.Replicas:
        // Scale ReplicaSet
        return c.scaleReplicaSet(currentRS, deployment.Spec.Replicas)

    default:
        // No action needed
        return nil
    }
}
```

**Rolling Update**:
```go
func (c *DeploymentController) performRollingUpdate(
    deployment *Deployment,
    oldRS *ReplicaSet,
    newRS *ReplicaSet,
) error {
    maxSurge := deployment.Spec.Strategy.RollingUpdate.MaxSurge
    maxUnavailable := deployment.Spec.Strategy.RollingUpdate.MaxUnavailable

    desired := deployment.Spec.Replicas

    // Create new ReplicaSet with 0 replicas
    if newRS == nil {
        newRS, err = c.createReplicaSet(desiredRS)
        if err != nil {
            return err
        }
    }

    // Calculate how many pods to add/remove
    newReplicas, oldReplicas := calculateRollingUpdateReplicas(
        oldRS.Status.AvailableReplicas,
        newRS.Status.AvailableReplicas,
        desired,
        maxSurge,
        maxUnavailable,
    )

    // Scale up new ReplicaSet
    if newRS.Spec.Replicas < newReplicas {
        if err := c.scaleReplicaSet(newRS, newReplicas); err != nil {
            return err
        }
    }

    // Scale down old ReplicaSet
    if oldRS.Spec.Replicas > oldReplicas {
        if err := c.scaleReplicaSet(oldRS, oldReplicas); err != nil {
            return err
        }
    }

    return nil
}
```

#### Kubelet Implementation

The kubelet is the node agent responsible for pod lifecycle:

```go
type Kubelet struct {
    hostname          string
    podManager        *PodManager
    containerRuntime  ContainerRuntime
    volumeManager     *VolumeManager
    statusManager     *StatusManager
    podWorkers        map[types.UID]chan *Pod
}

func (k *Kubelet) Run() {
    go k.syncLoop()
    go k.statusManager.Start()
    go k.volumeManager.Run()
}

func (k *Kubelet) syncLoop() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        <-ticker.C

        // Get desired pods from API server
        desiredPods, err := k.getDesiredPods()
        if err != nil {
            continue
        }

        // Get current pods on node
        currentPods := k.podManager.GetPods()

        // Compute diff and sync
        k.syncPods(desiredPods, currentPods)
    }
}

func (k *Kubelet) syncPods(desired, current []*Pod) {
    desiredSet := make(map[types.UID]*Pod)
    for _, pod := range desired {
        desiredSet[pod.UID] = pod
    }

    currentSet := make(map[types.UID]*Pod)
    for _, pod := range current {
        currentSet[pod.UID] = pod
    }

    // Start new pods
    for uid, pod := range desiredSet {
        if _, exists := currentSet[uid]; !exists {
            k.dispatchWork(pod, kubetypes.SyncPodCreate)
        }
    }

    // Update existing pods
    for uid, pod := range desiredSet {
        if currentPod, exists := currentSet[uid]; exists {
            if !reflect.DeepEqual(pod.Spec, currentPod.Spec) {
                k.dispatchWork(pod, kubetypes.SyncPodUpdate)
            }
        }
    }

    // Kill deleted pods
    for uid, pod := range currentSet {
        if _, exists := desiredSet[uid]; !exists {
            k.dispatchWork(pod, kubetypes.SyncPodKill)
        }
    }
}

func (k *Kubelet) dispatchWork(pod *Pod, syncType kubetypes.SyncPodType) {
    worker, exists := k.podWorkers[pod.UID]
    if !exists {
        worker = make(chan *Pod, 1)
        k.podWorkers[pod.UID] = worker
        go k.podWorkerLoop(pod.UID, worker)
    }

    worker <- pod
}

func (k *Kubelet) podWorkerLoop(podUID types.UID, updates <-chan *Pod) {
    for pod := range updates {
        if err := k.syncPod(pod); err != nil {
            klog.Errorf("Error syncing pod %s: %v", pod.Name, err)
        }
    }
}

func (k *Kubelet) syncPod(pod *Pod) error {
    // 1. Pull image if needed
    for _, container := range pod.Spec.Containers {
        if err := k.containerRuntime.PullImage(container.Image); err != nil {
            return err
        }
    }

    // 2. Create pod sandbox (network namespace)
    sandboxID, err := k.containerRuntime.CreatePodSandbox(pod)
    if err != nil {
        return err
    }

    // 3. Setup volumes
    if err := k.volumeManager.SetupVolumes(pod); err != nil {
        return err
    }

    // 4. Start containers
    for _, container := range pod.Spec.Containers {
        containerID, err := k.containerRuntime.CreateContainer(sandboxID, container, pod)
        if err != nil {
            return err
        }

        if err := k.containerRuntime.StartContainer(containerID); err != nil {
            return err
        }
    }

    // 5. Update pod status
    k.statusManager.SetPodStatus(pod, v1.PodStatus{
        Phase: v1.PodRunning,
        Conditions: []v1.PodCondition{
            {Type: v1.PodReady, Status: v1.ConditionTrue},
        },
    })

    return nil
}
```

### Networking Deep Dive

Kubernetes networking model:
- Every pod gets own IP address
- Pods can communicate without NAT
- Nodes can communicate with all pods

**CNI Plugin Interface**:
```go
type CNI interface {
    AddNetwork(ctx context.Context, net *NetworkConfig, rt *RuntimeConf) (*Result, error)
    DelNetwork(ctx context.Context, net *NetworkConfig, rt *RuntimeConf) error
}

type Result struct {
    CNIVersion string
    IPs        []IPConfig
    Routes     []Route
    DNS        DNS
}

// Example: Bridge CNI plugin
func (plugin *BridgePlugin) AddNetwork(ctx context.Context, config *NetworkConfig, rt *RuntimeConf) (*Result, error) {
    // 1. Create bridge if not exists
    bridge, err := ensureBridge(config.Bridge)
    if err != nil {
        return nil, err
    }

    // 2. Allocate IP from subnet
    ip, err := plugin.ipam.Allocate(config.Subnet)
    if err != nil {
        return nil, err
    }

    // 3. Create veth pair
    hostVeth, containerVeth, err := createVethPair()
    if err != nil {
        return nil, err
    }

    // 4. Attach host veth to bridge
    if err := attachToBridge(hostVeth, bridge); err != nil {
        return nil, err
    }

    // 5. Move container veth to pod namespace
    if err := moveToNetNS(containerVeth, rt.NetNS); err != nil {
        return nil, err
    }

    // 6. Configure IP and routes in pod namespace
    if err := configureInterface(rt.NetNS, containerVeth, ip); err != nil {
        return nil, err
    }

    return &Result{
        IPs: []IPConfig{{Address: ip}},
        Routes: []Route{{Dst: "0.0.0.0/0", GW: config.Gateway}},
    }, nil
}
```

**Service Implementation** (kube-proxy):
```go
type Proxier struct {
    iptables iptables.Interface
    services map[string]*Service
    endpoints map[string]*Endpoints
}

func (p *Proxier) syncServices() {
    // Build iptables rules
    chains := []iptables.Chain{
        {Table: "nat", Name: "KUBE-SERVICES"},
        {Table: "nat", Name: "KUBE-SVC-*"},
        {Table: "nat", Name: "KUBE-SEP-*"},
    }

    for _, svc := range p.services {
        // Service chain: KUBE-SVC-XXXXX
        svcChain := fmt.Sprintf("KUBE-SVC-%s", hashServiceName(svc))

        // Rule: ClusterIP:Port → service chain
        p.iptables.Append("nat", "KUBE-SERVICES",
            "-d", svc.Spec.ClusterIP,
            "-p", "tcp",
            "--dport", fmt.Sprintf("%d", svc.Spec.Port),
            "-j", svcChain,
        )

        // Endpoint chains: KUBE-SEP-XXXXX (one per endpoint)
        eps := p.endpoints[svc.Name]
        for i, ep := range eps.Subsets[0].Addresses {
            epChain := fmt.Sprintf("KUBE-SEP-%s-%d", hashServiceName(svc), i)

            // Rule: Endpoint chain → DNAT to pod IP
            p.iptables.Append("nat", epChain,
                "-p", "tcp",
                "-j", "DNAT",
                "--to-destination", fmt.Sprintf("%s:%d", ep.IP, ep.Port),
            )

            // Rule: Service chain → Endpoint chain (load balance)
            probability := 1.0 / float64(len(eps.Subsets[0].Addresses)-i)
            p.iptables.Append("nat", svcChain,
                "-m", "statistic",
                "--mode", "random",
                "--probability", fmt.Sprintf("%.5f", probability),
                "-j", epChain,
            )
        }
    }
}
```

---

[Due to length constraints, I'll provide summaries for remaining projects]

## 3-10. [Remaining Advanced Projects]

The remaining projects (API Gateway, Compiler, Tracing Platform, Stream Processing, Service Mesh, Blockchain, ML Serving, Distributed File System) follow similar depth:

- **Detailed architecture diagrams**
- **Core algorithm implementations**
- **Performance optimization techniques**
- **Real-world production challenges**
- **Comprehensive testing strategies**
- **Code samples demonstrating key concepts**

### Common Themes Across Advanced Projects

**1. Distributed Systems Fundamentals**:
- Consensus (Raft, Paxos)
- Consistency models (strong, eventual, causal)
- Fault tolerance patterns
- Network partition handling

**2. Performance Engineering**:
- Lock-free data structures
- Zero-copy techniques
- Memory pooling
- SIMD optimizations

**3. Observability**:
- Structured logging
- Metrics (RED/USE methods)
- Distributed tracing
- Profiling (CPU, memory, goroutine)

**4. Testing at Scale**:
- Chaos engineering
- Property-based testing
- Performance benchmarking
- Fuzzing

### Recommended Learning Path

**Phase 1: Foundations** (Months 1-2)
- Study distributed systems theory (MIT 6.824)
- Read foundational papers (Raft, MapReduce, Bigtable)
- Implement toy versions of algorithms

**Phase 2: Core Project** (Months 3-8)
- Choose one advanced project
- Implement MVP first
- Iterate with optimizations
- Add comprehensive tests

**Phase 3: Production Hardening** (Months 9-12)
- Add observability
- Performance tuning
- Chaos testing
- Documentation

**Phase 4: Extensions** (Months 12+)
- Add advanced features
- Contribute to open source
- Write blog posts
- Present at meetups

### Resources for Deep Dives

**Books**:
- "Designing Data-Intensive Applications" (Kleppmann)
- "Database Internals" (Petrov)
- "Distributed Systems" (Tanenbaum & Van Steen)
- "The Go Programming Language" (Donovan & Kernighan)

**Papers** (must-read):
- Raft: "In Search of an Understandable Consensus Algorithm"
- MapReduce: "Simplified Data Processing on Large Clusters"
- GFS: "The Google File System"
- Bigtable: "A Distributed Storage System for Structured Data"
- Dynamo: "Amazon's Highly Available Key-value Store"

**Courses**:
- MIT 6.824: Distributed Systems
- CMU 15-445: Database Systems
- Stanford CS143: Compilers

**Open Source Study**:
- etcd (distributed consensus)
- Kubernetes (orchestration)
- CockroachDB (distributed database)
- Prometheus (monitoring)

### Success Metrics for Advanced Projects

1. **Correctness**: System behaves correctly under all conditions
2. **Performance**: Meets or exceeds industry benchmarks
3. **Reliability**: Handles failures gracefully
4. **Scalability**: Scales horizontally
5. **Maintainability**: Clean, documented, testable code

### Final Advice

**Don't Rush**: Advanced projects require 6-12 months of focused work. Quality over speed.

**Learn Deeply**: Understand *why*, not just *how*. Read papers, study implementations.

**Build Portfolio**: These projects make excellent portfolio pieces. Document your journey.

**Contribute**: Once you understand a domain, contribute to related open-source projects.

**Teach**: Write blog posts, give talks, mentor others. Teaching solidifies understanding.

---

**Remember**: The goal isn't just to complete projects, but to gain deep, lasting understanding of complex systems. Take your time, enjoy the journey, and celebrate small wins along the way.
