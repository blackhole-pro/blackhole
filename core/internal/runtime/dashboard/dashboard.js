class ServiceDashboard {
    constructor() {
        this.services = [
            'daemon', 'orchestrator', 'mesh',
            'identity', 'storage', 'node', 'ledger', 
            'social', 'indexer', 'analytics', 'wallet'
        ];
        this.websocket = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectInterval = 5000; // 5 seconds
        
        // Single update timer
        this.updateTimer = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.connectWebSocket();
    }

    startUpdates() {
        // Clear any existing timer first (in case of reconnection)
        if (this.updateTimer) {
            clearInterval(this.updateTimer);
        }
        
        // Start immediate update
        this.requestStatusUpdate();
        
        // Set up regular updates every 3 seconds
        this.updateTimer = setInterval(() => {
            this.requestStatusUpdate();
        }, 3000);
    }

    setupEventListeners() {
        // Service action buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('action-btn')) {
                e.stopPropagation();
                const service = e.target.dataset.service;
                const action = e.target.dataset.action;
                this.handleServiceAction(service, action, e.target);
            }
        });

        // Service row clicks (navigate to detail page)
        document.addEventListener('click', (e) => {
            const serviceRow = e.target.closest('tr[data-service]');
            if (serviceRow && !e.target.classList.contains('action-btn') && !e.target.closest('.actions')) {
                const serviceName = serviceRow.dataset.service;
                if (serviceName) {
                    window.location.href = `service.html?service=${serviceName}`;
                }
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'r' && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                this.requestStatusUpdate();
            }
        });
    }

    connectWebSocket() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/status`;
            
            this.websocket = new WebSocket(wsUrl);
            this.updateConnectionStatus('connecting');
            
            this.websocket.onopen = () => {
                console.log('WebSocket connected');
                this.updateConnectionStatus('connected');
                this.reconnectAttempts = 0;
                
                // Start updates now that WebSocket is connected
                this.startUpdates();
            };
            
            this.websocket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleWebSocketMessage(data);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };
            
            this.websocket.onclose = (event) => {
                console.log('WebSocket disconnected:', event.code, event.reason);
                this.updateConnectionStatus('disconnected');
                
                // Attempt to reconnect
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    setTimeout(() => {
                        this.reconnectAttempts++;
                        console.log(`Reconnecting WebSocket (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
                        this.connectWebSocket();
                    }, this.reconnectInterval);
                }
            };
            
            this.websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('disconnected');
            };
            
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            this.updateConnectionStatus('disconnected');
        }
    }

    updateConnectionStatus(status) {
        const statusIndicator = document.getElementById('connectionStatus');
        const statusText = statusIndicator.nextElementSibling;
        
        if (statusIndicator) {
            statusIndicator.className = `status-indicator ${status}`;
            
            switch (status) {
                case 'connected':
                    statusText.textContent = 'Live';
                    break;
                case 'connecting':
                    statusText.textContent = 'Connecting...';
                    break;
                case 'disconnected':
                    statusText.textContent = 'Disconnected';
                    break;
            }
        }
    }

    requestStatusUpdate() {
        // Request full status update for both services and daemon
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({ type: 'status_request' }));
        }
    }

    cleanup() {
        // Clean up timer when dashboard is destroyed
        if (this.updateTimer) {
            clearInterval(this.updateTimer);
            this.updateTimer = null;
        }
        if (this.websocket) {
            this.websocket.close();
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'status_update':
                this.updateLastRefreshTime();
                this.updateServiceCards(data.data);
                this.updateOverviewStats(data.data);
                break;
            case 'service_action_result':
                this.handleServiceActionResult(data.data);
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }

    updateServiceCards(statusData) {
        // Update all services (including daemon, orchestrator, mesh)
        this.services.forEach(serviceName => {
            const serviceData = statusData.services[serviceName];
            const row = document.querySelector(`tr[data-service="${serviceName}"]`);
            
            if (row && serviceData) {
                // Update status badge
                const statusBadge = document.getElementById(`${serviceName}-status`);
                if (statusBadge) {
                    statusBadge.textContent = serviceData.status;
                    statusBadge.className = `status-badge ${serviceData.status}`;
                }
                
                // Update table row class for styling
                row.className = serviceData.status;
                
                // Update basic details
                const portElement = document.getElementById(`${serviceName}-port`);
                if (portElement) {
                    portElement.textContent = serviceData.port || '-';
                }
                
                const pidElement = document.getElementById(`${serviceName}-pid`);
                if (pidElement) {
                    pidElement.textContent = serviceData.pid || '-';
                }
                
                // Update detailed metrics if available
                this.updateServiceDetails(serviceName, serviceData);
                
                // Update action buttons based on status (only for regular services, not core components)
                if (['identity', 'storage', 'node', 'ledger', 'social', 'indexer', 'analytics', 'wallet'].includes(serviceName)) {
                    this.updateServiceButtons(serviceName, serviceData.status);
                }
            }
        });
    }

    updateServiceDetails(serviceName, serviceData) {
        // Update detailed metrics from status data
        const uptimeElement = document.getElementById(`${serviceName}-uptime`);
        if (uptimeElement) {
            uptimeElement.textContent = serviceData.uptime || '-';
        }
        
        const cpuElement = document.getElementById(`${serviceName}-cpu`);
        if (cpuElement) {
            cpuElement.textContent = serviceData.cpuUsage || '-';
        }
        
        const memoryElement = document.getElementById(`${serviceName}-memory`);
        if (memoryElement) {
            memoryElement.textContent = serviceData.memoryUsage || '-';
        }
        
    }

    updateServiceButtons(serviceName, status) {
        const startBtn = document.querySelector(`[data-service="${serviceName}"][data-action="start"]`);
        const stopBtn = document.querySelector(`[data-service="${serviceName}"][data-action="stop"]`);
        const restartBtn = document.querySelector(`[data-service="${serviceName}"][data-action="restart"]`);
        
        if (status === 'running') {
            if (startBtn) startBtn.disabled = true;
            if (stopBtn) stopBtn.disabled = false;
            if (restartBtn) restartBtn.disabled = false;
        } else {
            if (startBtn) startBtn.disabled = false;
            if (stopBtn) stopBtn.disabled = true;
            if (restartBtn) restartBtn.disabled = true;
        }
    }

    async handleServiceAction(serviceName, action, buttonElement) {
        if (buttonElement.disabled || buttonElement.classList.contains('loading')) {
            return;
        }

        // Set loading state
        buttonElement.classList.add('loading');
        buttonElement.disabled = true;

        try {
            const response = await fetch(`/api/services/${serviceName}/${action}`, {
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
                this.showSuccess(`${action.charAt(0).toUpperCase() + action.slice(1)} ${serviceName} service successfully`);
                
                // Request status update via WebSocket
                setTimeout(() => {
                    this.requestStatusUpdate();
                }, 1000);
            } else {
                throw new Error(result.error || `Failed to ${action} ${serviceName} service`);
            }
        } catch (error) {
            console.error(`Failed to ${action} ${serviceName}:`, error);
            this.showError(`Failed to ${action} ${serviceName} service: ${error.message}`);
        } finally {
            // Remove loading state
            buttonElement.classList.remove('loading');
            buttonElement.disabled = false;
        }
    }

    handleServiceActionResult(data) {
        if (data.success) {
            this.showSuccess(data.message);
        } else {
            this.showError(data.error);
        }
    }

    updateOverviewStats(statusData) {
        const services = Object.values(statusData.services);
        const runningCount = services.filter(s => s.status === 'running').length;
        const downCount = services.filter(s => s.status !== 'running').length;
        
        const runningElement = document.getElementById('servicesRunning');
        const downElement = document.getElementById('servicesDown');
        const uptimeElement = document.getElementById('uptime');
        
        if (runningElement) {
            runningElement.textContent = runningCount;
            runningElement.style.color = runningCount === services.length ? '#22543d' : '#d53f8c';
        }
        
        if (downElement) {
            downElement.textContent = downCount;
            downElement.style.color = downCount === 0 ? '#22543d' : '#e53e3e';
        }
        
        if (uptimeElement) {
            uptimeElement.textContent = statusData.uptime || 'Unknown';
        }
    }

    updateCoreStatusFromServices(statusData) {
        // Extract daemon, orchestrator, and mesh data from the services response
        const daemonData = statusData.services.daemon;
        const orchestratorData = statusData.services.orchestrator;
        const meshData = statusData.services.mesh;
        
        if (daemonData) {
            this.updateStatusBadge('daemon-status', daemonData.status);
            this.updateCoreComponentMetricsFromService('daemon', daemonData);
        }
        
        if (orchestratorData) {
            this.updateStatusBadge('orchestrator-status', orchestratorData.status);
            this.updateCoreComponentMetricsFromService('orchestrator', orchestratorData);
        }
        
        if (meshData) {
            this.updateStatusBadge('mesh-status', meshData.status);
            this.updateCoreComponentMetricsFromService('mesh', meshData);
        }
    }

    updateCoreStatus(coreData) {
        // Update daemon status and metrics
        this.updateStatusBadge('daemon-status', coreData.daemon.status);
        this.updateCoreComponentMetrics('daemon', coreData.daemon);
        
        // Update orchestrator status and metrics
        this.updateStatusBadge('orchestrator-status', coreData.orchestrator.status);
        this.updateCoreComponentMetrics('orchestrator', coreData.orchestrator);
        
        // Update service mesh status and metrics
        this.updateStatusBadge('mesh-status', coreData.mesh.status);
        this.updateCoreComponentMetrics('mesh', coreData.mesh);
    }

    updateCoreComponentMetricsFromService(componentName, serviceData) {
        // Update using ServiceStatus structure
        if (componentName === 'daemon') {
            const portElement = document.getElementById(`${componentName}-port`);
            if (portElement) {
                portElement.textContent = serviceData.port || '-';
            }
            
            const pidElement = document.getElementById(`${componentName}-pid`);
            if (pidElement) {
                pidElement.textContent = serviceData.pid || '-';
            }
        }
        
        // Update uptime
        const uptimeElement = document.getElementById(`${componentName}-uptime`);
        if (uptimeElement) {
            uptimeElement.textContent = serviceData.uptime || '-';
        }
        
        // Update CPU and memory (if available in service data)
        const cpuElement = document.getElementById(`${componentName}-cpu`);
        if (cpuElement) {
            cpuElement.textContent = serviceData.cpuUsage || '-';
        }
        
        const memoryElement = document.getElementById(`${componentName}-memory`);
        if (memoryElement) {
            memoryElement.textContent = serviceData.memoryUsage || '-';
        }
    }

    updateCoreComponentMetrics(componentName, componentData) {
        // Update port (only for daemon)
        if (componentName === 'daemon') {
            const portElement = document.getElementById(`${componentName}-port`);
            if (portElement) {
                portElement.textContent = componentData.port || '-';
            }
            
            // Update PID (only for daemon)
            const pidElement = document.getElementById(`${componentName}-pid`);
            if (pidElement) {
                pidElement.textContent = componentData.pid || '-';
            }
        }
        
        // Update uptime
        const uptimeElement = document.getElementById(`${componentName}-uptime`);
        if (uptimeElement) {
            uptimeElement.textContent = componentData.uptime || '-';
        }
        
        // Update CPU usage
        const cpuElement = document.getElementById(`${componentName}-cpu`);
        if (cpuElement) {
            cpuElement.textContent = componentData.cpuUsage || '-';
        }
        
        // Update memory usage
        const memoryElement = document.getElementById(`${componentName}-memory`);
        if (memoryElement) {
            memoryElement.textContent = componentData.memoryUsage || '-';
        }
    }

    updateStatusBadge(elementId, status) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = status;
            element.className = `status-badge ${status}`;
        }
    }

    updateLastRefreshTime() {
        const now = new Date();
        const lastUpdatedElement = document.getElementById('lastUpdated');
        if (lastUpdatedElement) {
            lastUpdatedElement.textContent = now.toLocaleTimeString();
        }
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showNotification(message, type) {
        const notification = document.createElement('div');
        notification.className = `notification ${type}-message`;
        notification.textContent = message;
        
        const colors = {
            error: {
                bg: '#fed7d7',
                color: '#742a2a',
                border: '#feb2b2'
            },
            success: {
                bg: '#c6f6d5',
                color: '#22543d',
                border: '#9ae6b4'
            }
        };
        
        const style = colors[type] || colors.error;
        
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

    formatUptime(seconds) {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        
        if (days > 0) {
            return `${days}d ${hours}h ${minutes}m`;
        } else if (hours > 0) {
            return `${hours}h ${minutes}m`;
        } else {
            return `${minutes}m`;
        }
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new ServiceDashboard();
});