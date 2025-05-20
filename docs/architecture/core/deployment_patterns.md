# Deployment Patterns

## Overview

The Blackhole platform supports multiple deployment patterns, from single-node development setups to distributed production deployments. All patterns use the same single binary that can run as either an orchestrator or a specific service subprocess.

## Single Binary Architecture

### Binary Modes

The Blackhole binary operates in different modes based on command arguments:

```bash
# Run as orchestrator (default)
./blackhole

# Run as specific service
./blackhole service --name identity
./blackhole service --name storage

# Run with specific configuration
./blackhole --config /etc/blackhole/config.yaml

# Development mode with hot reload
./blackhole --dev
```

### Binary Structure

```go
func main() {
    app := &cli.App{
        Name:    "blackhole",
        Version: version.String(),
        Commands: []*cli.Command{
            {
                Name:  "service",
                Usage: "Run as a specific service subprocess",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "name",
                        Required: true,
                        Usage:    "Service name to run",
                    },
                },
                Action: runService,
            },
            {
                Name:   "orchestrator",
                Usage:  "Run as the main orchestrator (default)",
                Action: runOrchestrator,
            },
        },
        DefaultCommand: "orchestrator",
    }
    
    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
```

## Deployment Patterns

### 1. Development Pattern

Single process with all services for development:

```yaml
# config/development.yaml
deployment:
  mode: development
  
services:
  identity:
    enabled: true
    mode: embedded    # Run in same process
    port: 50001
    
  storage:
    enabled: true
    mode: embedded
    port: 50002
    
  ledger:
    enabled: true
    mode: embedded
    port: 50003
    
  # Other services...
  
development:
  hot_reload: true
  debug_logging: true
  insecure_mode: true  # Skip TLS for local dev
```

Development startup:

```bash
# Start all services in one process
./blackhole --config config/development.yaml --dev

# Or with docker-compose
docker-compose -f docker-compose.dev.yml up
```

### 2. Single Node Production

All services as subprocesses on one machine:

```yaml
# config/single-node.yaml
deployment:
  mode: single_node
  
orchestrator:
  subprocess_management: true
  health_check_interval: 10s
  
services:
  identity:
    enabled: true
    mode: subprocess
    resources:
      memory: 1GB
      cpu: 0.5
    restart_policy:
      max_attempts: 5
      delay: 10s
      
  storage:
    enabled: true
    mode: subprocess
    resources:
      memory: 4GB
      cpu: 2.0
    restart_policy:
      max_attempts: 3
      delay: 30s
      
  # Other services...
```

Single node startup:

```bash
# Install binary
sudo cp blackhole /usr/local/bin/
sudo chmod +x /usr/local/bin/blackhole

# Create systemd service
sudo systemctl enable blackhole
sudo systemctl start blackhole

# Or run directly
sudo ./blackhole --config /etc/blackhole/config.yaml
```

### 3. Multi-Node Distributed

Services distributed across multiple nodes:

```yaml
# config/distributed.yaml
deployment:
  mode: distributed
  node_id: node-001
  cluster:
    coordinator: node-001
    nodes:
      - node-001
      - node-002
      - node-003
      
services:
  # Core services on primary node
  identity:
    enabled: true
    mode: subprocess
    placement: primary
    
  ledger:
    enabled: true
    mode: subprocess
    placement: primary
    
  # Storage distributed across nodes
  storage:
    enabled: true
    mode: subprocess
    placement: any
    replicas: 3
    
  # Analytics on dedicated node
  analytics:
    enabled: true
    mode: subprocess
    placement: node-003
```

Multi-node coordination:

```go
type DistributedOrchestrator struct {
    nodeID      string
    coordinator bool
    peers       map[string]*PeerNode
    services    map[string]ServicePlacement
}

type ServicePlacement struct {
    Service   string
    Nodes     []string
    Primary   string
    Replicas  int
    Strategy  PlacementStrategy
}

func (d *DistributedOrchestrator) PlaceService(service string) ([]string, error) {
    placement := d.services[service]
    
    switch placement.Strategy {
    case PrimaryOnly:
        return []string{placement.Primary}, nil
        
    case Replicated:
        return d.selectNodes(placement.Replicas), nil
        
    case LoadBalanced:
        return d.selectLeastLoadedNodes(placement.Replicas), nil
        
    default:
        return nil, fmt.Errorf("unknown placement strategy")
    }
}
```

### 4. Kubernetes Deployment

Running in Kubernetes with operators:

