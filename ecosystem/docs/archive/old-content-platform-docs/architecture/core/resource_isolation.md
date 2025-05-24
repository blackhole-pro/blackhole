# Resource Isolation Architecture

## Overview

Resource isolation in the Blackhole platform ensures that each service subprocess operates within defined boundaries, preventing resource starvation and maintaining system stability. The orchestrator enforces OS-level isolation using cgroups, resource limits, and process priorities.

## Isolation Mechanisms

### Process-Level Isolation

Each service runs as a separate OS process with strict boundaries:

```go
type ProcessIsolation struct {
    PID              int
    ServiceName      string
    ResourceLimits   ResourceLimits
    SecurityContext  SecurityContext
    NamespaceConfig  NamespaceConfig
}

type ResourceLimits struct {
    CPU      CPULimits
    Memory   MemoryLimits
    IO       IOLimits
    Network  NetworkLimits
    FileDesc FDLimits
}
```

### Cgroup Configuration

The orchestrator uses cgroups v2 for resource enforcement:

```go
type CgroupManager struct {
    rootPath     string
    controllers  []string
    services     map[string]*ServiceCgroup
}

func (m *CgroupManager) CreateServiceCgroup(service string, pid int) error {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Create cgroup hierarchy
    if err := os.MkdirAll(cgroupPath, 0755); err != nil {
        return err
    }
    
    // Add process to cgroup
    if err := ioutil.WriteFile(
        filepath.Join(cgroupPath, "cgroup.procs"),
        []byte(strconv.Itoa(pid)),
        0644,
    ); err != nil {
        return err
    }
    
    return nil
}
```

## CPU Isolation

### CPU Quotas

Enforce CPU usage limits per service:

```go
type CPULimits struct {
    Quota    float64  // CPU cores (e.g., 1.5 = 150%)
    Period   int64    // Scheduling period in microseconds
    Shares   int      // Relative weight for CPU scheduling
    CPUSet   string   // CPU affinity (specific cores)
}

func (m *CgroupManager) SetCPULimits(service string, limits CPULimits) error {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Set CPU quota (e.g., 1.5 cores = 150000 microseconds per 100000)
    quotaMicros := int64(limits.Quota * 100000)
    if err := ioutil.WriteFile(
        filepath.Join(cgroupPath, "cpu.max"),
        []byte(fmt.Sprintf("%d %d", quotaMicros, limits.Period)),
        0644,
    ); err != nil {
        return err
    }
    
    // Set CPU shares
    if err := ioutil.WriteFile(
        filepath.Join(cgroupPath, "cpu.weight"),
        []byte(strconv.Itoa(limits.Shares)),
        0644,
    ); err != nil {
        return err
    }
    
    // Set CPU affinity if specified
    if limits.CPUSet != "" {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "cpuset.cpus"),
            []byte(limits.CPUSet),
            0644,
        ); err != nil {
            return err
        }
    }
    
    return nil
}
```

### CPU Monitoring

Track CPU usage per service:

```go
func (m *CgroupManager) GetCPUStats(service string) (*CPUStats, error) {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Read CPU statistics
    statData, err := ioutil.ReadFile(filepath.Join(cgroupPath, "cpu.stat"))
    if err != nil {
        return nil, err
    }
    
    stats := &CPUStats{}
    scanner := bufio.NewScanner(bytes.NewReader(statData))
    for scanner.Scan() {
        fields := strings.Fields(scanner.Text())
        if len(fields) != 2 {
            continue
        }
        
        value, _ := strconv.ParseInt(fields[1], 10, 64)
        switch fields[0] {
        case "usage_usec":
            stats.UsageNanos = value * 1000
        case "user_usec":
            stats.UserNanos = value * 1000
        case "system_usec":
            stats.SystemNanos = value * 1000
        case "nr_throttled":
            stats.ThrottledCount = value
        case "throttled_usec":
            stats.ThrottledNanos = value * 1000
        }
    }
    
    return stats, nil
}
```

## Memory Isolation

### Memory Limits

Enforce memory usage boundaries:

