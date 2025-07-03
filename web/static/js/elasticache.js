// GOV.UK Reports Dashboard - RDS Module JavaScript
// Handles RDS instances table, filtering, sorting, and data loading

class ElastiCachesPage {
    constructor() {
        this.caches = [];
        this.filteredCaches = [];
        this.currentSort = { field: 'name', direction: 'asc' };
        this.currentFilter = 'all';
        this.currentSearch = '';
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadElastiCacheData();
    }

    setupEventListeners() {
        // Action buttons
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                this.loadElastiCacheData();
            });
        }
    }

    async loadElastiCacheData() {
        this.showLoading();
        this.hideError();

        try {
            const summaryResponse = await fetch('/api/elasticache/clusters')

            if (!summaryResponse.ok) {
                throw new Error(`Failed to load summary: HTTP ${summaryResponse.status}`);
            }

            const summaryData = await summaryResponse.json();

            this.updateSummaryCards(summaryData);
            this.caches = summaryData.replication_groups
              .concat(summaryData.non_replicated_cache_clusters)
              .concat(summaryData.serverless_caches);
            this.filteredCaches = [...this.caches];
            
            this.renderCaches();
            this.showCachesTable();
            this.hideLoading();

        } catch (error) {
            console.error('Failed to load ElastiCache data:', error);
            this.showError(error.message);
            this.hideLoading();
        }
    }

    updateSummaryCards(data) {
        const totalCaches = (data.total_clusters + data.total_serverless_caches) || '0';
        const unappliedUpdates = data.unapplied_update_actions_summary.total_unapplied_updates || '0'; 
        const unappliedImportantUpdates = data.unapplied_update_actions_summary.total_unapplied_important_updates || '0';
        const unappliedCriticalUpdates =  data.unapplied_update_actions_summary.total_unapplied_critical_updates || '0';

        // Update summary metrics
        document.getElementById('total-elasticaches').textContent = totalCaches;
        document.getElementById('unapplied-updates').textContent = unappliedUpdates;
        document.getElementById('unapplied-important-updates').textContent = unappliedImportantUpdates;
        document.getElementById('unapplied-critical-updates').textContent = unappliedCriticalUpdates;
        
        // Update card styling based on values
        this.updateCardStyling('unapplied-critical-updates', unappliedCriticalUpdates);
        this.updateCardStyling('unapplied-important-updates', unappliedImportantUpdates);
    }

    updateCardStyling(elementId, value) {
        const element = document.getElementById(elementId);
        const card = element.closest('.elasticache-summary-card');
        
        if (value > 0) {
            if (elementId === 'unapplied-critical-updates') {
                card.classList.add('elasticache-summary-card--critical');
            } else if (elementId === 'unapplied-important-updates') {
                card.classList.add('elasticache-summary-card--warning');
            }
        } else {
            card.classList.remove('elasticache-summary-card--critical', 'elasticache-summary-card--warning');
        }
    }

    renderCaches() {
        const tbody = document.getElementById('caches-tbody');
        
        if (!tbody) return;

        // Clear existing content
        tbody.innerHTML = '';

        if (this.filteredCaches.length === 0) {
            //this.showNoResults();
            return;
        }

        // Sort instances
        this.SortInstances();

        // Render each instance
        this.filteredCaches.forEach(cache => {
            const row = this.createCacheRow(cache);
            tbody.appendChild(row);
        });
    }

    cacheType(cache) {
        var type = 'Unknown'

        if (Object.hasOwn(cache, 'replication_group_id')) {
            type = 'Replication Group'
        } else if (Object.hasOwn(cache, 'cache_cluster_id')) {
            type = 'Cache Cluster'
        } else if (Object.hasOwn(cache, 'serverless_cache_name')) {
            type = 'Serverless'
        }

        return type
    }

    engineVersion(cache) {
        var engineVersion = 'Unknown';

        switch (this.cacheType(cache)) {
            case 'Serverless':
                engineVersion = cache.full_engine_version;
                break;
            case 'Cache Cluster':
                engineVersion = cache.engine_version;
                break;
            case 'Replication Group':
                const allEngineVersions = [
                  ... new Set(cache.member_clusters.map((cluster) => cluster.engine_version))
                ];
                console.log("ALL ENGINE")
                console.log(allEngineVersions)
                allEngineVersions.sort()
                engineVersion = allEngineVersions.join(',')
                break;
        }

        return engineVersion
    }

    createCacheRow(cache) {
        console.log("WHAAAA")
        console.log(cache)
        const row = document.createElement('tr');
        row.className = 'govuk-table__row';
        
        var type = this.cacheType(cache)

        // Cache name
        const idCell = document.createElement('td');
        idCell.className = 'govuk-table__cell';
        idCell.textContent = cache.replication_group_id || cache.cache_cluster_id || cache.serverless_cache_name;
        
        // ElastiCache type
        const typeCell = document.createElement('td');
        typeCell.className = 'govuk-table__cell';
        typeCell.textContent = type
        
        // Engine
        const engCell = document.createElement('td');
        engCell.className = 'govuk-table__cell';
        const engTag = document.createElement('span');
        engTag.textContent = cache.engine
        engCell.appendChild(engTag);

        // Version
        const versionCell = document.createElement('td');
        versionCell.className = 'govuk-table__cell';
        var engineVersion = this.engineVersion(cache)
        versionCell.innerHTML = engineVersion
        
        // Critical Updates
        const criticalCell = document.createElement('td');
        criticalCell.className = 'govuk-table__cell';
        if (type === "Serverless" ) {
          criticalCell.innerHTML = "N/A";
        } else {
          criticalCell.innerHTML = cache.update_action_summary.total_unapplied_critical_updates
        }

        // Important Updates
        const importantCell = document.createElement('td');
        importantCell.className = 'govuk-table__cell';
        if (type === "Serverless" ) {
          importantCell.innerHTML = "N/A";
        } else {
          importantCell.innerHTML = cache.update_action_summary.total_unapplied_important_updates
        }
        
        // Total Updates
        const totalCell = document.createElement('td');
        totalCell.className = 'govuk-table__cell';
        if (type === "Serverless" ) {
          totalCell.innerHTML = "N/A";
        } else {
          totalCell.innerHTML = cache.update_action_summary.total_unapplied_updates
        }
        
        // Actions
        const actionsCell = document.createElement('td');
        actionsCell.className = 'govuk-table__cell';
        
        const viewButton = document.createElement('a');
        viewButton.className = 'govuk-button govuk-button--secondary govuk-button--small';
        viewButton.textContent = 'View Details';
        actionsCell.appendChild(viewButton);
        
        // Append all cells
        row.appendChild(idCell);
        row.appendChild(typeCell);
        row.appendChild(engCell);
        row.appendChild(versionCell);
        row.appendChild(criticalCell);
        row.appendChild(importantCell);
        row.appendChild(totalCell);
        
        return row;
    }

    SortInstances() {
        this.filteredCaches.sort((a, b) => {
            let aValue, bValue;
            
            switch (this.currentSort.field) {
                case 'name':
                    aValue = (a.cache_cluster_id || a.replication_group_id || a.serverless_cache_name).toLowerCase()
                    bValue = (b.cache_cluster_id || b.replication_group_id || b.serverless_cache_name).toLowerCase()
                    break;
                // case 'type':
                //     aValue = this.cacheType(aValue)
                //     bValue = this.cacheType(bValue)
                //     break;
                // case 'engine':
                //     aValue = (a.environment || '').toLowerCase();
                //     bValue = (b.environment || '').toLowerCase();
                //     break;
                // case 'version':
                //     aValue = this.engineVersion(a).toLowerCase()
                //     bValue = this.engineVersion(b).toLowerCase()
                //     break;
                // case 'critical_updates':
                //     aValue = a.unapplied_update_actions_summary.total_unapplied_critical_updates
                //     bValue = b.unapplied_update_actions_summary.total_unapplied_critical_updates
                //     break;
                // case 'important_updates':
                //     aValue = a.unapplied_update_actions_summary.total_unapplied_important_updates
                //     bValue = b.unapplied_update_actions_summary.total_unapplied_important_updates
                //     break;
                // case 'total_updates':
                //     aValue = a.unapplied_update_actions_summary.total_unapplied_updates
                //     bValue = b.unapplied_update_actions_summary.total_unapplied_updates
                //     break;
                default:
                    return 0;
            }
            
            if (aValue < bValue) return this.currentSort.direction === 'asc' ? -1 : 1;
            if (aValue > bValue) return this.currentSort.direction === 'asc' ? 1 : -1;
            return 0;
        });
    }

    showLoading() {
        const loadingState = document.getElementById('loading-state');
        if (loadingState) {
            loadingState.style.display = 'block';
        }
    }

    hideLoading() {
        const loadingState = document.getElementById('loading-state');
        if (loadingState) {
            loadingState.style.display = 'none';
        }
    }

    showError(message) {
        const errorState = document.getElementById('error-state');
        const errorMessage = document.getElementById('error-message');
        
        if (errorState) {
            errorState.style.display = 'block';
        }
        
        if (errorMessage) {
            errorMessage.textContent = message || 'Failed to load ElastiCache data. Please try again.';
        }
    }

    hideError() {
        const errorState = document.getElementById('error-state');
        if (errorState) {
            errorState.style.display = 'none';
        }
    }

    showCachesTable() {
        const container = document.getElementById('caches-container');
        if (container) {
            container.style.display = 'block';
        }
    }
}