```yaml
# k8s/blackhole-operator.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: blackhole-system

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blackhole-operator
  namespace: blackhole-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: blackhole-operator
  template:
    metadata:
      labels:
        app: blackhole-operator
    spec:
      containers:
      - name: operator
        image: blackhole/operator:latest
        env:
        - name: OPERATOR_MODE
          value: kubernetes
```

Service deployment:

```yaml
# k8s/blackhole-services.yaml
apiVersion: blackhole.io/v1
kind: BlackholeNode
metadata:
  name: blackhole-node
  namespace: blackhole-system
spec:
  version: v1.0.0
  replicas: 3
  
  services:
    identity:
      enabled: true
      resources:
        memory: 1Gi
        cpu: 500m
      
    storage:
      enabled: true
      resources:
        memory: 4Gi
        cpu: 2000m
      persistence:
        size: 100Gi
        
    ledger:
      enabled: true
      resources:
        memory: 2Gi
        cpu: 1000m
```

### 5. Docker Swarm Deployment

Using Docker Swarm for orchestration:

```yaml
# docker-compose.swarm.yml
version: '3.8'

services:
  orchestrator:
    image: blackhole:latest
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager
    command: ["orchestrator"]
    networks:
      - blackhole
      
  identity:
    image: blackhole:latest
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 1G
          cpus: '0.5'
    command: ["service", "--name", "identity"]
    networks:
      - blackhole
      
  storage:
    image: blackhole:latest
    deploy:
      replicas: 3
      placement:
        max_replicas_per_node: 1
      resources:
        limits:
          memory: 4G
          cpus: '2.0'
    command: ["service", "--name", "storage"]
    volumes:
      - storage_data:/data
    networks:
      - blackhole

networks:
  blackhole:
    driver: overlay
    
volumes:
  storage_data:
```

### 6. Edge Deployment

Lightweight deployment for edge nodes:

```yaml
# config/edge.yaml
deployment:
  mode: edge
  resources:
    total_memory: 2GB
    total_cpu: 1.0
    
services:
  # Only essential services
  identity:
    enabled: true
    mode: subprocess
    resources:
      memory: 256MB
      cpu: 0.25
      
  storage:
    enabled: true
    mode: subprocess
    resources:
      memory: 512MB
      cpu: 0.5
    cache_only: true  # Don't store, just cache
    
  node:
    enabled: true
    mode: subprocess
    resources:
      memory: 256MB
      cpu: 0.25
    p2p:
      mode: light  # Light client mode
      
  # Disable heavy services
  analytics:
    enabled: false
  telemetry:
    enabled: false
```

## Deployment Configuration

### Environment Variables

```bash
# Common environment variables
export BLACKHOLE_CONFIG=/etc/blackhole/config.yaml
export BLACKHOLE_DATA_DIR=/var/lib/blackhole
export BLACKHOLE_LOG_DIR=/var/log/blackhole
export BLACKHOLE_NODE_ID=node-001
export BLACKHOLE_CLUSTER_NAME=production

# Service-specific
export BLACKHOLE_IDENTITY_PORT=50001
export BLACKHOLE_STORAGE_PORT=50002
export BLACKHOLE_LEDGER_PORT=50003

# Security
export BLACKHOLE_TLS_CERT=/etc/blackhole/certs/server.crt
export BLACKHOLE_TLS_KEY=/etc/blackhole/certs/server.key
export BLACKHOLE_CA_CERT=/etc/blackhole/certs/ca.crt
```

### Configuration Management

```go
type DeploymentConfig struct {
    Mode           DeploymentMode
    NodeID         string
    ClusterName    string
    ConfigSources  []ConfigSource
}

type ConfigSource interface {
    GetConfig(key string) (interface{}, error)
    WatchConfig(key string, callback func(interface{}))
}

// Configuration precedence (highest to lowest)
// 1. Command line flags
// 2. Environment variables
// 3. Config file
// 4. Remote config (etcd, consul)
// 5. Default values

func LoadConfig() (*Config, error) {
    config := &Config{}
    
    // Load from file
    if err := loadFromFile(config); err != nil {
        return nil, err
    }
    
    // Override with environment
    if err := loadFromEnv(config); err != nil {
        return nil, err
    }
    
    // Override with flags
    if err := loadFromFlags(config); err != nil {
        return nil, err
    }
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    return config, nil
}
```

### Secret Management