```go
type MemoryLimits struct {
    Limit      int64  // Hard memory limit in bytes
    Soft       int64  // Soft limit (memory.high)
    Swap       int64  // Swap limit
    Kernel     int64  // Kernel memory limit
    OOMKillDisable bool // Disable OOM killer
}

func (m *CgroupManager) SetMemoryLimits(service string, limits MemoryLimits) error {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Set hard memory limit
    if err := ioutil.WriteFile(
        filepath.Join(cgroupPath, "memory.max"),
        []byte(strconv.FormatInt(limits.Limit, 10)),
        0644,
    ); err != nil {
        return err
    }
    
    // Set soft memory limit
    if limits.Soft > 0 {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "memory.high"),
            []byte(strconv.FormatInt(limits.Soft, 10)),
            0644,
        ); err != nil {
            return err
        }
    }
    
    // Set swap limit
    if err := ioutil.WriteFile(
        filepath.Join(cgroupPath, "memory.swap.max"),
        []byte(strconv.FormatInt(limits.Swap, 10)),
        0644,
    ); err != nil {
        return err
    }
    
    // Configure OOM behavior
    if limits.OOMKillDisable {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "memory.oom_control"),
            []byte("1"),
            0644,
        ); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Memory Monitoring

Track memory usage and pressure:

```go
type MemoryStats struct {
    Current    int64
    Peak       int64
    Swap       int64
    Cache      int64
    RSS        int64
    Pressure   PressureStats
}

func (m *CgroupManager) GetMemoryStats(service string) (*MemoryStats, error) {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Read current memory usage
    current, err := readInt64(filepath.Join(cgroupPath, "memory.current"))
    if err != nil {
        return nil, err
    }
    
    // Read detailed memory statistics
    statData, err := ioutil.ReadFile(filepath.Join(cgroupPath, "memory.stat"))
    if err != nil {
        return nil, err
    }
    
    stats := &MemoryStats{Current: current}
    scanner := bufio.NewScanner(bytes.NewReader(statData))
    for scanner.Scan() {
        fields := strings.Fields(scanner.Text())
        if len(fields) != 2 {
            continue
        }
        
        value, _ := strconv.ParseInt(fields[1], 10, 64)
        switch fields[0] {
        case "anon":
            stats.RSS = value
        case "file":
            stats.Cache = value
        case "peak":
            stats.Peak = value
        case "swap":
            stats.Swap = value
        }
    }
    
    // Read memory pressure
    pressure, err := readPressureStats(filepath.Join(cgroupPath, "memory.pressure"))
    if err == nil {
        stats.Pressure = pressure
    }
    
    return stats, nil
}
```

## I/O Isolation

### I/O Limits

Control disk I/O per service:

```go
type IOLimits struct {
    ReadBPS    int64  // Read bytes per second
    WriteBPS   int64  // Write bytes per second
    ReadIOPS   int64  // Read operations per second
    WriteIOPS  int64  // Write operations per second
    Priority   int    // I/O priority class
}

func (m *CgroupManager) SetIOLimits(service string, device string, limits IOLimits) error {
    cgroupPath := filepath.Join(m.rootPath, "blackhole", service)
    
    // Get device major:minor numbers
    deviceID, err := getDeviceID(device)
    if err != nil {
        return err
    }
    
    // Set bandwidth limits
    if limits.ReadBPS > 0 {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "io.max"),
            []byte(fmt.Sprintf("%s rbps=%d", deviceID, limits.ReadBPS)),
            0644,
        ); err != nil {
            return err
        }
    }
    
    if limits.WriteBPS > 0 {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "io.max"),
            []byte(fmt.Sprintf("%s wbps=%d", deviceID, limits.WriteBPS)),
            0644,
        ); err != nil {
            return err
        }
    }
    
    // Set IOPS limits
    if limits.ReadIOPS > 0 {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "io.max"),
            []byte(fmt.Sprintf("%s riops=%d", deviceID, limits.ReadIOPS)),
            0644,
        ); err != nil {
            return err
        }
    }
    
    if limits.WriteIOPS > 0 {
        if err := ioutil.WriteFile(
            filepath.Join(cgroupPath, "io.max"),
            []byte(fmt.Sprintf("%s wiops=%d", deviceID, limits.WriteIOPS)),
            0644,
        ); err != nil {
            return err
        }
    }
    
    return nil
}
```

## File Descriptor Limits

### FD Limits via rlimit

Control file descriptor usage:

```go
type FDLimits struct {
    Soft int64  // Soft limit
    Hard int64  // Hard limit
}

