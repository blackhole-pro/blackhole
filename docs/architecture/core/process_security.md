# Process Security Architecture

## Overview

The Blackhole platform implements multiple layers of security for subprocess isolation, ensuring that each service runs with the minimum required privileges and cannot compromise other services or the host system.

## Security Principles

1. **Least Privilege**: Each subprocess runs with minimal required permissions
2. **Defense in Depth**: Multiple security layers prevent compromise
3. **Fail Secure**: Security defaults to deny in case of misconfiguration
4. **Audit Trail**: All security events are logged for analysis
5. **Zero Trust**: Services must authenticate even within the same host

## Process Sandboxing

### User and Group Isolation

Each service runs as a dedicated user:

```go
type ProcessSecurity struct {
    ServiceName string
    UID         uint32
    GID         uint32
    HomeDir     string
    Capabilities []string
}

func (s *ProcessSecurity) ApplySecurityContext(cmd *exec.Cmd) error {
    // Set user and group
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Credential: &syscall.Credential{
            Uid: s.UID,
            Gid: s.GID,
        },
        // Create new process group
        Setpgid: true,
        // Create new session
        Setsid: true,
    }
    
    // Set secure environment
    cmd.Env = []string{
        fmt.Sprintf("HOME=%s", s.HomeDir),
        fmt.Sprintf("USER=%s", s.ServiceName),
        "PATH=/usr/bin:/bin",
    }
    
    return nil
}
```

### Linux Capabilities

Drop unnecessary capabilities:

```go
func (s *ProcessSecurity) DropCapabilities() error {
    // Keep only required capabilities
    allowedCaps := map[string]bool{
        "CAP_NET_BIND_SERVICE": true, // Bind to ports < 1024
        "CAP_SYS_RESOURCE":     true, // Set resource limits
    }
    
    // Get current capabilities
    caps := cap.GetProc()
    
    // Clear all capabilities
    caps.Clear(cap.Permitted)
    caps.Clear(cap.Effective)
    caps.Clear(cap.Inheritable)
    
    // Set only allowed capabilities
    for capName := range allowedCaps {
        cap, err := cap.FromName(capName)
        if err != nil {
            continue
        }
        caps.Set(cap.Permitted, cap)
        caps.Set(cap.Effective, cap)
    }
    
    return caps.SetProc()
}
```

### Namespace Isolation

Use Linux namespaces for isolation:

```go
func (s *ProcessSecurity) ApplyNamespaces(cmd *exec.Cmd) error {
    cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWNET |    // Network namespace
                                syscall.CLONE_NEWNS |     // Mount namespace
                                syscall.CLONE_NEWPID |    // PID namespace
                                syscall.CLONE_NEWIPC |    // IPC namespace
                                syscall.CLONE_NEWUTS      // UTS namespace
    
    // Pre-exec function to setup namespaces
    cmd.SysProcAttr.CmdLineFunc = func() error {
        // Mount new /tmp for this process
        if err := syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, ""); err != nil {
            return err
        }
        
        // Set hostname in UTS namespace
        if err := syscall.Sethostname([]byte(s.ServiceName)); err != nil {
            return err
        }
        
        return nil
    }
    
    return nil
}
```

## Filesystem Isolation

### Read-Only Root Filesystem

Mount root as read-only:

```go
func (s *ProcessSecurity) SetupFilesystem() error {
    // Remount root as read-only
    if err := syscall.Mount("", "/", "", syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
        return fmt.Errorf("remount root read-only: %w", err)
    }
    
    // Create writable directories
    writableDirs := []string{
        fmt.Sprintf("/var/run/blackhole/%s", s.ServiceName),
        fmt.Sprintf("/var/log/blackhole/%s", s.ServiceName),
        fmt.Sprintf("/tmp/blackhole/%s", s.ServiceName),
    }
    
    for _, dir := range writableDirs {
        if err := os.MkdirAll(dir, 0750); err != nil {
            return fmt.Errorf("create writable dir %s: %w", dir, err)
        }
        
        // Mount as tmpfs
        if err := syscall.Mount("tmpfs", dir, "tmpfs", 0, "size=100M"); err != nil {
            return fmt.Errorf("mount tmpfs %s: %w", dir, err)
        }
        
        // Set ownership
        if err := os.Chown(dir, int(s.UID), int(s.GID)); err != nil {
            return fmt.Errorf("chown %s: %w", dir, err)
        }
    }
    
    return nil
}
```

### Restricted File Access

Limit file access with AppArmor/SELinux:

```go
type FileAccessPolicy struct {
    AllowedPaths []PathRule
    DeniedPaths  []string
}

type PathRule struct {
    Path string
    Mode string // "r", "w", "rw"
}

func (p *FileAccessPolicy) GenerateAppArmorProfile(service string) string {
    profile := fmt.Sprintf(`
