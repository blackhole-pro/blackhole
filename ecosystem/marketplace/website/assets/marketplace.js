// Blackhole Plugin Marketplace JavaScript

class PluginMarketplace {
    constructor() {
        this.plugins = [];
        this.filteredPlugins = [];
        this.init();
    }

    async init() {
        // Show loading state
        this.showLoading();
        
        // Fetch catalog
        await this.fetchCatalog();
        
        // Set up event listeners
        this.setupEventListeners();
        
        // Initial render
        this.filterAndRender();
    }

    async fetchCatalog() {
        try {
            const response = await fetch('api/v1/catalog.json');
            const data = await response.json();
            this.plugins = data.plugins || [];
        } catch (error) {
            console.error('Failed to fetch catalog:', error);
            this.showError('Failed to load plugins. Please try again later.');
        }
    }

    setupEventListeners() {
        const searchInput = document.getElementById('search');
        const categoryFilter = document.getElementById('category-filter');
        const officialOnly = document.getElementById('official-only');

        searchInput.addEventListener('input', () => this.filterAndRender());
        categoryFilter.addEventListener('change', () => this.filterAndRender());
        officialOnly.addEventListener('change', () => this.filterAndRender());
    }

    filterAndRender() {
        const searchTerm = document.getElementById('search').value.toLowerCase();
        const selectedCategory = document.getElementById('category-filter').value;
        const officialOnly = document.getElementById('official-only').checked;

        this.filteredPlugins = this.plugins.filter(plugin => {
            // Search filter
            const matchesSearch = !searchTerm || 
                plugin.name.toLowerCase().includes(searchTerm) ||
                plugin.description.toLowerCase().includes(searchTerm) ||
                (plugin.keywords && plugin.keywords.some(k => k.toLowerCase().includes(searchTerm)));

            // Category filter
            const matchesCategory = !selectedCategory || plugin.category === selectedCategory;

            // Official filter
            const matchesOfficial = !officialOnly || plugin.official;

            return matchesSearch && matchesCategory && matchesOfficial;
        });

        this.render();
    }

    render() {
        const grid = document.getElementById('plugin-grid');
        
        if (this.filteredPlugins.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <h3>No plugins found</h3>
                    <p>Try adjusting your filters or search terms</p>
                </div>
            `;
            return;
        }

        grid.innerHTML = this.filteredPlugins.map(plugin => this.renderPlugin(plugin)).join('');
        
        // Add click handlers
        grid.querySelectorAll('.plugin-card').forEach((card, index) => {
            card.addEventListener('click', (e) => {
                if (!e.target.classList.contains('plugin-install')) {
                    this.showPluginDetails(this.filteredPlugins[index]);
                }
            });
        });

        // Add install button handlers
        grid.querySelectorAll('.plugin-install').forEach((btn, index) => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.showInstallCommand(this.filteredPlugins[index]);
            });
        });
    }

    renderPlugin(plugin) {
        const stats = plugin.stats || {};
        const metadata = plugin.metadata || {};
        
        return `
            <div class="plugin-card" data-plugin-id="${plugin.id}">
                <div class="plugin-header">
                    <h3 class="plugin-title">${plugin.name}</h3>
                    <div class="plugin-badges">
                        ${plugin.official ? '<span class="badge badge-official">Official</span>' : ''}
                        <span class="badge badge-category">${plugin.category || 'uncategorized'}</span>
                    </div>
                </div>
                <p class="plugin-description">${plugin.description}</p>
                <div class="plugin-meta">
                    <div class="plugin-stats">
                        <span>v${plugin.version}</span>
                        ${stats.downloads ? `<span>↓ ${this.formatNumber(stats.downloads.total || 0)}</span>` : ''}
                        ${stats.stars ? `<span>★ ${this.formatNumber(stats.stars)}</span>` : ''}
                    </div>
                    <button class="plugin-install">Install</button>
                </div>
            </div>
        `;
    }

    showPluginDetails(plugin) {
        // In a real implementation, this would show a modal or navigate to a details page
        console.log('Show details for:', plugin);
        window.open(plugin.repository || plugin.documentation || '#', '_blank');
    }

    showInstallCommand(plugin) {
        const command = `blackhole plugin install ${plugin.id}@${plugin.version}`;
        
        // Try to copy to clipboard
        if (navigator.clipboard) {
            navigator.clipboard.writeText(command).then(() => {
                alert(`Install command copied to clipboard:\n\n${command}`);
            }).catch(() => {
                prompt('Copy this command:', command);
            });
        } else {
            prompt('Copy this command:', command);
        }
    }

    showLoading() {
        document.getElementById('plugin-grid').innerHTML = '<div class="loading">Loading plugins</div>';
    }

    showError(message) {
        document.getElementById('plugin-grid').innerHTML = `
            <div class="empty-state">
                <h3>Error</h3>
                <p>${message}</p>
            </div>
        `;
    }

    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'k';
        }
        return num.toString();
    }
}

// Initialize marketplace when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new PluginMarketplace();
});