func (m *RlimitManager) SetFDLimits(pid int, limits FDLimits) error {
    // Attach to process
    proc, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    
    // Set resource limits via syscall
    rlimit := &syscall.Rlimit{
        Cur: uint64(limits.Soft),
        Max: uint64(limits.Hard),
    }
    
    // This requires CAP_SYS_RESOURCE capability
    return syscall.Prlimit(pid, syscall.RLIMIT_NOFILE, rlimit, nil)
}

func (m *RlimitManager) GetFDUsage(pid int) (int, error) {
    // Count open file descriptors
    fdPath := fmt.Sprintf("/proc/%d/fd", pid)
    entries, err := ioutil.ReadDir(fdPath)
    if err != nil {
        return 0, err
    }
    
    return len(entries), nil
}
```

## Network Isolation

### Network Namespaces

Isolate network interfaces per service:

```go
type NetworkLimits struct {
    Bandwidth  BandwidthLimits
    Namespace  string
    Interfaces []string
}

type BandwidthLimits struct {
    IngressRate  int64  // Bytes per second
    EgressRate   int64  // Bytes per second
    IngressBurst int64  // Burst size
    EgressBurst  int64  // Burst size
}

func (m *NetworkManager) CreateServiceNamespace(service string) error {
    // Create network namespace
    nsPath := fmt.Sprintf("/var/run/netns/%s", service)
    
    // Create namespace
    if err := exec.Command("ip", "netns", "add", service).Run(); err != nil {
        return err
    }
    
    // Setup loopback in namespace
    cmds := [][]string{
        {"ip", "netns", "exec", service, "ip", "link", "set", "lo", "up"},
        {"ip", "netns", "exec", service, "ip", "addr", "add", "127.0.0.1/8", "dev", "lo"},
    }
    
    for _, cmd := range cmds {
        if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Traffic Control

Limit network bandwidth:

```go
func (m *NetworkManager) SetBandwidthLimits(service string, limits BandwidthLimits) error {
    // Use tc (traffic control) to set bandwidth limits
    interface := fmt.Sprintf("veth-%s", service)
    
    // Clear existing qdisc
    exec.Command("tc", "qdisc", "del", "dev", interface, "root").Run()
    
    // Add root qdisc
    if err := exec.Command("tc", "qdisc", "add", "dev", interface, "root", "handle", "1:", "htb").Run(); err != nil {
        return err
    }
    
    // Add rate limiting class
    args := []string{
        "tc", "class", "add", "dev", interface,
        "parent", "1:", "classid", "1:1", "htb",
        "rate", fmt.Sprintf("%dkbit", limits.EgressRate/1024),
    }
    
    if limits.EgressBurst > 0 {
        args = append(args, "burst", strconv.FormatInt(limits.EgressBurst, 10))
    }
    
    return exec.Command(args[0], args[1:]...).Run()
}
```

## Security Isolation

### Process Capabilities

Limit process capabilities:

```go
type SecurityContext struct {
    User         string
    Group        string
    Capabilities []string
    NoNewPrivs   bool
    SeccompProfile string
}

func (m *SecurityManager) ApplySecurityContext(pid int, ctx SecurityContext) error {
    // Drop capabilities
    if len(ctx.Capabilities) > 0 {
        caps := capability.NewPid2(pid)
        if err := caps.Load(); err != nil {
            return err
        }
        
        // Clear all capabilities first
        caps.Clear(capability.BOUNDS)
        caps.Clear(capability.EFFECTIVE)
        caps.Clear(capability.INHERITABLE)
        caps.Clear(capability.PERMITTED)
        
        // Add only allowed capabilities
        for _, capStr := range ctx.Capabilities {
            cap, err := capability.FromName(capStr)
            if err != nil {
                return err
            }
            caps.Set(capability.BOUNDS, cap)
            caps.Set(capability.PERMITTED, cap)
        }
        
        if err := caps.Apply(capability.BOUNDS | capability.PERMITTED); err != nil {
            return err
        }
    }
    
    // Set no_new_privs flag
    if ctx.NoNewPrivs {
        if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Implementation Guide

### Service Startup Isolation

Complete isolation setup during service startup:

```go
func (o *Orchestrator) StartServiceWithIsolation(service ServiceConfig) error {
    // Create command
    cmd := exec.Command(service.Executable, service.Args...)
    cmd.Env = service.Environment
    cmd.Dir = service.WorkingDir
    
    // Set process attributes for isolation
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
        Pdeathsig:  syscall.SIGKILL,
    }
    
    // Start process
    if err := cmd.Start(); err != nil {
        return err
    }
    
    pid := cmd.Process.Pid
    
    // Apply resource isolation
    if err := o.cgroupManager.CreateServiceCgroup(service.Name, pid); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Set resource limits
    if err := o.cgroupManager.SetCPULimits(service.Name, service.Resources.CPU); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    if err := o.cgroupManager.SetMemoryLimits(service.Name, service.Resources.Memory); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Apply security context
    if err := o.securityManager.ApplySecurityContext(pid, service.Security); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Set file descriptor limits
    if err := o.rlimitManager.SetFDLimits(pid, service.Resources.FileDesc); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Start monitoring
    o.monitor.WatchProcess(pid, service)
    
    return nil
}
```

### Resource Violation Handling

Handle resource limit violations:

```go
func (m *ProcessMonitor) HandleResourceViolation(service string, violation ResourceViolation) {
    switch violation.Type {
    case MemoryViolation:
        if violation.Severity == Critical {
            // OOM about to trigger
            m.alerts.TriggerAlert(AlertOOM, service, violation)
            m.orchestrator.RestartService(service)
        }
        
    case CPUThrottling:
        if violation.Duration > 30*time.Second {
            // Sustained CPU throttling
            m.alerts.TriggerAlert(AlertCPUThrottle, service, violation)
        }
        
    case IOThrottling:
        // Log I/O bottleneck
        m.metrics.RecordIOThrottle(service, violation)
        
    case FDExhaustion:
        if violation.Usage > 0.9 {
            // 90% of FD limit reached
            m.alerts.TriggerAlert(AlertFDLimit, service, violation)
        }
    }
}
```

## Configuration

### Resource Isolation Configuration

```yaml
resource_isolation:
  # Cgroup configuration
  cgroup:
    version: 2
    root_path: /sys/fs/cgroup
    controllers:
      - cpu
      - memory
      - io
      - pids
  
  # Default limits (can be overridden per service)
  defaults:
    cpu:
      quota: 1.0      # 1 CPU core
      shares: 1024    # Default weight
    memory:
      limit: 1G       # 1GB hard limit
      soft: 768M      # 768MB soft limit
      swap: 512M      # 512MB swap
    io:
      read_bps: 100M  # 100MB/s read
      write_bps: 50M  # 50MB/s write
    fds:
      soft: 1024
      hard: 4096
  
  # Security defaults
  security:
    no_new_privs: true
    capabilities:
      - CAP_NET_BIND_SERVICE
    seccomp_profile: default
```

### Per-Service Configuration

```yaml
services:
  identity:
    resources:
      cpu:
        quota: 0.5
        cpuset: "0-1"   # Cores 0 and 1
      memory:
        limit: 512M
        soft: 384M
      fds:
        soft: 512
        hard: 1024
    security:
      user: blackhole
      group: blackhole
      capabilities:
        - CAP_NET_BIND_SERVICE
        - CAP_NET_ADMIN
  
  storage:
    resources:
      cpu:
        quota: 2.0
        cpuset: "2-5"   # Cores 2-5
      memory:
        limit: 4G
        soft: 3G
      io:
        read_bps: 500M
        write_bps: 200M
      fds:
        soft: 4096
        hard: 8192
```

## Best Practices

### Resource Planning

1. **Conservative Defaults**: Start with conservative limits and adjust based on monitoring
2. **Burst Capacity**: Allow burst capacity for temporary spikes
3. **Service Prioritization**: Critical services get priority access
4. **Resource Headroom**: Leave 10-20% system resources unallocated

### Monitoring and Tuning

1. **Continuous Monitoring**: Track resource usage patterns
2. **Alert Thresholds**: Set alerts at 80% usage
3. **Regular Review**: Adjust limits based on usage data
4. **Capacity Planning**: Project future resource needs

### Security Considerations

1. **Minimal Privileges**: Grant only necessary capabilities
2. **Network Isolation**: Use namespaces for sensitive services
3. **File System Isolation**: Restrict access to service directories
4. **Process Isolation**: Use PID namespaces when possible

### Troubleshooting

1. **OOM Events**: Check memory limits and usage patterns
2. **CPU Throttling**: Review CPU quotas and scheduling
3. **I/O Bottlenecks**: Monitor disk usage and adjust limits
4. **FD Exhaustion**: Check for connection leaks

The resource isolation system ensures that each service subprocess operates within well-defined boundaries, maintaining system stability and preventing resource starvation while enabling efficient resource utilization across the platform.