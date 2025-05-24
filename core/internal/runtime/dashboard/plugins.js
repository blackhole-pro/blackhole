class PluginDashboard {
    constructor() {
        this.websocket = null;
        this.plugins = new Map();
        this.availablePlugins = new Map();
        this.currentPlugin = null;
        this.uploadedFile = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.connectWebSocket();
        this.setupDragAndDrop();
    }

    setupEventListeners() {
        // Plugin action buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('plugin-action')) {
                e.stopPropagation();
                const pluginId = e.target.dataset.plugin;
                const action = e.target.dataset.action;
                this.handlePluginAction(pluginId, action);
            }
        });

        // File input
        const fileInput = document.getElementById('pluginFile');
        if (fileInput) {
            fileInput.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(e.target.files[0]);
                }
            });
        }
    }

    setupDragAndDrop() {
        const uploadArea = document.getElementById('uploadArea');
        if (!uploadArea) return;

        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            uploadArea.addEventListener(eventName, preventDefaults, false);
        });

        function preventDefaults(e) {
            e.preventDefault();
            e.stopPropagation();
        }

        ['dragenter', 'dragover'].forEach(eventName => {
            uploadArea.addEventListener(eventName, () => {
                uploadArea.classList.add('drag-over');
            }, false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            uploadArea.addEventListener(eventName, () => {
                uploadArea.classList.remove('drag-over');
            }, false);
        });

        uploadArea.addEventListener('drop', (e) => {
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileSelect(files[0]);
            }
        }, false);
    }

    connectWebSocket() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/plugins`;
            
            this.websocket = new WebSocket(wsUrl);
            this.updateConnectionStatus('connecting');
            
            this.websocket.onopen = () => {
                console.log('WebSocket connected');
                this.updateConnectionStatus('connected');
                this.requestPluginStatus();
            };
            
            this.websocket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleWebSocketMessage(data);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };
            
            this.websocket.onclose = () => {
                console.log('WebSocket disconnected');
                this.updateConnectionStatus('disconnected');
                // Attempt reconnect after 5 seconds
                setTimeout(() => this.connectWebSocket(), 5000);
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

    requestPluginStatus() {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({ type: 'plugin_status_request' }));
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'plugin_status':
                this.updatePluginDisplay(data.data);
                break;
            case 'plugin_action_result':
                this.handleActionResult(data.data);
                break;
            case 'hot_swap_progress':
                this.updateHotSwapProgress(data.data);
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }

    updatePluginDisplay(data) {
        // Update stats
        document.getElementById('pluginsLoaded').textContent = data.loaded || 0;
        document.getElementById('pluginsRunning').textContent = data.running || 0;
        document.getElementById('pluginsAvailable').textContent = data.available || 0;
        document.getElementById('lastUpdated').textContent = new Date().toLocaleTimeString();

        // Update loaded plugins table
        this.updateLoadedPlugins(data.plugins || []);
        
        // Update available plugins
        this.updateAvailablePlugins(data.availablePlugins || []);
    }

    updateLoadedPlugins(plugins) {
        const tbody = document.getElementById('loadedPlugins');
        tbody.innerHTML = '';

        plugins.forEach(plugin => {
            this.plugins.set(plugin.id, plugin);
            
            const row = document.createElement('tr');
            row.className = plugin.status.toLowerCase();
            row.innerHTML = `
                <td>
                    <strong>${plugin.name}</strong>
                    <span class="plugin-id">${plugin.id}</span>
                </td>
                <td>${plugin.version}</td>
                <td><span class="status-badge ${plugin.status.toLowerCase()}">${plugin.status}</span></td>
                <td><span class="type-badge">${plugin.type || 'Process'}</span></td>
                <td>${this.formatUptime(plugin.uptime)}</td>
                <td>
                    <div class="resource-info">
                        <span>CPU: ${plugin.resources?.cpu || '-'}</span>
                        <span>Mem: ${plugin.resources?.memory || '-'}</span>
                    </div>
                </td>
                <td>
                    <button class="btn btn-small plugin-action" data-plugin="${plugin.id}" data-action="view-state">
                        View State
                    </button>
                </td>
                <td class="actions">
                    ${this.getPluginActions(plugin)}
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    updateAvailablePlugins(plugins) {
        const container = document.getElementById('availablePlugins');
        container.innerHTML = '';

        plugins.forEach(plugin => {
            this.availablePlugins.set(plugin.id, plugin);
            
            const card = document.createElement('div');
            card.className = 'plugin-card';
            card.innerHTML = `
                <div class="plugin-card-header">
                    <h4>${plugin.name}</h4>
                    <span class="version">${plugin.version}</span>
                </div>
                <div class="plugin-card-body">
                    <p>${plugin.description || 'No description available'}</p>
                    <div class="plugin-meta">
                        <span>Author: ${plugin.author || 'Unknown'}</span>
                        <span>Size: ${this.formatSize(plugin.size)}</span>
                    </div>
                </div>
                <div class="plugin-card-footer">
                    <button class="btn btn-primary btn-small plugin-action" 
                            data-plugin="${plugin.id}" 
                            data-action="load">
                        Load Plugin
                    </button>
                </div>
            `;
            container.appendChild(card);
        });
    }

    getPluginActions(plugin) {
        const actions = [];
        
        if (plugin.status === 'Running') {
            actions.push(`
                <button class="action-btn plugin-action" data-plugin="${plugin.id}" data-action="stop" title="Stop">⏹</button>
                <button class="action-btn plugin-action" data-plugin="${plugin.id}" data-action="restart" title="Restart">↻</button>
                <button class="action-btn plugin-action" data-plugin="${plugin.id}" data-action="hot-swap" title="Hot Swap">🔄</button>
            `);
        } else {
            actions.push(`
                <button class="action-btn plugin-action" data-plugin="${plugin.id}" data-action="start" title="Start">▶</button>
            `);
        }
        
        actions.push(`
            <button class="action-btn plugin-action" data-plugin="${plugin.id}" data-action="unload" title="Unload">🗑</button>
        `);
        
        return actions.join('');
    }

    async handlePluginAction(pluginId, action) {
        console.log(`Plugin action: ${action} for ${pluginId}`);
        
        switch (action) {
            case 'view-state':
                this.viewPluginState(pluginId);
                break;
            case 'hot-swap':
                this.showHotSwapDialog(pluginId);
                break;
            case 'load':
                await this.loadPlugin(pluginId);
                break;
            case 'unload':
                await this.unloadPlugin(pluginId);
                break;
            case 'start':
            case 'stop':
            case 'restart':
                await this.controlPlugin(pluginId, action);
                break;
        }
    }

    async loadPlugin(pluginId) {
        try {
            const response = await fetch(`/api/plugins/${pluginId}/load`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            
            const result = await response.json();
            if (result.success) {
                this.showSuccess(`Plugin ${pluginId} loaded successfully`);
                this.requestPluginStatus();
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Failed to load plugin: ${error.message}`);
        }
    }

    async unloadPlugin(pluginId) {
        if (!confirm(`Are you sure you want to unload plugin ${pluginId}?`)) {
            return;
        }
        
        try {
            const response = await fetch(`/api/plugins/${pluginId}/unload`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            
            const result = await response.json();
            if (result.success) {
                this.showSuccess(`Plugin ${pluginId} unloaded successfully`);
                this.requestPluginStatus();
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Failed to unload plugin: ${error.message}`);
        }
    }

    async controlPlugin(pluginId, action) {
        try {
            const response = await fetch(`/api/plugins/${pluginId}/${action}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            
            const result = await response.json();
            if (result.success) {
                this.showSuccess(`Plugin ${pluginId} ${action}ed successfully`);
                this.requestPluginStatus();
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Failed to ${action} plugin: ${error.message}`);
        }
    }

    async viewPluginState(pluginId) {
        try {
            const response = await fetch(`/api/plugins/${pluginId}/state`);
            const result = await response.json();
            
            if (result.success) {
                this.currentPlugin = pluginId;
                document.getElementById('stateViewer').textContent = 
                    JSON.stringify(result.state, null, 2);
                document.getElementById('stateDialog').style.display = 'block';
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Failed to get plugin state: ${error.message}`);
        }
    }

    showHotSwapDialog(pluginId) {
        const plugin = this.plugins.get(pluginId);
        if (!plugin) return;
        
        this.currentPlugin = pluginId;
        document.getElementById('currentPluginName').textContent = plugin.name;
        document.getElementById('currentPluginVersion').textContent = plugin.version;
        
        // TODO: Get available versions and state size
        document.getElementById('hotSwapDialog').style.display = 'block';
    }

    async performHotSwap() {
        const newVersion = document.getElementById('newPluginVersion').value;
        if (!newVersion) {
            this.showError('Please select a new version');
            return;
        }
        
        document.getElementById('swapProgress').style.display = 'block';
        document.getElementById('swapBtn').disabled = true;
        
        try {
            const response = await fetch(`/api/plugins/${this.currentPlugin}/hot-swap`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ version: newVersion })
            });
            
            const result = await response.json();
            if (result.success) {
                this.showSuccess('Hot swap completed successfully');
                this.closeHotSwapDialog();
                this.requestPluginStatus();
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Hot swap failed: ${error.message}`);
        } finally {
            document.getElementById('swapBtn').disabled = false;
        }
    }

    updateHotSwapProgress(data) {
        const progressFill = document.getElementById('progressFill');
        const progressStatus = document.getElementById('progressStatus');
        
        if (progressFill) {
            progressFill.style.width = `${data.progress}%`;
        }
        
        if (progressStatus) {
            progressStatus.textContent = data.status;
        }
    }

    handleFileSelect(file) {
        if (!file.name.match(/\.(zip|tar\.gz)$/)) {
            this.showError('Invalid file type. Please select a .zip or .tar.gz file');
            return;
        }
        
        this.uploadedFile = file;
        document.getElementById('uploadPluginName').textContent = file.name;
        document.getElementById('uploadPluginSize').textContent = this.formatSize(file.size);
        document.getElementById('uploadDetails').style.display = 'block';
        document.getElementById('uploadBtn').disabled = false;
        
        // TODO: Extract and display plugin metadata
    }

    async uploadPlugin() {
        if (!this.uploadedFile) return;
        
        const formData = new FormData();
        formData.append('plugin', this.uploadedFile);
        
        document.getElementById('uploadBtn').disabled = true;
        document.getElementById('uploadBtn').textContent = 'Uploading...';
        
        try {
            const response = await fetch('/api/plugins/upload', {
                method: 'POST',
                body: formData
            });
            
            const result = await response.json();
            if (result.success) {
                this.showSuccess('Plugin uploaded successfully');
                this.closeUploadDialog();
                this.requestPluginStatus();
            } else {
                throw new Error(result.error);
            }
        } catch (error) {
            this.showError(`Upload failed: ${error.message}`);
        } finally {
            document.getElementById('uploadBtn').disabled = false;
            document.getElementById('uploadBtn').textContent = 'Upload';
        }
    }

    // UI Helper Functions
    formatUptime(seconds) {
        if (!seconds) return '-';
        
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        
        if (days > 0) {
            return `${days}d ${hours}h`;
        } else if (hours > 0) {
            return `${hours}h ${minutes}m`;
        } else {
            return `${minutes}m`;
        }
    }

    formatSize(bytes) {
        if (!bytes) return '0 B';
        
        const units = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${units[i]}`;
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
}

// Modal functions
function showUploadDialog() {
    document.getElementById('uploadDialog').style.display = 'block';
}

function closeUploadDialog() {
    document.getElementById('uploadDialog').style.display = 'none';
    document.getElementById('uploadDetails').style.display = 'none';
    document.getElementById('uploadBtn').disabled = true;
    document.getElementById('pluginFile').value = '';
}

function closeHotSwapDialog() {
    document.getElementById('hotSwapDialog').style.display = 'none';
    document.getElementById('swapProgress').style.display = 'none';
}

function closeStateDialog() {
    document.getElementById('stateDialog').style.display = 'none';
}

function showMarketplace() {
    // TODO: Implement marketplace view
    alert('Marketplace coming soon!');
}

function refreshPlugins() {
    if (window.pluginDashboard) {
        window.pluginDashboard.requestPluginStatus();
    }
}

function exportState() {
    const state = document.getElementById('stateViewer').textContent;
    const blob = new Blob([state], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `plugin-state-${window.pluginDashboard.currentPlugin}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function importState() {
    // TODO: Implement state import
    alert('State import coming soon!');
}

function refreshState() {
    if (window.pluginDashboard && window.pluginDashboard.currentPlugin) {
        window.pluginDashboard.viewPluginState(window.pluginDashboard.currentPlugin);
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.pluginDashboard = new PluginDashboard();
});