```go
type SecretManager struct {
    backend SecretBackend
}

type SecretBackend interface {
    GetSecret(key string) ([]byte, error)
    SetSecret(key string, value []byte) error
    DeleteSecret(key string) error
}

// Supported backends
type VaultBackend struct {
    client *vault.Client
}

type KubernetesSecretBackend struct {
    client kubernetes.Interface
}

type FileSecretBackend struct {
    dir string
}

func (m *SecretManager) GetDatabasePassword(service string) (string, error) {
    key := fmt.Sprintf("blackhole/%s/db_password", service)
    secret, err := m.backend.GetSecret(key)
    if err != nil {
        return "", fmt.Errorf("get secret: %w", err)
    }
    
    return string(secret), nil
}
```

## Infrastructure as Code

### Terraform Configuration

```hcl
# terraform/blackhole.tf
resource "aws_instance" "blackhole_node" {
  count         = var.node_count
  instance_type = var.instance_type
  ami           = data.aws_ami.blackhole.id
  
  tags = {
    Name    = "blackhole-node-${count.index}"
    Cluster = var.cluster_name
  }
  
  user_data = templatefile("${path.module}/userdata.sh", {
    node_id      = "node-${count.index}"
    cluster_name = var.cluster_name
    config_url   = var.config_url
  })
}

resource "aws_security_group" "blackhole" {
  name = "blackhole-cluster"
  
  ingress {
    description = "gRPC internal"
    from_port   = 50000
    to_port     = 50100
    protocol    = "tcp"
    self        = true
  }
  
  ingress {
    description = "P2P"
    from_port   = 4001
    to_port     = 4001
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
```

### Ansible Playbooks

```yaml
# ansible/deploy.yml
---
- name: Deploy Blackhole Node
  hosts: blackhole_nodes
  become: yes
  
  vars:
    blackhole_version: "{{ lookup('env', 'BLACKHOLE_VERSION') }}"
    
  tasks:
    - name: Install dependencies
      package:
        name:
          - ca-certificates
          - curl
        state: present
        
    - name: Download Blackhole binary
      get_url:
        url: "https://releases.blackhole.io/{{ blackhole_version }}/blackhole-linux-amd64"
        dest: /usr/local/bin/blackhole
        mode: '0755'
        
    - name: Create blackhole user
      user:
        name: blackhole
        system: yes
        shell: /bin/false
        
    - name: Create directories
      file:
        path: "{{ item }}"
        state: directory
        owner: blackhole
        group: blackhole
        mode: '0755'
      loop:
        - /etc/blackhole
        - /var/lib/blackhole
        - /var/log/blackhole
        
    - name: Deploy configuration
      template:
        src: config.yaml.j2
        dest: /etc/blackhole/config.yaml
        owner: blackhole
        group: blackhole
        mode: '0644'
      notify: restart blackhole
      
    - name: Deploy systemd service
      template:
        src: blackhole.service.j2
        dest: /etc/systemd/system/blackhole.service
      notify: restart blackhole
      
  handlers:
    - name: restart blackhole
      systemd:
        name: blackhole
        state: restarted
        daemon_reload: yes
        enabled: yes
```

## Monitoring and Observability

### Prometheus Integration

```yaml
# prometheus/blackhole.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'blackhole'
    static_configs:
      - targets: ['localhost:9090']
        labels:
          service: 'orchestrator'
          
      - targets: ['localhost:9091']
        labels:
          service: 'identity'
          
      - targets: ['localhost:9092']
        labels:
          service: 'storage'
```

### Grafana Dashboards

```json
{
  "dashboard": {
    "title": "Blackhole Node Metrics",
    "panels": [
      {
        "title": "Service Health",
        "type": "graph",
        "targets": [
          {
            "expr": "up{job='blackhole'}",
            "legendFormat": "{{service}}"
          }
        ]
      },
      {
        "title": "Process Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "process_resident_memory_bytes{job='blackhole'}",
            "legendFormat": "{{service}}"
          }
        ]
      },
      {
        "title": "gRPC Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(grpc_server_handled_total[5m])",
            "legendFormat": "{{service}}-{{method}}"
          }
        ]
      }
    ]
  }
}
```

## Backup and Recovery

### Backup Strategy

```go
type BackupManager struct {
    services []BackupableService
    storage  BackupStorage
}

type BackupableService interface {
    GetBackupData() (io.Reader, error)
    RestoreFromBackup(io.Reader) error
}

func (m *BackupManager) PerformBackup() error {
    backup := &Backup{
        Timestamp: time.Now(),
        Version:   version.String(),
        Services:  make(map[string][]byte),
    }
    
    for _, service := range m.services {
        data, err := service.GetBackupData()
        if err != nil {
            return fmt.Errorf("backup %s: %w", service.Name(), err)
        }
        
        // Compress and encrypt
        compressed := compress(data)
        encrypted := encrypt(compressed)
        
        backup.Services[service.Name()] = encrypted
    }
    
    return m.storage.Store(backup)
}
```

