// GOV.UK Reports Dashboard - RDS Module JavaScript
// Handles RDS instances table, filtering, sorting, and data loading

class RDSInstancesPage {
    constructor() {
        this.instances = [];
        this.filteredInstances = [];
        this.currentSort = { field: 'instance_id', direction: 'asc' };
        this.currentFilter = 'all';
        this.currentSearch = '';
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadRDSData();
    }

    setupEventListeners() {
        // Search functionality
        const searchInput = document.getElementById('search-instances');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.handleSearch(e.target.value);
            });
        }

        // Filter buttons
        const filterButtons = document.querySelectorAll('[data-filter]');
        filterButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleFilter(e.target.dataset.filter);
                this.setActiveFilter(e.target);
            });
        });

        // Sort functionality
        const sortableHeaders = document.querySelectorAll('.sortable');
        sortableHeaders.forEach(header => {
            header.addEventListener('click', (e) => {
                this.handleSort(e.currentTarget.dataset.sort);
            });
        });

        // Action buttons
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                this.loadRDSData();
            });
        }

        const refreshButton = document.getElementById('refresh-data');
        if (refreshButton) {
            refreshButton.addEventListener('click', () => {
                this.loadRDSData();
            });
        }

        const clearFiltersButton = document.getElementById('clear-filters');
        if (clearFiltersButton) {
            clearFiltersButton.addEventListener('click', () => {
                this.clearAllFilters();
            });
        }
    }

    async loadRDSData() {
        this.showLoading();
        this.hideError();

        try {
            // Load summary and instances data in parallel
            const [summaryResponse, instancesResponse] = await Promise.all([
                fetch('/api/rds/summary'),
                fetch('/api/rds/instances')
            ]);

            if (!summaryResponse.ok) {
                throw new Error(`Failed to load summary: HTTP ${summaryResponse.status}`);
            }
            if (!instancesResponse.ok) {
                throw new Error(`Failed to load instances: HTTP ${instancesResponse.status}`);
            }

            const summaryData = await summaryResponse.json();
            const instancesData = await instancesResponse.json();

            this.updateSummaryCards(summaryData);
            this.instances = instancesData.instances || [];
            this.filteredInstances = [...this.instances];
            
            this.renderInstances();
            this.renderVersionChart();
            this.showInstancesTable();
            this.hideLoading();

        } catch (error) {
            console.error('Failed to load RDS data:', error);
            this.showError(error.message);
            this.hideLoading();
        }
    }

    updateSummaryCards(data) {
        // Update summary metrics
        document.getElementById('total-instances').textContent = data.postgresql_count || '0';
        document.getElementById('eol-instances').textContent = data.eol_instances || '0';
        document.getElementById('outdated-instances').textContent = data.outdated_instances || '0';
        
        // Calculate and display compliance percentage
        const total = data.postgresql_count || 0;
        const eol = data.eol_instances || 0;
        const outdated = data.outdated_instances || 0;
        const compliant = total - eol - outdated;
        const compliancePercentage = total > 0 ? ((compliant / total) * 100).toFixed(1) : '0';
        document.getElementById('compliance-rate').textContent = `${compliancePercentage}%`;

        // Update card styling based on values
        this.updateCardStyling('eol-instances', eol);
        this.updateCardStyling('outdated-instances', outdated);
    }

    updateCardStyling(elementId, value) {
        const element = document.getElementById(elementId);
        const card = element.closest('.rds-summary-card');
        
        if (value > 0) {
            if (elementId === 'eol-instances') {
                card.classList.add('rds-summary-card--critical');
            } else if (elementId === 'outdated-instances') {
                card.classList.add('rds-summary-card--warning');
            }
        } else {
            card.classList.remove('rds-summary-card--critical', 'rds-summary-card--warning');
        }
    }

    renderInstances() {
        const tbody = document.getElementById('instances-tbody');
        
        if (!tbody) return;

        // Clear existing content
        tbody.innerHTML = '';

        if (this.filteredInstances.length === 0) {
            this.showNoResults();
            return;
        }

        this.hideNoResults();

        // Sort instances
        this.sortInstances();

        // Render each instance
        this.filteredInstances.forEach(instance => {
            const row = this.createInstanceRow(instance);
            tbody.appendChild(row);
        });

        // Update footer stats
        this.updateTableFooter();
    }

    createInstanceRow(instance) {
        const row = document.createElement('tr');
        row.className = 'govuk-table__row';
        
        // Apply styling based on compliance status
        const complianceStatus = this.getComplianceStatus(instance);
        if (complianceStatus === 'eol') {
            row.classList.add('rds-row--critical');
        } else if (complianceStatus === 'outdated') {
            row.classList.add('rds-row--warning');
        }
        
        // Instance ID (with link to detail page)
        const idCell = document.createElement('td');
        idCell.className = 'govuk-table__cell';
        const idLink = document.createElement('a');
        idLink.href = `/rds/${encodeURIComponent(instance.instance_id)}`;
        idLink.className = 'govuk-link';
        idLink.textContent = instance.instance_id;
        idCell.appendChild(idLink);
        
        // Application
        const appCell = document.createElement('td');
        appCell.className = 'govuk-table__cell';
        appCell.textContent = instance.application || 'Unknown';
        
        // Environment
        const envCell = document.createElement('td');
        envCell.className = 'govuk-table__cell';
        const envTag = document.createElement('span');
        envTag.className = `environment-tag environment-tag--${(instance.environment || 'unknown').toLowerCase()}`;
        envTag.textContent = instance.environment || 'Unknown';
        envCell.appendChild(envTag);
        
        // Version
        const versionCell = document.createElement('td');
        versionCell.className = 'govuk-table__cell';
        versionCell.innerHTML = `
            <span class="version-info">
                ${instance.version}
                <small class="version-major">(v${instance.major_version})</small>
            </span>
        `;
        
        // Compliance Status
        const complianceCell = document.createElement('td');
        complianceCell.className = 'govuk-table__cell';
        const complianceTag = this.createComplianceTag(instance);
        complianceCell.appendChild(complianceTag);
        
        // Instance Class
        const classCell = document.createElement('td');
        classCell.className = 'govuk-table__cell';
        classCell.textContent = instance.instance_class || 'N/A';
        
        // Region
        const regionCell = document.createElement('td');
        regionCell.className = 'govuk-table__cell';
        regionCell.textContent = instance.region || 'N/A';
        
        // Status
        const statusCell = document.createElement('td');
        statusCell.className = 'govuk-table__cell';
        const statusTag = document.createElement('span');
        statusTag.className = `status-tag status-tag--${instance.status.toLowerCase()}`;
        statusTag.textContent = instance.status;
        statusCell.appendChild(statusTag);
        
        // Actions
        const actionsCell = document.createElement('td');
        actionsCell.className = 'govuk-table__cell';
        
        const viewButton = document.createElement('a');
        viewButton.href = `/rds/${encodeURIComponent(instance.instance_id)}`;
        viewButton.className = 'govuk-button govuk-button--secondary govuk-button--small';
        viewButton.textContent = 'View Details';
        actionsCell.appendChild(viewButton);
        
        // Append all cells
        row.appendChild(idCell);
        row.appendChild(appCell);
        row.appendChild(envCell);
        row.appendChild(versionCell);
        row.appendChild(complianceCell);
        row.appendChild(classCell);
        row.appendChild(regionCell);
        row.appendChild(statusCell);
        row.appendChild(actionsCell);
        
        return row;
    }

    createComplianceTag(instance) {
        const tag = document.createElement('span');
        
        if (instance.is_eol) {
            tag.className = 'compliance-tag compliance-tag--critical';
            tag.textContent = 'End-of-Life';
            tag.title = 'This version is end-of-life and should be upgraded immediately';
        } else {
            // Could add logic for outdated detection here
            const isOutdated = this.isInstanceOutdated(instance);
            if (isOutdated) {
                tag.className = 'compliance-tag compliance-tag--warning';
                tag.textContent = 'Outdated';
                tag.title = 'This version is outdated and should be updated';
            } else {
                tag.className = 'compliance-tag compliance-tag--success';
                tag.textContent = 'Compliant';
                tag.title = 'This version is current and compliant';
            }
        }
        
        return tag;
    }

    isInstanceOutdated(instance) {
        // Simple heuristic: versions below 13 are considered outdated (but not EOL)
        const majorVersion = parseInt(instance.major_version);
        return !instance.is_eol && majorVersion < 13;
    }

    getComplianceStatus(instance) {
        if (instance.is_eol) return 'eol';
        if (this.isInstanceOutdated(instance)) return 'outdated';
        return 'compliant';
    }

    handleSearch(searchTerm) {
        this.currentSearch = searchTerm.toLowerCase().trim();
        this.applyFiltersAndSearch();
    }

    handleFilter(filter) {
        this.currentFilter = filter;
        this.applyFiltersAndSearch();
    }

    applyFiltersAndSearch() {
        let filtered = [...this.instances];
        
        // Apply filter
        if (this.currentFilter !== 'all') {
            filtered = filtered.filter(instance => {
                const status = this.getComplianceStatus(instance);
                return status === this.currentFilter;
            });
        }
        
        // Apply search
        if (this.currentSearch) {
            filtered = filtered.filter(instance => 
                instance.instance_id.toLowerCase().includes(this.currentSearch) ||
                (instance.application || '').toLowerCase().includes(this.currentSearch) ||
                (instance.environment || '').toLowerCase().includes(this.currentSearch) ||
                instance.version.toLowerCase().includes(this.currentSearch) ||
                instance.major_version.toLowerCase().includes(this.currentSearch) ||
                (instance.region || '').toLowerCase().includes(this.currentSearch)
            );
        }
        
        this.filteredInstances = filtered;
        this.renderInstances();
    }

    handleSort(field) {
        if (this.currentSort.field === field) {
            // Toggle direction
            this.currentSort.direction = this.currentSort.direction === 'asc' ? 'desc' : 'asc';
        } else {
            // New field, default to ascending
            this.currentSort.field = field;
            this.currentSort.direction = 'asc';
        }
        
        this.updateSortIndicators();
        this.renderInstances();
    }

    sortInstances() {
        this.filteredInstances.sort((a, b) => {
            let aValue, bValue;
            
            switch (this.currentSort.field) {
                case 'instance_id':
                    aValue = a.instance_id.toLowerCase();
                    bValue = b.instance_id.toLowerCase();
                    break;
                case 'application':
                    aValue = (a.application || '').toLowerCase();
                    bValue = (b.application || '').toLowerCase();
                    break;
                case 'environment':
                    aValue = (a.environment || '').toLowerCase();
                    bValue = (b.environment || '').toLowerCase();
                    break;
                case 'version':
                    aValue = a.version.toLowerCase();
                    bValue = b.version.toLowerCase();
                    break;
                case 'instance_class':
                    aValue = (a.instance_class || '').toLowerCase();
                    bValue = (b.instance_class || '').toLowerCase();
                    break;
                case 'region':
                    aValue = (a.region || '').toLowerCase();
                    bValue = (b.region || '').toLowerCase();
                    break;
                default:
                    return 0;
            }
            
            if (aValue < bValue) return this.currentSort.direction === 'asc' ? -1 : 1;
            if (aValue > bValue) return this.currentSort.direction === 'asc' ? 1 : -1;
            return 0;
        });
    }

    updateSortIndicators() {
        // Reset all sort indicators
        document.querySelectorAll('.sortable').forEach(header => {
            header.classList.remove('asc', 'desc');
        });
        
        // Set current sort indicator
        const currentHeader = document.querySelector(`[data-sort="${this.currentSort.field}"]`);
        if (currentHeader) {
            currentHeader.classList.add(this.currentSort.direction);
        }
    }

    setActiveFilter(activeButton) {
        // Remove active class from all filter buttons
        document.querySelectorAll('[data-filter]').forEach(button => {
            button.classList.remove('active');
        });
        
        // Add active class to clicked button
        activeButton.classList.add('active');
    }

    clearAllFilters() {
        this.currentFilter = 'all';
        this.currentSearch = '';
        
        // Reset UI elements
        document.getElementById('search-instances').value = '';
        this.setActiveFilter(document.getElementById('filter-all'));
        
        // Reapply filters
        this.applyFiltersAndSearch();
    }

    updateTableFooter() {
        document.getElementById('visible-count').textContent = this.filteredInstances.length;
        document.getElementById('total-count').textContent = this.instances.length;
        
        const filterInfo = document.getElementById('filter-info');
        if (this.currentFilter !== 'all' || this.currentSearch) {
            let filterText = '';
            if (this.currentFilter !== 'all') {
                filterText += ` • Filtered by: ${this.currentFilter}`;
            }
            if (this.currentSearch) {
                filterText += ` • Search: "${this.currentSearch}"`;
            }
            filterInfo.textContent = filterText;
        } else {
            filterInfo.textContent = '';
        }
    }

    renderVersionChart() {
        // Simple version distribution display
        const chartContainer = document.getElementById('version-chart-container');
        const versionSummary = document.getElementById('version-summary');
        
        if (!chartContainer || !versionSummary) return;
        
        // Count versions
        const versionCounts = {};
        this.instances.forEach(instance => {
            const version = instance.major_version;
            versionCounts[version] = (versionCounts[version] || 0) + 1;
        });
        
        // Create version summary
        let summaryHTML = '<div class="version-distribution">';
        Object.entries(versionCounts)
            .sort((a, b) => b[0] - a[0]) // Sort by version descending
            .forEach(([version, count]) => {
                const percentage = ((count / this.instances.length) * 100).toFixed(1);
                summaryHTML += `
                    <div class="version-item">
                        <span class="version-label">PostgreSQL ${version}</span>
                        <span class="version-count">${count} instances (${percentage}%)</span>
                    </div>
                `;
            });
        summaryHTML += '</div>';
        
        versionSummary.innerHTML = summaryHTML;
        chartContainer.style.display = 'block';
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
            errorMessage.textContent = message || 'Failed to load RDS data. Please try again.';
        }
    }

    hideError() {
        const errorState = document.getElementById('error-state');
        if (errorState) {
            errorState.style.display = 'none';
        }
    }

    showInstancesTable() {
        const container = document.getElementById('instances-container');
        if (container) {
            container.style.display = 'block';
        }
    }

    showNoResults() {
        const noResults = document.getElementById('no-results');
        if (noResults) {
            noResults.style.display = 'block';
        }
    }

    hideNoResults() {
        const noResults = document.getElementById('no-results');
        if (noResults) {
            noResults.style.display = 'none';
        }
    }
}

