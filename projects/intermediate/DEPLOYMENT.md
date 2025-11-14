# Deployment Guide for Intermediate Go Projects

This guide provides practical, cost-effective deployment strategies for showcasing your intermediate Go projects to recruiters and on your portfolio site.

---

## Table of Contents
1. [Infrastructure Options](#infrastructure-options)
2. [General Deployment Strategy](#general-deployment-strategy)
3. [Project-Specific Deployments](#project-specific-deployments)
4. [Monitoring & Observability](#monitoring--observability)
5. [Portfolio Presentation](#portfolio-presentation)
6. [Recruiter Demo Strategies](#recruiter-demo-strategies)
7. [Cost Optimization](#cost-optimization)

---

## Infrastructure Options

### Recommended Budget-Friendly Providers

#### Option 1: DigitalOcean Droplets (Best Value)
**Recommended**: Basic Droplet ($6/month)
- 1 vCPU, 1GB RAM, 25GB SSD
- Sufficient for 2-3 intermediate projects
- Easy setup, great documentation
- **Portfolio URL**: `https://project-name.yourdomain.com`

```bash
# Initial setup
ssh root@your-droplet-ip
apt update && apt upgrade -y
apt install -y docker.io docker-compose nginx certbot python3-certbot-nginx
```

#### Option 2: Oracle Cloud Free Tier (FREE!)
**Recommended**: ARM-based Ampere A1
- 4 vCPUs, 24GB RAM (!!!)
- Always free tier
- Perfect for multiple projects
- Requires credit card but never charged

#### Option 3: AWS Lightsail ($5/month)
- Easy AWS entry point
- 1GB RAM, 1 vCPU
- Good for single project demos

#### Option 4: Linode ($5/month)
- Nanode: 1GB RAM
- Simple, reliable
- Good customer support

### Cost Comparison Table

| Provider | Cost | RAM | vCPU | Storage | Notes |
|----------|------|-----|------|---------|-------|
| Oracle Cloud | **FREE** | 24GB | 4 | 200GB | Best value, ARM |
| DigitalOcean | $6/mo | 1GB | 1 | 25GB | Easiest setup |
| AWS Lightsail | $5/mo | 1GB | 1 | 40GB | AWS ecosystem |
| Linode | $5/mo | 1GB | 1 | 25GB | Reliable |
| Hetzner | €4.5/mo | 4GB | 2 | 40GB | EU-based, great specs |

**💡 Pro Tip**: Use Oracle Cloud Free Tier for development/demos, DigitalOcean for production portfolio.

---

## General Deployment Strategy

### Architecture for Portfolio Projects

```
                           ┌─────────────────┐
                           │  Domain Name    │
                           │ your-project.io │
                           └────────┬────────┘
                                    │
                           ┌────────▼────────┐
                           │   Cloudflare    │
                           │  (Free CDN/SSL) │
                           └────────┬────────┘
                                    │
                      ┌─────────────▼─────────────┐
                      │     Nginx Reverse Proxy   │
                      │    (SSL Termination)      │
                      └─────────────┬─────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
    ┌───▼────┐              ┌───────▼────────┐         ┌───────▼────────┐
    │Project 1│              │   Project 2    │         │   Project 3    │
    │(Docker) │              │   (Docker)     │         │   (Docker)     │
    └─────────┘              └────────────────┘         └────────────────┘
```

### Standard Docker Compose Setup

Each project gets its own Docker Compose file:

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: mydb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### Nginx Configuration Template

```nginx
# /etc/nginx/sites-available/project1.conf
server {
    listen 80;
    server_name project1.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name project1.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/project1.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/project1.yourdomain.com/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req zone=api_limit burst=20 nodelay;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # Static files (if any)
    location /static/ {
        alias /var/www/project1/static/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### SSL Setup with Let's Encrypt

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d project1.yourdomain.com

# Auto-renewal (already set up by certbot)
# Test renewal
sudo certbot renew --dry-run
```

---

## Project-Specific Deployments

### 1. Distributed Task Queue System

**Demo Showcase**: Task processing dashboard with real-time stats

**Deployment Setup**:
```yaml
# docker-compose.yml
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis://redis:6379
      - POSTGRES_URL=postgres://user:pass@postgres:5432/taskqueue
    depends_on:
      - redis
      - postgres

  worker:
    build: .
    command: ["./worker"]
    deploy:
      replicas: 3
    environment:
      - REDIS_URL=redis://redis:6379
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: taskqueue
      POSTGRES_USER: user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  # Dashboard for demo
  dashboard:
    build: ./dashboard
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://api:8080

volumes:
  redis_data:
  postgres_data:
```

**Demo Features to Highlight**:
- Live task queue length
- Task processing rate (tasks/second)
- Success/failure metrics
- Worker health status
- Retry attempts visualization

**Portfolio Screenshot Ideas**:
1. Dashboard showing 1000+ tasks processed
2. Multiple workers running in parallel
3. Failed task retry mechanism
4. Task execution timeline

**Recruiter Demo Script** (2 minutes):
```
1. Show dashboard with live metrics
2. Submit 100 tasks via API:
   curl -X POST https://taskqueue.yourdomain.com/api/tasks/batch -d '{"count": 100}'
3. Watch workers process tasks in real-time
4. Show task details and retry logic
5. Demonstrate graceful worker shutdown
```

**Cost**: $6/month (DigitalOcean Basic Droplet)

---

### 2. Real-Time Chat Application with WebSockets

**Demo Showcase**: Live chat with multiple rooms and online users

**Deployment Setup**:
```yaml
services:
  chat-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis://redis:6379
      - POSTGRES_URL=postgres://user:pass@postgres:5432/chat
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - redis
      - postgres

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  postgres:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data

  # Simple web client
  web:
    build: ./web
    ports:
      - "3000:80"
    environment:
      - CHAT_API_URL=https://chat-api.yourdomain.com

volumes:
  redis_data:
  postgres_data:
```

**Demo Features to Highlight**:
- Real-time message delivery
- Online presence indicators
- Typing indicators
- Message history
- Multiple chat rooms
- File sharing
- Read receipts

**Portfolio Setup**:
Create 2-3 demo accounts and pre-populate with conversation history.

**Interactive Demo**:
Host at `https://chat.yourdomain.com` with:
- Guest login enabled
- Demo room with bot responses
- Stats page showing WebSocket connections

**Recruiter Demo Script** (2 minutes):
```
1. Open chat in two browser windows side-by-side
2. Show real-time message delivery
3. Demonstrate typing indicators
4. Upload and share a file
5. Show presence (online/offline) updates
6. Display chat history and search
```

**Recording Tips**:
- Use OBS to record screen
- Upload to YouTube as unlisted
- Embed video on portfolio site

**Cost**: $6/month

---

### 3. CLI Tool for Cloud Infrastructure Management

**Demo Showcase**: Terminal recording + deployed infrastructure

**Deployment Strategy**:
This project is a CLI tool, so deployment means:

1. **GitHub Releases** with binaries
```bash
# Use goreleaser for cross-platform builds
goreleaser release --snapshot --clean
```

2. **Demo Infrastructure** deployed to show it works
```bash
# Example: Deploy a simple 3-tier app using your CLI
$ cloudctl init aws --region us-east-1
$ cloudctl plan infra.yaml
$ cloudctl deploy infra.yaml
✓ Created VPC vpc-12345
✓ Created Subnet subnet-67890
✓ Created Instance i-abcdef
```

3. **Asciinema Recording**
```bash
# Record terminal session
asciinema rec demo.cast

# Convert to GIF for portfolio
docker run --rm -v $PWD:/data asciinema/asciicast2gif demo.cast demo.gif
```

**Portfolio Presentation**:
- Link to GitHub releases with download counts
- Embedded terminal recording on portfolio site
- "Infrastructure" page showing deployed resources
- Cost comparison vs Terraform

**Demo Features to Highlight**:
- Multi-cloud support (AWS + Azure)
- State management
- Dry-run/plan preview
- Resource drift detection
- Cost estimation

**Recruiter Demo Script**:
```
1. Show terminal recording of:
   - Init new project
   - Write infrastructure config
   - Preview changes (plan)
   - Deploy resources
   - Show deployed infrastructure in AWS console
   - Destroy resources

2. Show codebase structure and architecture
3. Explain key technical decisions
```

**Metrics to Show**:
- GitHub stars
- Download count
- Lines of code
- Supported resource types

**Cost**: $0 (just GitHub hosting) + ~$5/month for demo infra

---

### 4. GraphQL API with Code Generation

**Demo Showcase**: GraphQL Playground + performance metrics

**Deployment Setup**:
```yaml
services:
  graphql-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - POSTGRES_URL=postgres://user:pass@postgres:5432/graphql
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./seed.sql:/docker-entrypoint-initdb.d/seed.sql

  redis:
    image: redis:7-alpine

  # GraphQL Playground (for demo)
  playground:
    image: graphql/playground
    ports:
      - "3000:3000"
    environment:
      - GRAPHQL_ENDPOINT=https://graphql.yourdomain.com/query

volumes:
  postgres_data:
```

**Demo Features to Highlight**:
- GraphQL Playground for interactive queries
- DataLoader N+1 prevention
- Real-time subscriptions
- Cursor-based pagination
- Query complexity analysis
- Field-level caching

**Portfolio Presentation**:

1. **Interactive Demo** at `https://graphql.yourdomain.com/playground`

2. **Example Queries** page with benchmarks:
```graphql
# Query showing N+1 prevention
query EfficientQuery {
  posts(first: 100) {
    edges {
      node {
        title
        author {  # DataLoader batches these!
          name
          email
        }
        comments(first: 5) {
          totalCount
          edges {
            node {
              content
              author { name }
            }
          }
        }
      }
    }
  }
}

# Benchmark: 100 posts + authors = 2 DB queries (not 101!)
```

3. **Performance Dashboard**:
- Query latency (P50, P95, P99)
- DataLoader cache hit rate
- Queries per second
- Active subscriptions

**Recruiter Demo Script**:
```
1. Open GraphQL Playground
2. Run complex nested query
3. Show network tab: only 2 DB queries (DataLoader)
4. Demonstrate real-time subscription:
   - Open subscription in one tab
   - Create mutation in another tab
   - Show instant update
5. Show query complexity limit working
6. Explain code generation approach
```

**Cost**: $6/month

---

### 5. Metrics Aggregation and Monitoring Service

**Demo Showcase**: Live metrics ingestion + Grafana dashboards

**Deployment Setup**:
```yaml
services:
  metrics-api:
    build: .
    ports:
      - "8080:8080"
      - "8125:8125/udp"  # StatsD protocol
    environment:
      - INFLUXDB_URL=http://influxdb:8086
      - REDIS_URL=redis://redis:6379

  influxdb:
    image: influxdb:2.7-alpine
    ports:
      - "8086:8086"
    environment:
      - DOCKER_INFLUXDB_INIT_MODE=setup
      - DOCKER_INFLUXDB_INIT_USERNAME=admin
      - DOCKER_INFLUXDB_INIT_PASSWORD=${INFLUX_PASSWORD}
      - DOCKER_INFLUXDB_INIT_ORG=demo
      - DOCKER_INFLUXDB_INIT_BUCKET=metrics
    volumes:
      - influxdb_data:/var/lib/influxdb2

  redis:
    image: redis:7-alpine

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_INSTALL_PLUGINS=grafana-clock-panel
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources

  # Demo traffic generator
  load-generator:
    build: ./load-generator
    environment:
      - METRICS_URL=http://metrics-api:8080

volumes:
  influxdb_data:
  grafana_data:
```

**Demo Features to Highlight**:
- High-throughput metric ingestion
- Multiple protocol support (StatsD, HTTP, Prometheus)
- Real-time aggregation
- Grafana dashboards
- Alerting rules

**Portfolio Presentation**:

1. **Live Grafana Dashboard** at `https://metrics.yourdomain.com/grafana`
   - System metrics dashboard
   - Custom application metrics
   - Business KPIs visualization

2. **Load Test Results**:
```bash
# Show impressive numbers
✓ Ingested 10M metrics
✓ 50,000 metrics/second sustained
✓ P99 latency < 10ms
✓ Zero data loss
```

3. **Demo Metrics Generator**:
```bash
# Anyone can send test metrics
curl -X POST https://metrics.yourdomain.com/v1/metrics \
  -d '{"name": "test.counter", "value": 1, "tags": {"source": "demo"}}'
```

**Recruiter Demo Script**:
```
1. Show Grafana dashboard with live metrics
2. Run load generator:
   ./load-gen --rate 10000 --duration 60s
3. Watch metrics appear in real-time on dashboard
4. Show query API:
   curl 'metrics.yourdomain.com/query?metric=requests.count&range=1h'
5. Demonstrate aggregation (sum, avg, percentiles)
6. Show retention policies and downsampling
```

**Cost**: $12/month (need 2GB RAM for InfluxDB + Grafana)

---

### 6. Multi-Format File Converter Service

**Demo Showcase**: Web UI for file conversion + API

**Deployment Setup**:
```yaml
services:
  converter-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis://redis:6379
      - S3_BUCKET=${S3_BUCKET}
      - AWS_REGION=us-east-1
    volumes:
      - /tmp/conversions:/tmp/conversions
    deploy:
      resources:
        limits:
          memory: 1G

  worker:
    build: .
    command: ["./worker"]
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 2G
    environment:
      - REDIS_URL=redis://redis:6379
    volumes:
      - /tmp/conversions:/tmp/conversions

  redis:
    image: redis:7-alpine

  web-ui:
    build: ./web
    ports:
      - "3000:80"
    environment:
      - API_URL=https://converter-api.yourdomain.com

  # MinIO for file storage
  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_USER}
      - MINIO_ROOT_PASSWORD=${MINIO_PASSWORD}
    volumes:
      - minio_data:/data

volumes:
  minio_data:
```

**Demo Features to Highlight**:
- Multiple format support (show conversion matrix)
- Async job processing
- Progress tracking
- Download links with expiration
- Batch conversions

**Portfolio Presentation**:

1. **Interactive Web UI** at `https://convert.yourdomain.com`
   - Drag-and-drop file upload
   - Format selector
   - Real-time progress bar
   - Download converted file

2. **Supported Conversions Matrix**:
```
Images:  PNG ↔ JPG ↔ WebP ↔ GIF ↔ SVG
Docs:    DOCX ↔ PDF ↔ HTML ↔ Markdown
Video:   MP4 → WebM, AVI, GIF
Audio:   MP3 ↔ WAV ↔ OGG
```

3. **API Documentation** with curl examples:
```bash
# Upload and convert
curl -X POST https://convert.yourdomain.com/api/convert \
  -F "file=@image.png" \
  -F "format=webp" \
  -F "quality=80"
```

**Recruiter Demo Script**:
```
1. Open web UI in browser
2. Upload test image (PNG)
3. Select output format (WebP)
4. Show conversion progress
5. Download and compare file sizes
6. Demonstrate batch conversion (5+ files)
7. Show job history and status
8. Explain worker architecture and queue system
```

**Pre-populate Demo Data**:
- Keep last 50 conversions visible
- Show statistics (total conversions, popular formats)

**Cost**: $6/month + S3 ($1/month for storage)

---

### 7. OAuth2/OIDC Authentication Server

**Demo Showcase**: Working OAuth flow with demo apps

**Deployment Setup**:
```yaml
services:
  auth-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - POSTGRES_URL=postgres://user:pass@postgres:5432/auth
      - REDIS_URL=redis://redis:6379
      - JWT_PRIVATE_KEY=/keys/private.pem
      - JWT_PUBLIC_KEY=/keys/public.pem
    volumes:
      - ./keys:/keys:ro
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

  # Demo client app 1: Web application
  demo-web-app:
    build: ./demo-apps/web
    ports:
      - "3000:3000"
    environment:
      - OAUTH_ISSUER=https://auth.yourdomain.com
      - CLIENT_ID=demo-web
      - CLIENT_SECRET=${WEB_CLIENT_SECRET}

  # Demo client app 2: SPA
  demo-spa:
    build: ./demo-apps/spa
    ports:
      - "3001:80"

volumes:
  postgres_data:
```

**Demo Features to Highlight**:
- Full OAuth2 flows (authorization code, client credentials, refresh token)
- OIDC support with ID tokens
- PKCE for SPAs
- Consent screen
- Token introspection/revocation
- Social login integration (GitHub, Google)

**Portfolio Presentation**:

1. **Auth Server** at `https://auth.yourdomain.com`
   - Login page
   - Consent screen
   - Account management
   - Active sessions view

2. **Demo Applications**:
   - Web App: `https://demo-web.yourdomain.com`
   - SPA: `https://demo-spa.yourdomain.com`
   - Both use your auth server for login

3. **Developer Portal**:
   - Client registration
   - Token inspector
   - Flow visualizations
   - API documentation

**Recruiter Demo Script**:
```
1. Show demo web app (logged out)
2. Click "Login with OAuth"
3. Redirect to your auth server
4. Enter demo credentials
5. Show consent screen
6. Approve and redirect back
7. Display user info from ID token
8. Show token in developer tools
9. Demonstrate token refresh
10. Show logout and token revocation
```

**Technical Deep Dive for Interview**:
- Explain PKCE and why it's needed for SPAs
- Show JWT token structure and claims
- Discuss security measures (rate limiting, CSRF protection)
- Explain token rotation strategy

**Cost**: $6/month

---

### 8. Time Series Database Interface

**Demo Showcase**: Multi-database queries + performance comparison

**Deployment Setup**:
```yaml
services:
  tsdb-interface:
    build: .
    ports:
      - "8080:8080"
    environment:
      - INFLUXDB_URL=http://influxdb:8086
      - PROMETHEUS_URL=http://prometheus:9090
      - TIMESCALEDB_URL=postgres://user:pass@timescaledb:5432/tsdb

  influxdb:
    image: influxdb:2.7-alpine
    volumes:
      - influxdb_data:/var/lib/influxdb2

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  timescaledb:
    image: timescale/timescaledb:latest-pg15
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - timescaledb_data:/var/lib/postgresql/data

  # Query UI
  query-ui:
    build: ./ui
    ports:
      - "3000:80"

volumes:
  influxdb_data:
  prometheus_data:
  timescaledb_data:
```

**Demo Features to Highlight**:
- Unified query language across 3 databases
- Query translation and optimization
- Performance benchmarks
- Data federation
- Real-time query execution

**Portfolio Presentation**:

1. **Query Interface** at `https://tsdb.yourdomain.com`
   - SQL-like query editor with autocomplete
   - Result visualization (charts, tables)
   - Query explain plan
   - Performance metrics

2. **Comparison Dashboard**:
```
Query: Average CPU usage over 24 hours

InfluxDB:   Execution Time: 45ms
Prometheus: Execution Time: 120ms
TimescaleDB: Execution Time: 38ms

Your Interface: 55ms (with federation)
```

3. **Sample Queries**:
```sql
-- Your unified query language
SELECT
  time_bucket('5m', timestamp) AS time,
  avg(cpu_usage) AS avg_cpu
FROM metrics
WHERE
  host = 'web-server-1'
  AND timestamp > now() - interval '24 hours'
GROUP BY time
ORDER BY time;
```

**Recruiter Demo Script**:
```
1. Show query editor interface
2. Write simple query to get metrics
3. Execute against all 3 databases simultaneously
4. Compare execution times
5. Show query translation (original → database-specific)
6. Demonstrate data federation:
   - Join data from InfluxDB + TimescaleDB
7. Explain query optimization techniques
```

**Cost**: $12/month (need 2GB RAM for 3 databases)

---

### 9. Event-Driven Microservice with Message Broker

**Demo Showcase**: Event flow visualization + CQRS in action

**Deployment Setup**:
```yaml
services:
  # Command service
  command-service:
    build: ./services/command
    ports:
      - "8080:8080"
    environment:
      - KAFKA_BROKERS=kafka:9092
      - EVENT_STORE_URL=postgres://user:pass@eventstore:5432/events

  # Query service (read model)
  query-service:
    build: ./services/query
    ports:
      - "8081:8080"
    environment:
      - POSTGRES_URL=postgres://user:pass@querydb:5432/readmodel
      - KAFKA_BROKERS=kafka:9092

  # Event store
  eventstore:
    image: postgres:15-alpine
    volumes:
      - eventstore_data:/var/lib/postgresql/data

  # Read model database
  querydb:
    image: postgres:15-alpine
    volumes:
      - querydb_data:/var/lib/postgresql/data

  # Kafka
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    depends_on:
      - zookeeper
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1

  # Event flow visualizer
  visualizer:
    build: ./visualizer
    ports:
      - "3000:80"
    environment:
      - KAFKA_BROKERS=kafka:9092

volumes:
  eventstore_data:
  querydb_data:
```

**Demo Features to Highlight**:
- Event sourcing (append-only log)
- CQRS (separate read/write models)
- Event replay and projections
- Saga pattern for distributed transactions
- Event versioning

**Portfolio Presentation**:

1. **Interactive Event Flow Diagram** at `https://events.yourdomain.com`
   - Real-time event visualization
   - Event stream view
   - Service topology

2. **Demo Scenario: E-commerce Order**:
```
1. POST /api/orders (Command Service)
   ↓ Emits: OrderCreatedEvent

2. Events:
   - OrderCreatedEvent
   - InventoryReservedEvent
   - PaymentProcessedEvent
   - OrderConfirmedEvent

3. Projections Update:
   - Order read model
   - Inventory read model
   - Analytics model

4. Query /api/orders/{id} (Query Service)
   → Returns aggregated view from read model
```

3. **Event Store Browser**:
```json
{
  "aggregateId": "order-123",
  "events": [
    {"type": "OrderCreated", "version": 1, "data": {...}},
    {"type": "OrderPaid", "version": 2, "data": {...}},
    {"type": "OrderShipped", "version": 3, "data": {...}}
  ]
}
```

**Recruiter Demo Script**:
```
1. Show event visualizer (empty)
2. Create order via API:
   curl -X POST https://events.yourdomain.com/api/orders \
     -d '{"items": [...], "customer": {...}}'
3. Watch events flow through system in real-time
4. Show event store (append-only log)
5. Query read model for order status
6. Demonstrate event replay:
   - Delete read model
   - Replay events
   - Watch projections rebuild
7. Show how adding new projection doesn't require code changes
```

**Advanced Demo**:
- Simulate failure in payment service
- Show compensating transaction (Saga pattern)
- Demonstrate idempotency

**Cost**: $12/month (Kafka needs 2GB RAM)

---

### 10. Static Site Generator with Template Engine

**Demo Showcase**: Generated site + build performance

**Deployment Strategy**:

This is different—you're not deploying the generator, but:
1. **The generator itself** (CLI tool on GitHub)
2. **Example sites** built with it

**Portfolio Structure**:

1. **Generator Repository**: `https://github.com/yourname/static-gen`
   - README with quick start
   - Releases with binaries
   - Benchmarks vs Hugo/Jekyll

2. **Demo Sites** (static hosting, $0 cost):
   - Docs site: `https://docs.static-gen.io` (Cloudflare Pages)
   - Blog: `https://blog.example.com` (Netlify)
   - Landing page: `https://static-gen.io` (Vercel)

3. **Performance Benchmarks Page**:
```
Build Performance Comparison
----------------------------
1000 pages:

Hugo:       543ms
Jekyll:     12.3s
Your Gen:   892ms ⚡

10,000 pages:

Hugo:       4.2s
Jekyll:     124s
Your Gen:   6.8s ⚡
```

**Demo Site Setup**:

```yaml
# Example blog built with your generator
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Download static-gen
        run: |
          wget https://github.com/yourname/static-gen/releases/latest/download/static-gen-linux
          chmod +x static-gen-linux

      - name: Build site
        run: ./static-gen-linux build

      - name: Deploy to Cloudflare Pages
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CF_API_TOKEN }}
          accountId: ${{ secrets.CF_ACCOUNT_ID }}
          projectName: demo-blog
          directory: public
```

**Recruiter Demo Script**:
```
1. Show example sites built with your generator:
   - Personal blog
   - Documentation site
   - Company landing page

2. Demo the CLI:
   $ static-gen new my-blog
   $ cd my-blog
   $ static-gen dev    # Live reload server
   $ static-gen build  # Production build

3. Show template syntax and features:
   - Markdown with front matter
   - Template inheritance
   - Shortcodes
   - Syntax highlighting
   - Image optimization

4. Demonstrate hot reload (edit content, see instant changes)

5. Show build output:
   ✓ Built 100 pages in 892ms
   ✓ Optimized 50 images
   ✓ Generated sitemap
   ✓ Created RSS feed

6. Show GitHub stars and download statistics
```

**Portfolio Presentation**:
- Link to generator on GitHub
- Showcase 3-5 real sites built with it
- Performance comparison chart
- Feature comparison vs Hugo/Jekyll
- "Used by X developers" badge (from GitHub stars)

**Cost**: $0 (static hosting is free)

---

## Monitoring & Observability

### Lightweight Monitoring Stack

For portfolio projects, use lightweight but effective monitoring:

```yaml
# monitoring/docker-compose.yml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=7d'  # Keep 7 days only

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_INSTALL_PLUGINS=grafana-piechart-panel
    volumes:
      - grafana_data:/var/lib/grafana
      - ./dashboards:/etc/grafana/provisioning/dashboards

  # Lightweight log aggregation
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    volumes:
      - loki_data:/loki

  # Node exporter for system metrics
  node-exporter:
    image: prom/node-exporter:latest
    ports:
      - "9100:9100"

volumes:
  prometheus_data:
  grafana_data:
  loki_data:
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 30s
  evaluation_interval: 30s

scrape_configs:
  # Your Go applications
  - job_name: 'project1'
    static_configs:
      - targets: ['localhost:8080']

  - job_name: 'project2'
    static_configs:
      - targets: ['localhost:8081']

  # System metrics
  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']
```

### Application Instrumentation

```go
// Add to your Go applications
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Duration of HTTP requests",
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpDuration)
}

func main() {
    // Expose metrics endpoint
    http.Handle("/metrics", promhttp.Handler())

    // Your application routes
    http.HandleFunc("/api/...", handler)

    http.ListenAndServe(":8080", nil)
}
```

### Essential Grafana Dashboards

Create these dashboards and screenshot them for your portfolio:

1. **System Overview**:
   - CPU usage
   - Memory usage
   - Disk I/O
   - Network traffic

2. **Application Metrics**:
   - Requests per second
   - Response time (P50, P95, P99)
   - Error rate
   - Active connections

3. **Business Metrics** (project-specific):
   - Tasks processed (task queue)
   - Messages sent (chat app)
   - Files converted (file converter)
   - API calls (GraphQL)

---

## Portfolio Presentation

### Portfolio Website Structure

```
portfolio.yourdomain.com
├── /                           # Home with project grid
├── /projects/task-queue        # Individual project page
│   ├── Overview
│   ├── Live Demo Link
│   ├── GitHub Link
│   ├── Screenshots/Videos
│   ├── Architecture Diagram
│   ├── Tech Stack
│   └── Key Learnings
├── /projects/chat-app
├── /blog                       # Technical blog posts
│   ├── Building a Distributed Task Queue
│   └── Implementing Raft Consensus
└── /about                      # Your bio, skills, contact
```

### Individual Project Page Template

```markdown
# Distributed Task Queue System

> High-performance distributed task queue with Redis and PostgreSQL

[🚀 Live Demo](https://taskqueue.yourdomain.com) | [💻 GitHub](https://github.com/you/taskqueue) | [📊 Grafana](https://taskqueue.yourdomain.com/grafana)

## Overview
Brief description (2-3 sentences) of what the project does and why you built it.

## Key Features
- ✅ Feature 1
- ✅ Feature 2
- ✅ Feature 3

## Live Demo
[Embedded video or GIF showing the project in action]

**Try it yourself:**
```bash
curl -X POST https://taskqueue.yourdomain.com/api/tasks \
  -d '{"task": "demo", "data": {...}}'
```

## Architecture
[Architecture diagram]

### Tech Stack
- **Backend**: Go, Redis, PostgreSQL
- **Deployment**: Docker, Nginx, DigitalOcean
- **Monitoring**: Prometheus, Grafana

## Performance Metrics
- 50,000 tasks/second throughput
- <10ms median latency
- 99.99% success rate
- Zero data loss

## Implementation Highlights

### Challenge 1: Task Retry Logic
[Code snippet showing interesting implementation]

### Challenge 2: Worker Discovery
[Code snippet showing solution]

## Learnings
- Key insight 1
- Key insight 2
- What I'd do differently

## Screenshots
[Gallery of 3-4 screenshots]

## Links
- [GitHub Repository](...)
- [Technical Blog Post](...)
- [API Documentation](...)
```

### Screenshot Guidelines

**What to Capture**:

1. **Dashboard/UI** (main view):
   - Clean, professional appearance
   - Real data, not lorem ipsum
   - Highlight key features

2. **Metrics/Monitoring**:
   - Grafana dashboards showing impressive numbers
   - Real-time updates
   - Performance graphs

3. **Code Quality**:
   - Well-commented code snippet
   - Tests passing
   - Good architecture

4. **API Documentation**:
   - Swagger/OpenAPI docs
   - Example requests/responses

5. **Terminal/CLI** (if applicable):
   - Clean, colorized output
   - Successful execution
   - Use asciinema for recordings

**Tools**:
- Browser screenshots: Use browser dev tools full-page capture
- Terminal: asciinema → asciinema2gif
- Editing: Annotate with arrows/highlights using Excalidraw
- Compression: TinyPNG for web optimization

### Video Demos

**Recording Setup**:
1. Use OBS Studio (free, professional)
2. 1920x1080 resolution
3. 2-3 minutes max per project
4. Add voiceover explaining features

**Video Structure**:
```
0:00 - 0:15  Introduction (what it does)
0:15 - 1:00  Main feature demo
1:00 - 2:00  Technical highlights
2:00 - 2:30  Performance/metrics
2:30 - 3:00  Architecture overview
```

**Hosting**:
- YouTube (unlisted) for easy embedding
- Vimeo for portfolio-only access
- Self-host on portfolio site (costs bandwidth)

---

## Recruiter Demo Strategies

### Preparation Checklist

**Before the Interview**:
- [ ] All projects are running and accessible
- [ ] SSL certificates are valid
- [ ] Monitoring dashboards are populated with data
- [ ] Demo accounts are created (if applicable)
- [ ] Screenshots are up-to-date
- [ ] Repository README is polished
- [ ] You can explain each technical decision

### Live Demo Best Practices

**Do**:
✅ Start with the business value, then go technical
✅ Have your projects open in tabs before the call
✅ Prepare a demo script for each project (2 min)
✅ Show monitoring/metrics to prove it works at scale
✅ Explain trade-offs and what you'd improve
✅ Have code open in IDE to show architecture
✅ Reference specific challenges and solutions

**Don't**:
❌ Assume they'll navigate your site during the call
❌ Spend too long on one project
❌ Get lost in implementation details too early
❌ Make excuses if something breaks (have backup recordings)
❌ Forget to show tests and monitoring

### Demo Script Template

```
Project: [Name]
Time: 2 minutes

1. The Problem (15 sec)
   "In production systems, X is a common challenge because..."

2. Your Solution (15 sec)
   "I built this [type of system] that solves this by..."

3. Live Demo (60 sec)
   [Show the project working]
   - Key feature 1
   - Key feature 2
   - Impressive metric/result

4. Technical Highlight (20 sec)
   "The interesting technical challenge was [X], which I solved by [Y]"
   [Show code snippet if time permits]

5. Impact/Results (10 sec)
   "The result is [performance metric] which demonstrates [skill/principle]"
```

### Handling Common Questions

**"Walk me through your portfolio"**:
```
Strategy: Cover 3 projects in 5 minutes, offering to deep-dive on any.

"I have 10 projects demonstrating different aspects of backend engineering:

1. [Most Impressive Project] - This is a distributed database implementing
   Raft consensus. [30 sec demo]. Happy to deep-dive on consensus algorithms.

2. [Most Relevant to Job] - This GraphQL API shows my API design skills.
   [30 sec demo]. Used DataLoaders to solve N+1 problem.

3. [Demonstrates Range] - Built a complete OAuth2 server from scratch.
   [30 sec demo]. Implements all grant types and OIDC.

Which would you like to explore further?"
```

**"How did you deploy these?"**:
```
"All projects are deployed on DigitalOcean droplets using Docker Compose
for orchestration. I use Nginx as a reverse proxy with Let's Encrypt for
SSL. Monitoring is done with Prometheus and Grafana.

For the distributed database, I'm running a 3-node cluster to demonstrate
fault tolerance. [Show Grafana dashboard]

Total infrastructure cost is about $25/month for all projects, which shows
I'm conscious of cost optimization."
```

**"Can you show me the code?"**:
```
[Open GitHub repository]

"Let me show you the architecture:

- cmd/ contains the entry points
- internal/ has the core business logic
  - Here's the task queue implementation [open file]
  - This uses channels and goroutines for concurrency
- pkg/ contains reusable packages
- Tests are colocated with 85% coverage

The interesting part is this [specific file/function]:
[Explain a complex or clever solution]"
```

**"What would you do differently?"**:
```
Be honest and show growth mindset:

"If I rebuilt this today, I'd:
1. Use structured logging from the start (I added it later)
2. Implement circuit breakers for external dependencies
3. Add more comprehensive benchmarks earlier
4. Use Protocol Buffers instead of JSON for performance

These learnings informed my later projects, like [example]."
```

### Technical Deep Dive Preparation

For each project, prepare to discuss:

1. **Architecture Decisions**:
   - Why Go over other languages?
   - Why Redis vs other message brokers?
   - Why PostgreSQL vs other databases?

2. **Scalability**:
   - How does it scale horizontally?
   - What are the bottlenecks?
   - How would you handle 10x traffic?

3. **Reliability**:
   - How do you handle failures?
   - What monitoring is in place?
   - How do you ensure data consistency?

4. **Security**:
   - How is authentication handled?
   - How do you prevent common vulnerabilities?
   - Rate limiting strategy?

5. **Testing**:
   - Unit vs integration test strategy
   - How do you test distributed systems?
   - CI/CD pipeline

### Portfolio Analytics

Track and mention:
- GitHub stars and forks
- Project uptime (show 99.9%+)
- Response times from monitoring
- Number of API calls handled
- Total users (if applicable)

Example:
> "This project has been running for 6 months with 99.95% uptime,
> handled 10M+ API requests, and has 150 GitHub stars."

---

## Cost Optimization

### Single VM Multi-Project Setup

**Oracle Cloud Free Tier** (ARM 4vCPU, 24GB RAM):

```bash
# Can comfortably run:
- 5-7 intermediate projects
- 2-3 advanced projects
- Monitoring stack (Prometheus + Grafana)
- All with room to spare

Total Cost: $0/month 🎉
```

**Resource Allocation Strategy**:

```yaml
# docker-compose.yml with resource limits
services:
  project1:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M

  project2:
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
        reservations:
          memory: 512M
```

### Subdomain Strategy

Single VM, multiple projects:

```
projects.yourdomain.com     → Landing page
taskqueue.yourdomain.com    → Project 1
chat.yourdomain.com         → Project 2
graphql.yourdomain.com      → Project 3
...
```

All pointing to same IP, Nginx routes by subdomain.

### Cost Breakdown Examples

**Budget Setup ($0-6/month)**:
- Domain: $12/year = $1/month (Namecheap)
- Oracle Cloud Free Tier: $0/month
- **Total: $1/month**

**Standard Setup ($10-15/month)**:
- Domain: $1/month
- DigitalOcean Droplet (2GB): $12/month
- Object Storage (for file converter): $5/month
- **Total: $18/month**

**Professional Setup ($30-40/month)**:
- Domain: $1/month
- 2x DigitalOcean Droplets: $24/month
  - Droplet 1: Lightweight projects
  - Droplet 2: Resource-intensive projects
- Object Storage: $5/month
- Managed PostgreSQL (if needed): $15/month
- **Total: $45/month**

### Free Tier Strategies

**Cloudflare** (Free):
- CDN (speeds up your portfolio globally)
- SSL (alternative to Let's Encrypt)
- DDoS protection
- Analytics

**GitHub Pages** (Free):
- Host portfolio website
- Host documentation sites
- Unlimited bandwidth

**Netlify/Vercel** (Free):
- Deploy static sites
- Serverless functions (useful for demos)
- Preview deployments

---

## Final Deployment Checklist

Before showing to recruiters:

### Technical
- [ ] All projects are running and accessible
- [ ] SSL certificates are valid and auto-renewing
- [ ] Monitoring is set up and showing data
- [ ] Backups are configured (at least weekly)
- [ ] Logs are being collected (retain 7 days minimum)
- [ ] Health check endpoints return 200 OK
- [ ] No secrets in public repositories
- [ ] All dependencies are up to date
- [ ] Tests are passing and visible (CI badge)

### Content
- [ ] README files are comprehensive
- [ ] Architecture diagrams are clear
- [ ] Code is well-commented
- [ ] API documentation is available
- [ ] Demo accounts work (if applicable)
- [ ] Screenshots are up-to-date
- [ ] Videos are recorded and embedded
- [ ] Blog posts are written (optional but great)

### Portfolio Site
- [ ] All project links work
- [ ] Contact information is correct
- [ ] GitHub/LinkedIn links are present
- [ ] Resume is available for download
- [ ] Site is mobile-responsive
- [ ] Page load time < 3 seconds
- [ ] No lorem ipsum placeholder text
- [ ] Analytics are set up (optional)

### Demo Prep
- [ ] Demo scripts written for each project
- [ ] Practice demo (time yourself, 2 min per project)
- [ ] Screenshots ready if live demo fails
- [ ] Video recording as backup
- [ ] Know your talking points for each technical decision
- [ ] Prepare answers to common questions

---

## Maintenance

### Weekly Tasks
- Check that all projects are running
- Review monitoring for any issues
- Update SSL certificates if needed (should be automatic)

### Monthly Tasks
- Update dependencies (security patches)
- Review and clean up logs
- Check disk space usage
- Update screenshots if UI changed
- Review and respond to GitHub issues

### When Job Hunting
- Ensure 100% uptime during interview period
- Pre-populate projects with realistic demo data
- Test all demo accounts
- Have monitoring dashboards open during calls
- Keep infrastructure costs separate for tax purposes (if applicable)

---

## Conclusion

**Key Takeaways**:

1. **Start with Oracle Cloud Free Tier** - Best value for money (free!)
2. **Use Docker Compose** - Easy management of multiple projects
3. **Implement basic monitoring** - Shows professionalism
4. **Focus on 3-4 polished projects** - Better than 10 half-done projects
5. **Have live demos ready** - Video backups for when things break
6. **Prepare talking points** - Know why you made each technical decision
7. **Show, don't tell** - Live demos are worth 1000 words

**Most Impressive to Recruiters**:
1. Live, working projects (not just on localhost)
2. Monitoring dashboards with real metrics
3. Clean, professional UI/UX
4. Comprehensive documentation
5. Tests and CI/CD
6. Ability to explain technical trade-offs

**Remember**: A single well-deployed, well-documented project with live monitoring is more impressive than 10 projects that only run on localhost. Quality over quantity!

Good luck with your portfolio! 🚀