### Disaster Recovery

```yaml
# dr-config.yaml
disaster_recovery:
  backup:
    schedule: "0 2 * * *"  # Daily at 2 AM
    retention_days: 30
    storage:
      type: s3
      bucket: blackhole-backups
      region: us-east-1
      
  restore:
    strategy: incremental
    validation: true
    test_restore: weekly
    
  failover:
    automatic: false
    rpo_seconds: 300  # Recovery Point Objective
    rto_seconds: 900  # Recovery Time Objective
```

## Security Considerations

### TLS Configuration

```go
func CreateTLSConfig(service string) (*tls.Config, error) {
    // Load certificates
    cert, err := tls.LoadX509KeyPair(
        fmt.Sprintf("/etc/blackhole/certs/%s.crt", service),
        fmt.Sprintf("/etc/blackhole/certs/%s.key", service),
    )
    if err != nil {
        return nil, err
    }
    
    // Load CA
    caCert, err := ioutil.ReadFile("/etc/blackhole/certs/ca.crt")
    if err != nil {
        return nil, err
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientCAs:    caCertPool,
        RootCAs:      caCertPool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
        MinVersion:   tls.VersionTLS13,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }, nil
}
```

### Network Policies

```yaml
# Network segmentation
network_policies:
  internal:
    # Service-to-service communication
    allow:
      - from: identity
        to: storage
        ports: [50002]
      - from: storage
        to: ledger
        ports: [50003]
    deny:
      - from: analytics
        to: identity
        
  external:
    # External API access
    allow:
      - from: internet
        to: api_gateway
        ports: [443]
    deny_all_others: true
```

## Performance Tuning

### OS Tuning

```bash
#!/bin/bash
# System tuning script

# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Network tuning
sysctl -w net.core.somaxconn=65535
sysctl -w net.ipv4.tcp_mem="786432 1048576 26777216"
sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"

# Memory tuning
echo "vm.swappiness=10" >> /etc/sysctl.conf
echo "vm.dirty_ratio=15" >> /etc/sysctl.conf
echo "vm.dirty_background_ratio=5" >> /etc/sysctl.conf
```

### Service Tuning

```yaml
performance:
  grpc:
    max_concurrent_streams: 1000
    max_message_size: 104857600  # 100MB
    keepalive_time: 30s
    keepalive_timeout: 10s
    
  go_runtime:
    GOGC: 100
    GOMEMLIMIT: 80%  # Use 80% of container memory
    GOMAXPROCS: 0    # Use all available CPUs
    
  subprocess:
    nice_value: 0
    io_priority: 4
    scheduler: SCHED_NORMAL
```

## Deployment Checklist

### Pre-deployment

- [ ] Review system requirements
- [ ] Validate configuration files
- [ ] Check network connectivity
- [ ] Verify security certificates
- [ ] Test backup procedures
- [ ] Review resource allocations

### Deployment

- [ ] Deploy binary/container
- [ ] Apply configuration
- [ ] Start services
- [ ] Verify health checks
- [ ] Check logs for errors
- [ ] Test service connectivity

### Post-deployment

- [ ] Monitor metrics
- [ ] Set up alerts
- [ ] Document deployment
- [ ] Update runbooks
- [ ] Schedule maintenance
- [ ] Train operators

## Best Practices

1. **Start Small**: Begin with single-node deployment, then scale
2. **Monitor Everything**: Use comprehensive monitoring from day one
3. **Automate Deployment**: Use IaC and CI/CD pipelines
4. **Regular Backups**: Implement and test backup procedures
5. **Security First**: Always use TLS and proper authentication
6. **Resource Planning**: Plan for growth and peak usage
7. **Documentation**: Keep deployment docs current
8. **Testing**: Test deployments in staging first
9. **Rollback Plan**: Always have a rollback strategy
10. **Training**: Ensure team knows the deployment process

## Conclusion

The Blackhole platform's flexible deployment patterns enable:

- Simple development environments
- Robust production deployments
- Efficient resource utilization
- High availability configurations
- Easy scaling strategies

Choose the deployment pattern that best fits your requirements and scale as needed.