// Initialize RDS page when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new RDSInstancesPage();
});

// Add comprehensive CSS for RDS styling
const rdsStyle = document.createElement('style');
rdsStyle.textContent = `
    /* RDS Summary Cards */
    .rds-summary-card {
        border: 2px solid #b1b4b6;
        border-radius: 4px;
        padding: 20px;
        margin-bottom: 20px;
        background: #ffffff;
        text-align: center;
    }
    
    .rds-summary-card--success {
        border-color: #00703c;
        background-color: #f3fff3;
    }
    
    .rds-summary-card--warning {
        border-color: #f47738;
        background-color: #fff8f0;
    }
    
    .rds-summary-card--critical {
        border-color: #d4351c;
        background-color: #fff5f5;
    }
    
    .rds-metric-value {
        font-size: 32px;
        font-weight: bold;
        margin: 10px 0;
        color: #0b0c0c;
    }
    
    .rds-metric-value.success {
        color: #00703c;
    }
    
    .rds-metric-value.warning {
        color: #f47738;
    }
    
    .rds-metric-value.critical {
        color: #d4351c;
    }
    
    .rds-metric-subtitle {
        font-size: 14px;
        color: #505a5f;
        margin: 0;
    }
    
    /* RDS Filters */
    .rds-filters {
        padding: 20px;
        background: #f8f8f8;
        border-radius: 4px;
        margin-bottom: 20px;
    }
    
    .filter-buttons {
        margin-top: 15px;
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }
    
    .filter-buttons .govuk-button {
        margin: 0;
    }
    
    .filter-buttons .govuk-button.active {
        background-color: #1d70b8;
        color: white;
        border-color: #1d70b8;
    }
    
    /* RDS Table Styling */
    .rds-instances-table {
        margin-top: 0;
    }
    
    .rds-row--critical {
        background-color: #fff5f5;
        border-left: 4px solid #d4351c;
    }
    
    .rds-row--warning {
        background-color: #fff8f0;
        border-left: 4px solid #f47738;
    }
    
    /* Compliance Tags */
    .compliance-tag {
        padding: 4px 8px;
        border-radius: 3px;
        font-size: 12px;
        font-weight: bold;
        text-transform: uppercase;
        display: inline-block;
    }
    
    .compliance-tag--success {
        background-color: #00703c;
        color: white;
    }
    
    .compliance-tag--warning {
        background-color: #f47738;
        color: white;
    }
    
    .compliance-tag--critical {
        background-color: #d4351c;
        color: white;
    }
    
    /* Environment Tags */
    .environment-tag {
        padding: 2px 6px;
        border-radius: 3px;
        font-size: 11px;
        font-weight: bold;
        text-transform: uppercase;
        display: inline-block;
    }
    
    .environment-tag--production {
        background-color: #d4351c;
        color: white;
    }
    
    .environment-tag--staging {
        background-color: #f47738;
        color: white;
    }
    
    .environment-tag--development {
        background-color: #1d70b8;
        color: white;
    }
    
    .environment-tag--test {
        background-color: #7a2e8d;
        color: white;
    }
    
    .environment-tag--unknown {
        background-color: #505a5f;
        color: white;
    }
    
    /* Status Tags */
    .status-tag {
        padding: 2px 6px;
        border-radius: 3px;
        font-size: 11px;
        font-weight: bold;
        display: inline-block;
    }
    
    .status-tag--available {
        background-color: #00703c;
        color: white;
    }
    
    .status-tag--stopped {
        background-color: #d4351c;
        color: white;
    }
    
    .status-tag--starting {
        background-color: #f47738;
        color: white;
    }
    
    .status-tag--stopping {
        background-color: #f47738;
        color: white;
    }
    
    /* Version Information */
    .version-info {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
    }
    
    .version-major {
        color: #505a5f;
        font-size: 11px;
        margin-top: 2px;
    }
    
    /* Small buttons */
    .govuk-button--small {
        font-size: 14px;
        padding: 5px 10px;
    }
    
    /* Action buttons */
    .action-buttons {
        margin: 20px 0;
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }
    
    .action-buttons .govuk-button {
        margin: 0;
    }
    
    /* Table footer */
    .table-footer {
        margin-top: 15px;
        padding: 10px 0;
        border-top: 1px solid #b1b4b6;
    }
    
    /* No results state */
    .no-results {
        text-align: center;
        padding: 40px;
        background: #f8f8f8;
        border-radius: 4px;
        margin-top: 20px;
    }
    
    /* Sort indicators */
    .sortable {
        cursor: pointer;
        position: relative;
    }
    
    .sortable:hover {
        background-color: #f8f8f8;
    }
    
    .sort-arrow {
        margin-left: 5px;
        opacity: 0.3;
    }
    
    .sortable.asc .sort-arrow::after {
        content: "↑";
        opacity: 1;
    }
    
    .sortable.desc .sort-arrow::after {
        content: "↓";
        opacity: 1;
    }
    
    /* Version distribution chart */
    .chart-placeholder {
        padding: 20px;
        background: #f8f8f8;
        border-radius: 4px;
        text-align: center;
    }
    
    .version-distribution {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 15px;
        margin-top: 20px;
    }
    
    .version-item {
        padding: 15px;
        background: white;
        border-radius: 4px;
        border: 1px solid #b1b4b6;
        text-align: left;
    }
    
    .version-label {
        display: block;
        font-weight: bold;
        margin-bottom: 5px;
    }
    
    .version-count {
        display: block;
        color: #505a5f;
        font-size: 14px;
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
document.head.appendChild(rdsStyle);