#include <tunables/global>

profile %s_service {
    #include <abstractions/base>
    
    # Network access
    network inet stream,
    network unix stream,
    
    # Service specific socket
    /var/run/blackhole/%s.sock rw,
    
    # Logs
    /var/log/blackhole/%s/** w,
    
    # Temporary files
    /tmp/blackhole/%s/** rw,
    
    # Read-only access to binary
    /usr/bin/blackhole r,
    
    # Deny everything else
    deny /** w,
    deny /** x,
}
`, service, service, service, service)
    
    return profile
}
```

## Network Security

### Unix Socket Permissions

Secure Unix socket access:

```go
func (s *ProcessSecurity) SecureUnixSocket(socketPath string) error {
    // Create socket directory with restricted permissions
    socketDir := filepath.Dir(socketPath)
    if err := os.MkdirAll(socketDir, 0750); err != nil {
        return err
    }
    
    // Set directory ownership
    if err := os.Chown(socketDir, int(s.UID), int(s.GID)); err != nil {
        return err
    }
    
    // Create socket with restricted permissions
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        return err
    }
    
    // Set socket permissions (only owner can access)
    if err := os.Chmod(socketPath, 0600); err != nil {
        return err
    }
    
    // Set socket ownership
    if err := os.Chown(socketPath, int(s.UID), int(s.GID)); err != nil {
        return err
    }
    
    return nil
}
```

### Network Namespace

Isolate network access:

```go
func (s *ProcessSecurity) SetupNetworkNamespace() error {
    // Create network namespace
    if err := syscall.Unshare(syscall.CLONE_NEWNET); err != nil {
        return fmt.Errorf("unshare network namespace: %w", err)
    }
    
    // Create veth pair
    vethHost := fmt.Sprintf("veth-%s", s.ServiceName)
    vethNS := "eth0"
    
    // Add veth pair
    cmd := exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("create veth pair: %w", err)
    }
    
    // Move one end to namespace
    cmd = exec.Command("ip", "link", "set", vethNS, "netns", fmt.Sprint(os.Getpid()))
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("move veth to namespace: %w", err)
    }
    
    // Configure interfaces
    if err := configureInterface(vethHost, s.HostIP); err != nil {
        return err
    }
    
    if err := configureInterface(vethNS, s.ServiceIP); err != nil {
        return err
    }
    
    return nil
}
```

## Process Monitoring

### Security Events

Monitor and log security events:

```go
type SecurityMonitor struct {
    service string
    logger  *log.Logger
}

func (m *SecurityMonitor) MonitorProcess(pid int) {
    // Monitor system calls
    go m.monitorSyscalls(pid)
    
    // Monitor file access
    go m.monitorFileAccess(pid)
    
    // Monitor network connections
    go m.monitorNetwork(pid)
    
    // Monitor resource usage
    go m.monitorResources(pid)
}

func (m *SecurityMonitor) monitorSyscalls(pid int) {
    // Setup ptrace
    if err := syscall.PtraceAttach(pid); err != nil {
        m.logger.Printf("Failed to attach ptrace: %v", err)
        return
    }
    defer syscall.PtraceDetach(pid)
    
    for {
        var regs syscall.PtraceRegs
        if err := syscall.PtraceGetRegs(pid, &regs); err != nil {
            continue
        }
        
        // Check for suspicious syscalls
        switch regs.Orig_rax {
        case syscall.SYS_OPEN:
            m.logFileAccess(pid, regs)
        case syscall.SYS_CONNECT:
            m.logNetworkAccess(pid, regs)
        case syscall.SYS_EXECVE:
            m.logExecAttempt(pid, regs)
        }
        
        // Continue execution
        syscall.PtraceCont(pid, 0)
    }
}
```

### Audit Logging

Log all security events:

```go
type SecurityAuditLog struct {
    file   *os.File
    mu     sync.Mutex
}

type SecurityEvent struct {
    Timestamp   time.Time
    Service     string
    PID         int
    EventType   string
    Description string
    Severity    string
    Metadata    map[string]interface{}
}

func (a *SecurityAuditLog) LogEvent(event SecurityEvent) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    // Format as JSON
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    // Write to audit log
    if _, err := a.file.Write(append(data, '\n')); err != nil {
        return err
    }
    
    // Sync to disk
    return a.file.Sync()
}

func (a *SecurityAuditLog) LogAccessDenied(service string, pid int, resource string) {
    a.LogEvent(SecurityEvent{
        Timestamp:   time.Now(),
        Service:     service,
        PID:         pid,
        EventType:   "ACCESS_DENIED",
        Description: fmt.Sprintf("Access denied to resource: %s", resource),
        Severity:    "WARNING",
        Metadata: map[string]interface{}{
            "resource": resource,
        },
    })
}
```

## Resource Limits

### Memory Protection

Prevent memory-based attacks:

```go
func (s *ProcessSecurity) SetMemoryLimits(cmd *exec.Cmd) error {
    // Disable core dumps
    cmd.Env = append(cmd.Env, "GOTRACEBACK=none")
    
    // Set memory limits
    cmd.SysProcAttr.Rlimits = []syscall.Rlimit{
        {
            Type: syscall.RLIMIT_AS,      // Address space limit
            Cur:  s.MaxMemory,
            Max:  s.MaxMemory,
        },
        {
            Type: syscall.RLIMIT_DATA,    // Data segment limit
            Cur:  s.MaxMemory / 2,
            Max:  s.MaxMemory / 2,
        },
        {
            Type: syscall.RLIMIT_STACK,   // Stack size limit
            Cur:  8 * 1024 * 1024,       // 8MB
            Max:  8 * 1024 * 1024,
        },
        {
            Type: syscall.RLIMIT_CORE,    // Core dump size
            Cur:  0,
            Max:  0,
        },
    }
    
    return nil
}
```

### File Descriptor Limits

Prevent file descriptor exhaustion:

```go
func (s *ProcessSecurity) SetFDLimits(cmd *exec.Cmd) error {
    cmd.SysProcAttr.Rlimits = append(cmd.SysProcAttr.Rlimits, syscall.Rlimit{
        Type: syscall.RLIMIT_NOFILE,
        Cur:  uint64(s.MaxFileDescriptors),
        Max:  uint64(s.MaxFileDescriptors),
    })
    
    return nil
}
```

## Platform-Specific Security

### macOS Sandbox

Use macOS sandbox profiles:

```go
func (s *ProcessSecurity) ApplyMacOSSandbox(cmd *exec.Cmd) error {
    sandboxProfile := fmt.Sprintf(`
(version 1)
(deny default)

; Allow reading system files
(allow file-read*
    (regex #"^/usr/lib/")
    (regex #"^/System/Library/"))

; Allow writing to service directories
(allow file-write*
    (regex #"^/var/run/blackhole/%s")
    (regex #"^/var/log/blackhole/%s"))

; Allow network access
(allow network-outbound)
(allow network-bind
    (local ip "localhost:%d"))

; Allow mach services
(allow mach-lookup
    (global-name "com.apple.system.logger"))
`, s.ServiceName, s.ServiceName, s.Port)
    
    // Write sandbox profile to temp file
    profilePath := fmt.Sprintf("/tmp/blackhole_%s.sb", s.ServiceName)
    if err := ioutil.WriteFile(profilePath, []byte(sandboxProfile), 0644); err != nil {
        return err
    }
    
    // Apply sandbox
    cmd.Args = append([]string{"sandbox-exec", "-f", profilePath}, cmd.Args...)
    cmd.Path = "/usr/bin/sandbox-exec"
    
    return nil
}
```

### Windows Security

Use Windows security features:

```go
func (s *ProcessSecurity) ApplyWindowsSecurity(cmd *exec.Cmd) error {
    // Create restricted token
    token, err := windows.OpenCurrentProcessToken()
    if err != nil {
        return err
    }
    defer token.Close()
    
    // Create restricted token
    var restrictedToken windows.Token
    err = windows.CreateRestrictedToken(
        token,
        windows.DISABLE_MAX_PRIVILEGE,
        nil,  // SIDs to disable
        nil,  // Privileges to delete
        nil,  // SIDs to restrict
        &restrictedToken,
    )
    if err != nil {
        return err
    }
    
    // Apply token to process
    cmd.SysProcAttr = &windows.SysProcAttr{
        Token: restrictedToken,
    }
    
    return nil
}
```

## Security Configuration

### Service Security Profile

Define security profiles per service:

```yaml
security_profiles:
  identity:
    user: blackhole_identity
    group: blackhole_services
    capabilities:
      - CAP_NET_BIND_SERVICE
    filesystem:
      read_only_root: true
      writable_paths:
        - /var/run/blackhole/identity
        - /var/log/blackhole/identity
    network:
      allow_localhost: true
      allow_unix_sockets: true
      allowed_ports:
        - 50001
    resource_limits:
      max_memory: 1GB
      max_file_descriptors: 4096
      max_processes: 100

  storage:
    user: blackhole_storage
    group: blackhole_services
    capabilities:
      - CAP_NET_BIND_SERVICE
      - CAP_SYS_RESOURCE
    filesystem:
      read_only_root: true
      writable_paths:
        - /var/run/blackhole/storage
        - /var/log/blackhole/storage
        - /var/data/blackhole/storage
    network:
      allow_localhost: true
      allow_unix_sockets: true
      allowed_ports:
        - 50002
    resource_limits:
      max_memory: 4GB
      max_file_descriptors: 8192
      max_processes: 200
```

### Security Policies

Implement security policies:

```go
type SecurityPolicy struct {
    Name                string
    EnforceMode         string // "enforce", "permissive", "disabled"
    AllowedSyscalls     []string
    DeniedSyscalls      []string
    FileAccessRules     []FileAccessRule
    NetworkAccessRules  []NetworkAccessRule
    ResourceLimits      ResourceLimits
}

func (p *SecurityPolicy) Enforce(cmd *exec.Cmd) error {
    switch p.EnforceMode {
    case "enforce":
        return p.applyAllRules(cmd)
    case "permissive":
        // Log violations but don't block
        go p.monitorViolations(cmd)
        return nil
    case "disabled":
        return nil
    default:
        return fmt.Errorf("unknown enforce mode: %s", p.EnforceMode)
    }
}
```

## Security Best Practices

### Process Creation

1. Always drop privileges after binding ports
2. Use dedicated users per service
3. Apply all security controls before exec
4. Validate all environment variables
5. Clear sensitive data from memory

### Communication Security

1. Use Unix sockets for local communication
2. Enable mTLS for remote communication
3. Validate all RPC requests
4. Implement rate limiting
5. Log all authentication failures

### Resource Protection

1. Set strict resource limits
2. Monitor resource usage
3. Implement circuit breakers
4. Use health checks
5. Enable automatic recovery

### Audit and Compliance

1. Log all security events
2. Implement tamper-proof audit logs
3. Regular security assessments
4. Compliance reporting
5. Incident response procedures

## Testing Security

### Security Test Suite

```go
func TestProcessIsolation(t *testing.T) {
    // Start service with security profile
    security := &ProcessSecurity{
        ServiceName: "test_service",
        UID:         65534, // nobody
        GID:         65534,
    }
    
    cmd := exec.Command("/usr/bin/blackhole", "service", "--name", "test")
    if err := security.ApplySecurityContext(cmd); err != nil {
        t.Fatal(err)
    }
    
    // Try to access restricted resources
    tests := []struct {
        name     string
        action   func() error
        expected error
    }{
        {
            name: "access_root_file",
            action: func() error {
                return ioutil.WriteFile("/etc/passwd", []byte("test"), 0644)
            },
            expected: syscall.EACCES,
        },
        {
            name: "bind_privileged_port",
            action: func() error {
                ln, err := net.Listen("tcp", ":80")
                if err == nil {
                    ln.Close()
                }
                return err
            },
            expected: syscall.EACCES,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.action()
            if !errors.Is(err, tt.expected) {
                t.Errorf("expected %v, got %v", tt.expected, err)
            }
        })
    }
}
```

### Penetration Testing

Regular security testing:

```go
func TestSecurityVulnerabilities(t *testing.T) {
    // Test for common vulnerabilities
    vulnerabilities := []SecurityTest{
        TestPrivilegeEscalation{},
        TestFileSystemTraversal{},
        TestNetworkEscape{},
        TestResourceExhaustion{},
        TestCodeInjection{},
    }
    
    for _, vuln := range vulnerabilities {
        t.Run(vuln.Name(), func(t *testing.T) {
            if err := vuln.Test(); err != nil {
                t.Errorf("Security vulnerability detected: %v", err)
            }
        })
    }
}
```

## Incident Response

### Security Breach Detection

```go
type SecurityIncidentDetector struct {
    monitor  *SecurityMonitor
    alerting *AlertingSystem
}

func (d *SecurityIncidentDetector) DetectAnomaly(event SecurityEvent) {
    // Check against known attack patterns
    if d.isAttackPattern(event) {
        incident := SecurityIncident{
            Timestamp:   time.Now(),
            Service:     event.Service,
            Type:        "ATTACK_DETECTED",
            Severity:    "CRITICAL",
            Description: fmt.Sprintf("Potential attack detected: %s", event.Description),
            Response:    d.getIncidentResponse(event),
        }
        
        // Alert security team
        d.alerting.SendAlert(incident)
        
        // Execute automatic response
        d.executeResponse(incident)
    }
}

func (d *SecurityIncidentDetector) executeResponse(incident SecurityIncident) {
    switch incident.Response {
    case "ISOLATE":
        d.isolateService(incident.Service)
    case "TERMINATE":
        d.terminateService(incident.Service)
    case "BLOCK":
        d.blockNetworkAccess(incident.Service)
    }
}
```

## Conclusion

The subprocess security architecture provides defense-in-depth protection through:

1. **Process Isolation**: User separation, capabilities, and namespaces
2. **Filesystem Security**: Read-only root, restricted access
3. **Network Security**: Unix socket permissions, network namespaces
4. **Resource Protection**: Memory and file descriptor limits
5. **Monitoring**: Security event logging and anomaly detection

This multi-layered approach ensures that even if one security layer is compromised, others remain intact to protect the system.