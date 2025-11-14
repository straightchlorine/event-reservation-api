# Intermediate Go Projects - Detailed Descriptions

This document provides in-depth descriptions, technical details, and architectural considerations for each intermediate-level Go project.

---

## Table of Contents
1. [Distributed Task Queue System](#1-distributed-task-queue-system)
2. [Real-Time Chat Application with WebSockets](#2-real-time-chat-application-with-websockets)
3. [CLI Tool for Cloud Infrastructure Management](#3-cli-tool-for-cloud-infrastructure-management)
4. [GraphQL API with Code Generation](#4-graphql-api-with-code-generation)
5. [Metrics Aggregation and Monitoring Service](#5-metrics-aggregation-and-monitoring-service)
6. [Multi-Format File Converter Service](#6-multi-format-file-converter-service)
7. [OAuth2/OIDC Authentication Server](#7-oauth2oidc-authentication-server)
8. [Time Series Database Interface](#8-time-series-database-interface)
9. [Event-Driven Microservice with Message Broker](#9-event-driven-microservice-with-message-broker)
10. [Static Site Generator with Template Engine](#10-static-site-generator-with-template-engine)

---

## 1. Distributed Task Queue System

### Detailed Description

A distributed task queue system is fundamental infrastructure for modern applications, enabling asynchronous processing of time-consuming operations. This project will teach you how to build a production-grade background job processing system that can scale horizontally.

### Architecture Overview

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Client    │ ───────▶│  Task Broker │◀───────│   Worker    │
│ Application │         │  (Redis/MQ)  │         │   Pool      │
└─────────────┘         └──────────────┘         └─────────────┘
                              │                         │
                              ▼                         ▼
                        ┌──────────────┐         ┌─────────────┐
                        │  PostgreSQL  │         │   Result    │
                        │   (Metadata) │         │   Backend   │
                        └──────────────┘         └─────────────┘
```

### Key Components

#### 1. Task Definition and Serialization
Tasks need to be serialized for transmission over the network. You'll learn to handle:
- **Function Name**: String identifier for the task handler
- **Arguments**: Serialized payload (JSON, MessagePack, or Protocol Buffers)
- **Options**: Priority, retry count, timeout, ETA (estimated time of arrival)
- **Metadata**: Correlation ID, trace context, user context

Example task structure:
```go
type Task struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Args        json.RawMessage        `json:"args"`
    Priority    int                    `json:"priority"`
    MaxRetries  int                    `json:"max_retries"`
    Timeout     time.Duration          `json:"timeout"`
    ETA         *time.Time             `json:"eta,omitempty"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

#### 2. Message Broker Integration
The message broker is the heart of the system. You'll implement:
- **Push Operations**: Enqueue tasks to specific queues
- **Pop Operations**: Dequeue tasks with blocking or polling
- **Acknowledgment**: Confirm successful processing
- **Dead Letter Queue**: Handle permanently failed tasks
- **Priority Queues**: Multiple priority levels

#### 3. Worker Architecture
Workers are the execution engines. Key features:
- **Worker Pool**: Configurable number of goroutines
- **Task Registry**: Map of task names to handler functions
- **Graceful Shutdown**: Complete in-flight tasks before stopping
- **Heartbeat**: Regular signals to show worker liveness
- **Resource Limits**: CPU and memory constraints

#### 4. Retry Logic
Sophisticated retry mechanisms:
- **Exponential Backoff**: 2^n * base_delay
- **Jitter**: Random delay to avoid thundering herd
- **Max Attempts**: Configurable retry ceiling
- **Retry Predicates**: Retry based on error type

Example retry implementation:
```go
func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
    backoff := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
    jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
    return backoff + jitter
}
```

### Real-World Use Cases

1. **Email Sending**: Queue email tasks to avoid blocking HTTP requests
2. **Image Processing**: Resize/compress images asynchronously
3. **Report Generation**: Generate PDF reports in background
4. **Data Import**: Process large CSV/Excel files
5. **Webhooks**: Retry failed webhook deliveries
6. **Scheduled Tasks**: Cron-like periodic jobs
7. **ETL Pipelines**: Extract, transform, load data

### Technical Challenges

#### Challenge 1: Task Ordering
**Problem**: Maintaining FIFO order while supporting priorities
**Solution**: Use priority queues with timestamp tiebreakers

#### Challenge 2: Exactly-Once Processing
**Problem**: Ensuring tasks aren't processed multiple times
**Solution**: Idempotency keys and transaction-based dequeuing

#### Challenge 3: Worker Discovery
**Problem**: Tracking available workers in a distributed system
**Solution**: Redis-based heartbeat with TTL expiration

#### Challenge 4: Task Timeout
**Problem**: Preventing runaway tasks from blocking workers
**Solution**: Context with timeout and goroutine cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
defer cancel()

done := make(chan error, 1)
go func() {
    done <- executeTask(ctx, task)
}()

select {
case err := <-done:
    return err
case <-ctx.Done():
    return fmt.Errorf("task timeout exceeded")
}
```

### Performance Considerations

1. **Batching**: Group multiple Redis operations
2. **Connection Pooling**: Reuse database connections
3. **Pipeline Operations**: Use Redis pipelining
4. **Compression**: Compress large task payloads
5. **Prefetching**: Workers prefetch tasks to reduce latency

### Monitoring and Observability

Key metrics to track:
- Task enqueue rate
- Task completion rate
- Task failure rate
- Queue depth (backlog)
- Worker utilization
- Task latency (P50, P95, P99)
- Retry rate

### Extensions

- **Task Chaining**: Link tasks in workflows
- **Task Groups**: Wait for multiple tasks to complete
- **Canvas**: Create complex task graphs
- **Beat Scheduler**: Periodic task execution
- **Result Backend Expiration**: Auto-clean old results
- **Task Revocation**: Cancel pending tasks

---

## 2. Real-Time Chat Application with WebSockets

### Detailed Description

Real-time chat applications demonstrate bidirectional communication patterns and are excellent for learning WebSocket protocols, connection management at scale, and building reactive systems.

### Architecture Overview

```
┌──────────┐                    ┌────────────────┐
│  Client  │◀──────WebSocket───▶│  Chat Server   │
└──────────┘                    │   (Go/WS Hub)  │
                                └────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
            ┌──────────────┐    ┌──────────────┐   ┌──────────────┐
            │  PostgreSQL  │    │    Redis     │   │   MinIO/S3   │
            │  (Messages)  │    │  (Presence)  │   │    (Files)   │
            └──────────────┘    └──────────────┘   └──────────────┘
```

### Key Components

#### 1. WebSocket Hub
The hub manages all active connections:

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    rooms      map[string]*Room
    mutex      sync.RWMutex
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

#### 2. Connection Management
Each WebSocket connection needs careful handling:
- **Upgrading HTTP**: Transform HTTP requests to WebSocket
- **Ping/Pong**: Heartbeat to detect dead connections
- **Buffered Channels**: Prevent blocking on slow clients
- **Graceful Disconnect**: Clean up resources properly

#### 3. Message Types
Different message categories require different handling:

```go
type MessageType string

const (
    MessageTypeText     MessageType = "text"
    MessageTypeImage    MessageType = "image"
    MessageTypeFile     MessageType = "file"
    MessageTypeTyping   MessageType = "typing"
    MessageTypeRead     MessageType = "read"
    MessageTypeSystem   MessageType = "system"
)

type Message struct {
    ID        string      `json:"id"`
    Type      MessageType `json:"type"`
    RoomID    string      `json:"room_id"`
    UserID    string      `json:"user_id"`
    Content   string      `json:"content"`
    Timestamp time.Time   `json:"timestamp"`
    Metadata  interface{} `json:"metadata,omitempty"`
}
```

#### 4. Presence System
Track user online/offline status:
- **Redis Sets**: Store online users per room
- **TTL**: Auto-expire presence if heartbeat stops
- **Pub/Sub**: Broadcast presence changes
- **Last Seen**: Track last activity timestamp

```go
func (p *PresenceService) SetOnline(userID, roomID string) error {
    key := fmt.Sprintf("presence:%s", roomID)
    return p.redis.SAdd(ctx, key, userID).Err()
    // Set expiry: p.redis.Expire(ctx, key, 5*time.Minute)
}
```

#### 5. Message Persistence
Store messages for history and search:
- **Partitioning**: Partition by room ID for scalability
- **Indexing**: Index on timestamp and user ID
- **Pagination**: Cursor-based pagination for infinite scroll
- **Search**: Full-text search with PostgreSQL or Elasticsearch

### Real-World Features

#### Typing Indicators
Broadcast when users are typing:
```go
type TypingIndicator struct {
    UserID    string    `json:"user_id"`
    RoomID    string    `json:"room_id"`
    IsTyping  bool      `json:"is_typing"`
    Timestamp time.Time `json:"timestamp"`
}
```

Debounce typing events to avoid flooding:
- Send "typing" event on first keystroke
- Debounce subsequent keystrokes (500ms)
- Send "stopped typing" after 3 seconds of inactivity

#### Read Receipts
Track message read status:
```go
type ReadReceipt struct {
    MessageID  string    `json:"message_id"`
    UserID     string    `json:"user_id"`
    ReadAt     time.Time `json:"read_at"`
}
```

Implementation:
- Store last read message ID per user per room
- Display "seen by" indicators
- Sync read status across devices

#### File Sharing
Handle file uploads in chat:
1. Client uploads to `/upload` endpoint (multipart/form-data)
2. Server stores in S3/MinIO
3. Generate thumbnail for images
4. Create message with file metadata
5. Return presigned download URL

```go
type FileMessage struct {
    FileName    string `json:"file_name"`
    FileSize    int64  `json:"file_size"`
    MimeType    string `json:"mime_type"`
    DownloadURL string `json:"download_url"`
    ThumbnailURL string `json:"thumbnail_url,omitempty"`
}
```

### Technical Challenges

#### Challenge 1: Horizontal Scaling
**Problem**: Multiple server instances can't share WebSocket connections
**Solution**:
- Use Redis Pub/Sub to broadcast messages across servers
- Implement sticky sessions or connection routing
- Consider using a WebSocket gateway

```go
func (h *Hub) SubscribeToRedis() {
    pubsub := h.redis.Subscribe(ctx, "chat:broadcast")
    for msg := range pubsub.Channel() {
        // Broadcast to local connections
        h.broadcast <- []byte(msg.Payload)
    }
}
```

#### Challenge 2: Connection Limits
**Problem**: OS limits on open file descriptors
**Solution**:
- Increase ulimit settings
- Implement connection throttling
- Use connection pools efficiently

#### Challenge 3: Message Ordering
**Problem**: Messages may arrive out of order
**Solution**:
- Use sequence numbers
- Implement client-side reordering
- Use timestamp + sequence number

#### Challenge 4: Offline Message Delivery
**Problem**: Users may be offline when messages are sent
**Solution**:
- Store undelivered messages in database
- Implement push notifications
- Sync messages on reconnection

### Performance Optimization

1. **Message Batching**: Send multiple messages in one frame
2. **Compression**: Use WebSocket per-message compression
3. **Binary Protocol**: Use Protocol Buffers instead of JSON
4. **Connection Pooling**: Reuse database connections
5. **Caching**: Cache user profiles and room metadata

### Security Considerations

1. **Authentication**: Verify JWT on WebSocket upgrade
2. **Rate Limiting**: Prevent message flooding
3. **Input Validation**: Sanitize message content
4. **XSS Prevention**: Escape HTML in messages
5. **CORS**: Configure proper CORS headers
6. **File Upload Limits**: Restrict file size and types

### Monitoring Metrics

- Active WebSocket connections
- Messages per second
- Message delivery latency
- Connection duration
- Failed connection attempts
- Room occupancy distribution
- File upload success rate

---

## 3. CLI Tool for Cloud Infrastructure Management

### Detailed Description

Building a CLI tool for cloud infrastructure management teaches you how to interact with multiple cloud provider APIs, design user-friendly command interfaces, and implement Infrastructure as Code (IaC) principles.

### Architecture Overview

```
┌──────────────┐
│   CLI Tool   │
└──────────────┘
       │
       ├─────────────┬─────────────┬─────────────┐
       ▼             ▼             ▼             ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ AWS SDK  │  │Azure SDK │  │ GCP SDK  │  │  State   │
└──────────┘  └──────────┘  └──────────┘  │  Storage │
                                           └──────────┘
```

### Key Components

#### 1. Command Structure
Using Cobra framework for clean CLI design:

```go
// Root command
var rootCmd = &cobra.Command{
    Use:   "cloudctl",
    Short: "Cloud infrastructure management tool",
}

// Subcommands
var deployCmd = &cobra.Command{
    Use:   "deploy [config-file]",
    Short: "Deploy infrastructure",
    Args:  cobra.ExactArgs(1),
    Run:   runDeploy,
}

var destroyCmd = &cobra.Command{
    Use:   "destroy [config-file]",
    Short: "Destroy infrastructure",
    Run:   runDestroy,
}
```

Command tree:
```
cloudctl
├── init          # Initialize configuration
├── plan          # Preview changes
├── deploy        # Apply changes
├── destroy       # Tear down resources
├── list          # List resources
├── import        # Import existing resources
├── validate      # Validate configuration
└── state
    ├── show      # Show current state
    ├── pull      # Download state
    └── push      # Upload state
```

#### 2. Configuration Format
Design a declarative configuration language:

```yaml
# infrastructure.yaml
provider:
  name: aws
  region: us-east-1
  credentials:
    profile: default

resources:
  - type: vpc
    name: main-vpc
    properties:
      cidr: 10.0.0.0/16
      enable_dns: true

  - type: subnet
    name: public-subnet
    properties:
      vpc: ${vpc.main-vpc.id}
      cidr: 10.0.1.0/24
      availability_zone: us-east-1a

  - type: instance
    name: web-server
    properties:
      ami: ami-12345678
      instance_type: t3.micro
      subnet: ${subnet.public-subnet.id}
      tags:
        Name: WebServer
        Environment: production
```

#### 3. Provider Abstraction
Create a unified interface for all cloud providers:

```go
type Provider interface {
    CreateResource(ctx context.Context, resource *Resource) error
    UpdateResource(ctx context.Context, resource *Resource) error
    DeleteResource(ctx context.Context, resourceID string) error
    GetResource(ctx context.Context, resourceID string) (*Resource, error)
    ListResources(ctx context.Context, resourceType string) ([]*Resource, error)
}

type AWSProvider struct {
    ec2Client *ec2.Client
    s3Client  *s3.Client
    // ... other service clients
}

func (p *AWSProvider) CreateResource(ctx context.Context, resource *Resource) error {
    switch resource.Type {
    case "instance":
        return p.createEC2Instance(ctx, resource)
    case "s3_bucket":
        return p.createS3Bucket(ctx, resource)
    // ... other resource types
    }
}
```

#### 4. State Management
Track deployed infrastructure state:

```go
type State struct {
    Version   int                    `json:"version"`
    Resources map[string]ResourceState `json:"resources"`
    Outputs   map[string]interface{} `json:"outputs"`
    UpdatedAt time.Time              `json:"updated_at"`
}

type ResourceState struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`
    Attributes map[string]interface{} `json:"attributes"`
    Dependencies []string             `json:"dependencies"`
}
```

State storage options:
- **Local**: JSON file on disk
- **S3**: Store in S3 bucket with versioning
- **Azure Blob**: Store in Azure Storage
- **Remote Backend**: Custom HTTP backend

#### 5. Dependency Resolution
Build a dependency graph for correct resource ordering:

```go
type Graph struct {
    nodes map[string]*Node
    edges map[string][]string
}

type Node struct {
    Resource *Resource
    State    NodeState
}

func (g *Graph) TopologicalSort() ([]*Node, error) {
    var sorted []*Node
    visited := make(map[string]bool)
    visiting := make(map[string]bool)

    var visit func(id string) error
    visit = func(id string) error {
        if visited[id] {
            return nil
        }
        if visiting[id] {
            return fmt.Errorf("circular dependency detected")
        }

        visiting[id] = true
        for _, depID := range g.edges[id] {
            if err := visit(depID); err != nil {
                return err
            }
        }
        visiting[id] = false
        visited[id] = true
        sorted = append(sorted, g.nodes[id])
        return nil
    }

    for id := range g.nodes {
        if err := visit(id); err != nil {
            return nil, err
        }
    }

    return sorted, nil
}
```

### Real-World Features

#### Resource Drift Detection
Detect when actual infrastructure differs from state:

```go
func detectDrift(resource *Resource, state *ResourceState) (*Drift, error) {
    actual, err := provider.GetResource(ctx, state.ID)
    if err != nil {
        return nil, err
    }

    drift := &Drift{
        ResourceID: state.ID,
        Changes:    make(map[string]Change),
    }

    for key, expectedValue := range state.Attributes {
        actualValue := actual.Attributes[key]
        if !reflect.DeepEqual(expectedValue, actualValue) {
            drift.Changes[key] = Change{
                Expected: expectedValue,
                Actual:   actualValue,
            }
        }
    }

    return drift, nil
}
```

#### Cost Estimation
Estimate costs before deployment:

```go
type CostEstimator interface {
    EstimateResource(resource *Resource) (float64, error)
}

type AWSCostEstimator struct {
    pricingClient *pricing.Client
}

func (e *AWSCostEstimator) EstimateResource(resource *Resource) (float64, error) {
    switch resource.Type {
    case "instance":
        instanceType := resource.Properties["instance_type"].(string)
        return e.getEC2Pricing(instanceType)
    // ... other resource types
    }
}
```

#### Resource Import
Import existing infrastructure:

```bash
$ cloudctl import aws_instance web-server i-1234567890abcdef0
```

Implementation:
1. Query cloud provider for resource details
2. Generate configuration file
3. Add to state file
4. Verify import with `plan` command

### Technical Challenges

#### Challenge 1: Idempotency
**Problem**: Ensuring operations are idempotent
**Solution**:
- Check if resource exists before creation
- Use update operations when appropriate
- Implement proper error handling

#### Challenge 2: Parallel Execution
**Problem**: Creating independent resources in parallel
**Solution**:
- Analyze dependency graph
- Create worker pool
- Execute independent resources concurrently

```go
func deployResources(resources []*Resource) error {
    graph := buildDependencyGraph(resources)
    levels := graph.GetLevels() // Group by dependency depth

    for _, level := range levels {
        var wg sync.WaitGroup
        errChan := make(chan error, len(level))

        for _, resource := range level {
            wg.Add(1)
            go func(r *Resource) {
                defer wg.Done()
                if err := provider.CreateResource(ctx, r); err != nil {
                    errChan <- err
                }
            }(resource)
        }

        wg.Wait()
        close(errChan)

        for err := range errChan {
            return err // Fail fast
        }
    }

    return nil
}
```

#### Challenge 3: Credential Management
**Problem**: Securely managing cloud credentials
**Solution**:
- Support multiple credential sources (environment, config file, IAM roles)
- Use credential chains
- Never store credentials in state files

### Performance Considerations

1. **Caching**: Cache provider API responses
2. **Batch Operations**: Group API calls when possible
3. **Parallel Execution**: Deploy independent resources concurrently
4. **Incremental Updates**: Only update changed resources
5. **State Compression**: Compress large state files

### Testing Strategy

1. **Unit Tests**: Test parsers, dependency resolution
2. **Integration Tests**: Use cloud provider sandboxes/localstack
3. **E2E Tests**: Deploy real infrastructure in test accounts
4. **Mock Providers**: Create mock providers for fast testing

---

## 4. GraphQL API with Code Generation

### Detailed Description

GraphQL provides a flexible and efficient alternative to REST APIs. This project teaches you about type-safe API development, code generation, and solving the N+1 query problem.

### Architecture Overview

```
┌──────────────┐
│GraphQL Schema│
└──────────────┘
       │
       ▼
┌──────────────┐     ┌──────────────┐
│   gqlgen     │────▶│  Generated   │
│ (Generator)  │     │   Resolvers  │
└──────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  DataLoaders │
                     └──────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  ┌──────────┐       ┌──────────┐       ┌──────────┐
  │PostgreSQL│       │ MongoDB  │       │  Redis   │
  └──────────┘       └──────────┘       └──────────┘
```

### Key Components

#### 1. Schema Design
GraphQL schema defines your API:

```graphql
type User {
  id: ID!
  username: String!
  email: String!
  posts: [Post!]!
  createdAt: Time!
}

type Post {
  id: ID!
  title: String!
  content: String!
  author: User!
  comments: [Comment!]!
  tags: [String!]!
  publishedAt: Time
}

type Comment {
  id: ID!
  content: String!
  author: User!
  post: Post!
  createdAt: Time!
}

type Query {
  user(id: ID!): User
  users(first: Int, after: String): UserConnection!
  post(id: ID!): Post
  posts(filter: PostFilter, orderBy: PostOrder, first: Int, after: String): PostConnection!
}

type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!
  createPost(input: CreatePostInput!): Post!
  addComment(postID: ID!, content: String!): Comment!
}

type Subscription {
  postAdded: Post!
  commentAdded(postID: ID!): Comment!
}

input CreateUserInput {
  username: String!
  email: String!
  password: String!
}

input PostFilter {
  authorID: ID
  tags: [String!]
  publishedAfter: Time
}

enum PostOrder {
  CREATED_AT_ASC
  CREATED_AT_DESC
  TITLE_ASC
  TITLE_DESC
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

#### 2. Code Generation with gqlgen
gqlgen generates type-safe code:

```go
//go:generate go run github.com/99designs/gqlgen generate

// Generated resolver interface
type Resolver interface {
    Query() QueryResolver
    Mutation() MutationResolver
    Subscription() SubscriptionResolver
    User() UserResolver
    Post() PostResolver
}

// You implement these interfaces
type queryResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.UserService.GetByID(ctx, id)
}

func (r *queryResolver) Posts(
    ctx context.Context,
    filter *model.PostFilter,
    orderBy *model.PostOrder,
    first *int,
    after *string,
) (*model.PostConnection, error) {
    return r.PostService.List(ctx, filter, orderBy, first, after)
}
```

#### 3. DataLoader Pattern
Solve N+1 query problem:

```go
// Without DataLoader (N+1 problem)
func (r *postResolver) Author(ctx context.Context, post *model.Post) (*model.User, error) {
    // This query runs for EACH post!
    return r.UserService.GetByID(ctx, post.AuthorID)
}

// With DataLoader
type UserLoader struct {
    db *sql.DB
}

func (l *UserLoader) Load(ctx context.Context, keys []string) ([]*model.User, []error) {
    // Single query for all user IDs
    query := "SELECT * FROM users WHERE id = ANY($1)"
    rows, err := l.db.QueryContext(ctx, query, pq.Array(keys))
    if err != nil {
        return nil, []error{err}
    }
    defer rows.Close()

    userMap := make(map[string]*model.User)
    for rows.Next() {
        var user model.User
        if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
            return nil, []error{err}
        }
        userMap[user.ID] = &user
    }

    // Return users in same order as keys
    users := make([]*model.User, len(keys))
    errs := make([]error, len(keys))
    for i, key := range keys {
        users[i] = userMap[key]
        if users[i] == nil {
            errs[i] = fmt.Errorf("user not found: %s", key)
        }
    }

    return users, errs
}

// Using DataLoader in resolver
func (r *postResolver) Author(ctx context.Context, post *model.Post) (*model.User, error) {
    loader := middleware.GetUserLoader(ctx)
    return loader.Load(ctx, post.AuthorID)
}
```

DataLoader batches and caches requests:
- **Batching**: Collects multiple Load() calls within a tick
- **Caching**: Caches results for the request lifetime
- **Deduplication**: Same key requested multiple times = one DB query

#### 4. Cursor-Based Pagination
Implement relay-style pagination:

```go
func (s *PostService) List(
    ctx context.Context,
    filter *model.PostFilter,
    orderBy *model.PostOrder,
    first *int,
    after *string,
) (*model.PostConnection, error) {
    limit := 10
    if first != nil {
        limit = *first + 1 // +1 to check if there's a next page
    }

    var cursor string
    if after != nil {
        cursor = *after
    }

    query := buildQuery(filter, orderBy, cursor, limit)
    posts, err := s.db.QueryPosts(ctx, query)
    if err != nil {
        return nil, err
    }

    hasNextPage := len(posts) > limit
    if hasNextPage {
        posts = posts[:limit]
    }

    edges := make([]*model.PostEdge, len(posts))
    for i, post := range posts {
        edges[i] = &model.PostEdge{
            Node:   post,
            Cursor: encodeCursor(post.ID, post.CreatedAt),
        }
    }

    var startCursor, endCursor *string
    if len(edges) > 0 {
        startCursor = &edges[0].Cursor
        endCursor = &edges[len(edges)-1].Cursor
    }

    return &model.PostConnection{
        Edges: edges,
        PageInfo: &model.PageInfo{
            HasNextPage:     hasNextPage,
            HasPreviousPage: after != nil,
            StartCursor:     startCursor,
            EndCursor:       endCursor,
        },
        TotalCount: s.getTotalCount(ctx, filter),
    }, nil
}

func encodeCursor(id string, createdAt time.Time) string {
    data := fmt.Sprintf("%s:%d", id, createdAt.Unix())
    return base64.StdEncoding.EncodeToString([]byte(data))
}
```

#### 5. GraphQL Subscriptions
Real-time updates over WebSocket:

```go
func (r *subscriptionResolver) PostAdded(ctx context.Context) (<-chan *model.Post, error) {
    posts := make(chan *model.Post, 1)

    // Subscribe to Redis pub/sub
    pubsub := r.Redis.Subscribe(ctx, "post:added")

    go func() {
        defer close(posts)
        defer pubsub.Close()

        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-pubsub.Channel():
                var post model.Post
                if err := json.Unmarshal([]byte(msg.Payload), &post); err != nil {
                    continue
                }
                posts <- &post
            }
        }
    }()

    return posts, nil
}

// In mutation resolver
func (r *mutationResolver) CreatePost(ctx context.Context, input model.CreatePostInput) (*model.Post, error) {
    post, err := r.PostService.Create(ctx, input)
    if err != nil {
        return nil, err
    }

    // Publish to subscribers
    data, _ := json.Marshal(post)
    r.Redis.Publish(ctx, "post:added", data)

    return post, nil
}
```

### Real-World Features

#### Field-Level Authorization
```go
directive @auth(requires: Role = USER) on FIELD_DEFINITION

enum Role {
  USER
  ADMIN
}

type Mutation {
  deleteUser(id: ID!): Boolean! @auth(requires: ADMIN)
  createPost(input: CreatePostInput!): Post! @auth(requires: USER)
}

// Directive implementation
func AuthDirective(ctx context.Context, obj interface{}, next graphql.Resolver, requires model.Role) (interface{}, error) {
    user := middleware.GetUser(ctx)
    if user == nil {
        return nil, fmt.Errorf("unauthorized")
    }

    if user.Role < requires {
        return nil, fmt.Errorf("insufficient permissions")
    }

    return next(ctx)
}
```

#### Query Complexity Analysis
Prevent expensive queries:

```go
func (r *Resolver) Complexity() graphql.ComplexityRoot {
    return graphql.ComplexityRoot{
        Query: graphql.QueryComplexityRoot{
            Users: func(childComplexity int, first *int, after *string) int {
                limit := 10
                if first != nil {
                    limit = *first
                }
                return childComplexity * limit
            },
        },
        User: graphql.UserComplexityRoot{
            Posts: func(childComplexity int) int {
                return childComplexity * 10 // Assume 10 posts per user
            },
        },
    }
}

// In server setup
srv.Use(extension.FixedComplexityLimit(1000)) // Max complexity = 1000
```

---

## 5-10. [Continued in next section due to length...]

### Summary of Remaining Projects

The remaining intermediate projects (5-10) cover:

- **Metrics Aggregation Service**: High-throughput data ingestion, time-series storage
- **File Converter Service**: Multi-format conversion, external process management
- **OAuth2/OIDC Server**: Authentication protocols, token management, security
- **Time Series DB Interface**: Query language design, data federation
- **Event-Driven Microservice**: Event sourcing, CQRS, saga patterns
- **Static Site Generator**: Template engines, build pipelines, plugin systems

Each project provides deep technical challenges and exposure to different domains, from distributed systems to security to developer tooling.

### Project Selection Guide

Choose projects based on your interests:
- **Backend Systems**: Projects 1, 5, 9
- **Real-Time Applications**: Projects 2, 8
- **Developer Tools**: Projects 3, 10
- **API Design**: Projects 4, 7
- **Data Processing**: Projects 6, 8

### Learning Path Recommendation

1. Start with Project 2 (Chat) or 4 (GraphQL) for immediate results
2. Move to Project 1 (Task Queue) or 9 (Event-Driven) for distributed systems
3. Tackle Project 3 (CLI) or 10 (Static Site) for tooling experience
4. Finish with Project 7 (OAuth2) or 8 (Time Series) for specialized knowledge

Remember: Deep understanding beats surface-level completion. Take time to explore each concept thoroughly.
