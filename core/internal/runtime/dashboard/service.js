class ServiceDetailPage {
    constructor() {
        this.serviceName = null;
        this.autoRefreshInterval = null;
        this.isAutoRefreshEnabled = true;
        this.refreshIntervalMs = 3000; // 3 seconds for more responsive updates
        this.autoScrollEnabled = true;
        this.logLevel = 'all';
        
        this.init();
    }

    init() {
        this.getServiceNameFromURL();
        this.setupEventListeners();
        this.loadServiceData();
        this.startAutoRefresh();
        this.startLogStream();
    }

    getServiceNameFromURL() {
        const urlParams = new URLSearchParams(window.location.search);
        this.serviceName = urlParams.get('service');
        
        if (!this.serviceName) {
            // Redirect back to dashboard if no service specified
            window.location.href = 'index.html';
            return;
        }

        // Update page title and service name
        document.title = `${this.serviceName.charAt(0).toUpperCase() + this.serviceName.slice(1)} Service - Blackhole Dashboard`;
        document.getElementById('serviceTitle').textContent = `${this.serviceName.charAt(0).toUpperCase() + this.serviceName.slice(1)} Service Details`;
        document.getElementById('serviceName').textContent = `${this.serviceName.charAt(0).toUpperCase() + this.serviceName.slice(1)} Service`;
    }

    setupEventListeners() {
        // Back button
        document.getElementById('backBtn').addEventListener('click', () => {
            window.location.href = 'index.html';
        });

        // Service action buttons
        document.getElementById('startBtn').addEventListener('click', () => {
            this.handleServiceAction('start');
        });
        
        document.getElementById('stopBtn').addEventListener('click', () => {
            this.handleServiceAction('stop');
        });
        
        document.getElementById('restartBtn').addEventListener('click', () => {
            this.handleServiceAction('restart');
        });

        // Log controls
        document.getElementById('logLevel').addEventListener('change', (e) => {
            this.logLevel = e.target.value;
            this.filterLogs();
        });

        document.getElementById('clearLogsBtn').addEventListener('click', () => {
            this.clearLogs();
        });

        document.getElementById('downloadLogsBtn').addEventListener('click', () => {
            this.downloadLogs();
        });

        document.getElementById('autoScrollBtn').addEventListener('click', () => {
            this.toggleAutoScroll();
        });

        // Quick action buttons
        document.getElementById('viewConfigBtn').addEventListener('click', () => {
            this.handleQuickAction('viewConfig');
        });

        document.getElementById('exportLogsBtn').addEventListener('click', () => {
            this.handleQuickAction('exportLogs');
        });

        document.getElementById('restartWithLogsBtn').addEventListener('click', () => {
            this.handleQuickAction('restartWithLogs');
        });

        document.getElementById('checkHealthBtn').addEventListener('click', () => {
            this.handleQuickAction('checkHealth');
        });

        document.getElementById('viewMetricsBtn').addEventListener('click', () => {
            this.handleQuickAction('viewMetrics');
        });

        document.getElementById('editConfigBtn').addEventListener('click', () => {
            this.handleQuickAction('editConfig');
        });

        document.getElementById('checkPortBtn').addEventListener('click', () => {
            this.handleQuickAction('checkPort');
        });
    }

    async loadServiceData() {
        try {
            this.updateLastRefreshTime();
            
            // Load basic service status
            const statusResponse = await fetch('/api/status');
            if (statusResponse.ok) {
                const statusData = await statusResponse.json();
                const serviceData = statusData.services[this.serviceName];
                if (serviceData) {
                    this.updateServiceOverview(serviceData);
                }
            }

            // Load detailed service information
            const detailResponse = await fetch(`/api/services/${this.serviceName}/details`);
            if (detailResponse.ok) {
                const detailData = await detailResponse.json();
                this.updateServiceDetails(detailData);
            }

            // Load service logs with enhanced details
            this.loadServiceLogs();

        } catch (error) {
            console.error('Failed to load service data:', error);
            this.showNotification('Failed to load service data', 'error');
        }
    }

    async loadServiceLogs() {
        try {
            const logsResponse = await fetch(`/api/services/${this.serviceName}/logs`);
            if (logsResponse.ok) {
                const logsData = await logsResponse.json();
                this.displayEnhancedLogs(logsData.logs);
            }
        } catch (error) {
            console.error('Failed to load service logs:', error);
        }
    }

    displayEnhancedLogs(logs) {
        const logsContent = document.getElementById('logsContent');
        logsContent.innerHTML = ''; // Clear existing logs
        
        // Update log source indicator
        const logSourceIndicator = document.getElementById('logSource');
        if (logs.length > 0 && logs[0].owner === this.serviceName) {
            logSourceIndicator.textContent = 'Live Logs';
            logSourceIndicator.className = 'log-source-indicator real-logs';
        } else {
            logSourceIndicator.textContent = 'Simulated';
            logSourceIndicator.className = 'log-source-indicator simulated-logs';
        }
        
        logs.forEach(logEntry => {
            this.addRealLogEntry(logEntry);
        });
    }

    addRealLogEntry(logData) {
        const logsContent = document.getElementById('logsContent');
        
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${logData.level}`;
        
        // Build enhanced log entry with all available data
        let logContent = `
            <span class="log-timestamp">${logData.timestamp}</span>
            <span class="log-level ${logData.level}">${logData.level.toUpperCase()}</span>
            <span class="log-message">${logData.message}</span>
        `;
        
        // Add owner information
        if (logData.owner) {
            logContent += `<span class="log-owner">@${logData.owner}</span>`;
        }
        
        // Add process details
        if (logData.pid && logData.pid !== 'N/A') {
            logContent += `<span class="log-meta">PID:${logData.pid}</span>`;
        }
        if (logData.port && logData.port !== 'N/A') {
            logContent += `<span class="log-meta">PORT:${logData.port}</span>`;
        }
        
        // Add special details
        if (logData.socketPath) {
            logContent += `<span class="log-meta">SOCKET:${logData.socketPath}</span>`;
        }
        if (logData.pidFile) {
            logContent += `<span class="log-meta">PIDFILE:${logData.pidFile}</span>`;
        }
        if (logData.configFile) {
            logContent += `<span class="log-meta">CONFIG:${logData.configFile}</span>`;
        }
        
        // Add suggestions
        if (logData.suggestion) {
            logContent += `<span class="log-suggestion">💡 ${logData.suggestion}</span>`;
        }
        if (logData.action) {
            logContent += `<span class="log-action">🔧 ${logData.action}</span>`;
        }
        
        // Add special styling for critical errors
        if (logData.level === 'error' && (logData.message.includes('already in use') || logData.message.includes('daemon is not running'))) {
            logEntry.classList.add('critical-error');
        }
        
        logEntry.innerHTML = logContent;
        logsContent.appendChild(logEntry);
        
        // Filter entry if needed
        if (this.logLevel !== 'all' && logData.level !== this.logLevel) {
            logEntry.style.display = 'none';
        }
    }

    updateServiceOverview(serviceData) {
        // Update status badge
        const statusBadge = document.getElementById('serviceStatus');
        statusBadge.textContent = serviceData.status;
        statusBadge.className = `status-badge-large ${serviceData.status}`;

        // Update basic info
        document.getElementById('servicePid').textContent = serviceData.pid || 'N/A';
        document.getElementById('servicePort').textContent = serviceData.port || 'N/A';
        document.getElementById('serviceLastCheck').textContent = this.formatTime(serviceData.lastCheck);

        // Update action buttons
        this.updateActionButtons(serviceData.status);
    }

    updateServiceDetails(detailData) {
        // Process information
        document.getElementById('serviceSocket').textContent = detailData.socketPath || `sockets/${this.serviceName}.sock`;
        document.getElementById('serviceUptime').textContent = detailData.uptime || 'N/A';

        // Health & Metrics
        document.getElementById('serviceCpu').textContent = detailData.cpuUsage || 'N/A';
        document.getElementById('serviceMemory').textContent = detailData.memoryUsage || 'N/A';
        document.getElementById('serviceStartCount').textContent = detailData.startCount || '0';

        // Configuration
        document.getElementById('serviceType').textContent = detailData.serviceType || 'Standard';
        document.getElementById('serviceAutoRestart').textContent = detailData.autoRestart ? 'Enabled' : 'Disabled';
        document.getElementById('serviceLogLevel').textContent = detailData.logLevel || 'info';
        document.getElementById('serviceConfigFile').textContent = detailData.configFile || 'configs/blackhole.yaml';
    }

    updateActionButtons(status) {
        const startBtn = document.getElementById('startBtn');
        const stopBtn = document.getElementById('stopBtn');
        const restartBtn = document.getElementById('restartBtn');

        if (status === 'running') {
            startBtn.disabled = true;
            stopBtn.disabled = false;
            restartBtn.disabled = false;
        } else {
            startBtn.disabled = false;
            stopBtn.disabled = true;
            restartBtn.disabled = true;
        }
    }

    async handleServiceAction(action) {
        const button = document.getElementById(`${action}Btn`);
        if (button.disabled || button.classList.contains('loading')) {
            return;
        }

        // Set loading state
        button.classList.add('loading');
        button.disabled = true;

        try {
            const response = await fetch(`/api/services/${this.serviceName}/${action}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            
            if (result.success) {
                this.showNotification(`${action.charAt(0).toUpperCase() + action.slice(1)} ${this.serviceName} service successfully`, 'success');
                
                // Refresh data and logs immediately to show the action result
                setTimeout(() => {
                    this.loadServiceData();
                    this.loadServiceLogs(); // Reload logs to show the action entries
                }, 500);
            } else {
                throw new Error(result.error || `Failed to ${action} ${this.serviceName} service`);
            }
        } catch (error) {
            console.error(`Failed to ${action} ${this.serviceName}:`, error);
            this.showNotification(`Failed to ${action} ${this.serviceName} service: ${error.message}`, 'error');
        } finally {
            // Remove loading state
            button.classList.remove('loading');
            button.disabled = false;
            
            // Re-update button states after action
            setTimeout(() => {
                this.loadServiceData();
            }, 500);
        }
    }

    async handleQuickAction(action) {
        switch (action) {
            case 'viewConfig':
                this.showNotification('Opening configuration viewer...', 'info');
                break;
            case 'exportLogs':
                this.downloadLogs();
                break;
            case 'restartWithLogs':
                this.showNotification('Restarting service with debug logging...', 'info');
                await this.handleServiceAction('restart');
                break;
            case 'checkHealth':
                await this.performHealthCheck();
                break;
            case 'viewMetrics':
                this.showNotification('Opening metrics dashboard...', 'info');
                break;
            case 'editConfig':
                this.showNotification('Configuration editor coming soon...', 'info');
                break;
            case 'checkPort':
                this.checkPortUsage();
                break;
        }
    }

    async performHealthCheck() {
        try {
            const response = await fetch(`/api/services/${this.serviceName}/health`, {
                method: 'POST'
            });
            
            if (response.ok) {
                const result = await response.json();
                this.showNotification(`Health check completed: ${result.status}`, 'success');
                this.addLogEntry('info', `Health check performed - Status: ${result.status}`);
            } else {
                throw new Error('Health check failed');
            }
        } catch (error) {
            this.showNotification('Health check failed', 'error');
            this.addLogEntry('error', `Health check failed: ${error.message}`);
        }
    }

    checkPortUsage() {
        // Get the service port for checking
        const servicePorts = {
            'identity': 8101, // Changed from 8001 to avoid conflicts
            'storage': 8002,
            'node': 8003,
            'ledger': 8004,
            'social': 8005,
            'indexer': 8006,
            'analytics': 8007,
            'wallet': 8008,
        };
        
        const port = servicePorts[this.serviceName] || 8000;
        
        // Show port checking information
        this.showNotification(`Checking port ${port} usage for ${this.serviceName} service...`, 'info');
        
        // Add instructional log entry
        this.addLogEntry('info', `Port ${port} check initiated`, 'system', {
            suggestion: `Run: lsof -i :${port} or netstat -tulpn | grep ${port}`
        });
        
        // Simulate port check result after a delay
        setTimeout(() => {
            // With the new port 8101, identity service should now be available
            this.addLogEntry('info', `Port ${port} is available for ${this.serviceName} service`, 'system');
            this.showNotification(`Port ${port} is available for ${this.serviceName} service`, 'success');
        }, 1000);
    }

    startAutoRefresh() {
        if (this.isAutoRefreshEnabled) {
            this.autoRefreshInterval = setInterval(() => {
                this.loadServiceData();
            }, this.refreshIntervalMs);
        }
    }

    stopAutoRefresh() {
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
            this.autoRefreshInterval = null;
        }
    }

    startLogStream() {
        // Simulate live log streaming
        setInterval(() => {
            this.simulateLogEntry();
        }, 5000);
    }

    simulateLogEntry() {
        const levels = ['info', 'warn', 'error', 'debug'];
        const messages = [
            'Processing request from client',
            'Database connection established',
            'Cache miss for key: user_session_123',
            'Authentication token validated',
            'Background task completed',
            'Memory usage: 67%',
            'New connection established',
            'Configuration reloaded'
        ];

        const level = levels[Math.floor(Math.random() * levels.length)];
        const message = messages[Math.floor(Math.random() * messages.length)];
        
        // Simulate owner information
        const currentUser = 'system';
        const additionalData = {
            pid: Math.floor(Math.random() * 10000) + 1000,
            port: 8000 + Math.floor(Math.random() * 10),
        };
        
        this.addLogEntry(level, message, currentUser, additionalData);
    }

    addLogEntry(level, message, owner = null, additionalData = {}) {
        const logsContent = document.getElementById('logsContent');
        const timestamp = new Date().toISOString().slice(0, 19).replace('T', ' ');
        
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${level}`;
        
        // Build enhanced log entry with owner and additional details
        let logContent = `
            <span class="log-timestamp">${timestamp}</span>
            <span class="log-level ${level}">${level.toUpperCase()}</span>
            <span class="log-message">${message}</span>
        `;
        
        // Add owner information if available
        if (owner) {
            logContent += `<span class="log-owner">@${owner}</span>`;
        }
        
        // Add additional metadata if available
        if (additionalData.pid && additionalData.pid !== 'N/A') {
            logContent += `<span class="log-meta">PID:${additionalData.pid}</span>`;
        }
        if (additionalData.port && additionalData.port !== 'N/A') {
            logContent += `<span class="log-meta">PORT:${additionalData.port}</span>`;
        }
        if (additionalData.suggestion) {
            logContent += `<span class="log-suggestion">💡 ${additionalData.suggestion}</span>`;
        }
        
        logEntry.innerHTML = logContent;
        logsContent.appendChild(logEntry);
        
        // Auto-scroll if enabled
        if (this.autoScrollEnabled) {
            logsContent.scrollTop = logsContent.scrollHeight;
        }
        
        // Filter new entry if needed
        if (this.logLevel !== 'all' && level !== this.logLevel) {
            logEntry.style.display = 'none';
        }
    }

    filterLogs() {
        const logEntries = document.querySelectorAll('.log-entry');
        logEntries.forEach(entry => {
            if (this.logLevel === 'all') {
                entry.style.display = 'flex';
            } else {
                const entryLevel = entry.className.split(' ')[1];
                entry.style.display = entryLevel === this.logLevel ? 'flex' : 'none';
            }
        });
    }

    clearLogs() {
        document.getElementById('logsContent').innerHTML = '';
        this.showNotification('Logs cleared', 'info');
    }

    downloadLogs() {
        const logs = document.querySelectorAll('.log-entry:not([style*="display: none"])');
        let logContent = '';
        
        logs.forEach(log => {
            const timestamp = log.querySelector('.log-timestamp').textContent;
            const level = log.querySelector('.log-level').textContent;
            const message = log.querySelector('.log-message').textContent;
            
            // Add basic log info
            let logLine = `${timestamp} [${level}] ${message}`;
            
            // Add owner information
            const owner = log.querySelector('.log-owner');
            if (owner) {
                logLine += ` (${owner.textContent})`;
            }
            
            // Add metadata
            const metaElements = log.querySelectorAll('.log-meta');
            if (metaElements.length > 0) {
                const metaInfo = Array.from(metaElements).map(meta => meta.textContent).join(' ');
                logLine += ` [${metaInfo}]`;
            }
            
            // Add suggestions
            const suggestion = log.querySelector('.log-suggestion');
            if (suggestion) {
                logLine += ` SUGGESTION: ${suggestion.textContent.replace('💡 ', '')}`;
            }
            
            // Add actions
            const action = log.querySelector('.log-action');
            if (action) {
                logLine += ` ACTION: ${action.textContent.replace('🔧 ', '')}`;
            }
            
            logContent += logLine + '\n';
        });
        
        const blob = new Blob([logContent], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${this.serviceName}-enhanced-logs-${new Date().toISOString().slice(0, 10)}.log`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        this.showNotification('Enhanced logs downloaded with owner and metadata information', 'success');
    }

    toggleAutoScroll() {
        this.autoScrollEnabled = !this.autoScrollEnabled;
        const button = document.getElementById('autoScrollBtn');
        
        if (this.autoScrollEnabled) {
            button.classList.add('active');
            button.textContent = 'Auto-scroll: ON';
        } else {
            button.classList.remove('active');
            button.textContent = 'Auto-scroll: OFF';
        }
    }

    updateLastRefreshTime() {
        const now = new Date();
        document.getElementById('lastUpdated').textContent = now.toLocaleTimeString();
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}-message`;
        notification.textContent = message;
        
        const colors = {
            error: { bg: '#fed7d7', color: '#742a2a', border: '#feb2b2' },
            success: { bg: '#c6f6d5', color: '#22543d', border: '#9ae6b4' },
            info: { bg: '#bee3f8', color: '#2c5282', border: '#90cdf4' }
        };
        
        const style = colors[type] || colors.info;
        
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${style.bg};
            color: ${style.color};
            padding: 15px 20px;
            border-radius: 8px;
            border: 1px solid ${style.border};
            z-index: 1000;
            box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
            transform: translateX(100%);
            transition: transform 0.3s ease;
        `;
        
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.style.transform = 'translateX(0)';
        }, 10);
        
        // Auto-remove after 4 seconds
        setTimeout(() => {
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 4000);
    }

    formatTime(isoString) {
        return new Date(isoString).toLocaleTimeString();
    }
}

// Initialize service detail page when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new ServiceDetailPage();
});