// Initialize RDS page when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new ElastiCachesPage();
});

// Add comprehensive CSS for RDS styling
const elastiCacheStyle = document.createElement('style');
elastiCacheStyle.textContent = `
    /* ElastiCache Summary Cards */
    .elasticache-summary-card {
        border: 2px solid #b1b4b6;
        border-radius: 4px;
        padding: 20px;
        margin-bottom: 20px;
        background: #ffffff;
        text-align: center;
    }
    
    .elasticache-summary-card--success {
        border-color: #00703c;
        background-color: #f3fff3;
    }
    
    .elasticache-summary-card--warning {
        border-color: #f47738;
        background-color: #fff8f0;
    }
    
    .elasticache-summary-card--critical {
        border-color: #d4351c;
        background-color: #fff5f5;
    }
    
    .elasticache-metric-value {
        font-size: 32px;
        font-weight: bold;
        margin: 10px 0;
        color: #0b0c0c;
    }
    
    .elasticache-metric-value.success {
        color: #00703c;
    }
    
    .elasticache-metric-value.warning {
        color: #f47738;
    }
    
    .elasticache-metric-value.critical {
        color: #d4351c;
    }
    
    .elasticache-metric-subtitle {
        font-size: 14px;
        color: #505a5f;
        margin: 0;
    }
    
    /* Loading and error states */
    .loading-container {
        text-align: center;
        padding: 40px;
    }
    
    .loading-spinner {
        border: 4px solid #f3f2f1;
        border-top: 4px solid #1d70b8;
        border-radius: 50%;
        width: 40px;
        height: 40px;
        animation: spin 1s linear infinite;
        margin: 0 auto 20px;
    }
    
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
    
    .error-container {
        margin: 20px 0;
    }
    
    /* Responsive design */
    @media (max-width: 768px) {
        .filter-buttons {
            flex-direction: column;
        }
        
        .action-buttons {
            flex-direction: column;
        }
        
        .version-distribution {
            grid-template-columns: 1fr;
        }
        
        .rds-instances-table {
            font-size: 14px;
        }
        
        .govuk-table__cell {
            padding: 8px 4px;
        }
    }
`;
document.head.appendChild(elastiCacheStyle);
