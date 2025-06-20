// GOV.UK Reports Dashboard JavaScript
// Handles dashboard loading and multiple report modules

class ReportsDashboard {
    constructor() {
        this.reports = [];
        this.costData = null;
        this.rdsData = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.detectPageType();
    }

    detectPageType() {
        // Check if we're on the main dashboard or applications page
        if (window.location.pathname === '/' || window.location.pathname === '/dashboard') {
            this.loadDashboard();
        } else if (window.location.pathname === '/applications') {
            this.loadApplications();
        }
    }

    setupEventListeners() {
        // Retry button
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                this.detectPageType();
            });
        }

        // Search functionality for applications page
        const searchInput = document.getElementById('search-input');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.handleSearch(e.target.value);
            });
        }

        // Filter buttons for applications page
        const filterButtons = document.querySelectorAll('[data-filter]');
        filterButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleFilter(e.target.dataset.filter);
                this.setActiveFilter(e.target);
            });
        });

        // Sort functionality for applications page
        const sortableHeaders = document.querySelectorAll('.sortable');
        sortableHeaders.forEach(header => {
            header.addEventListener('click', (e) => {
                this.handleSort(e.currentTarget.dataset.sort);
            });
        });
    }

    async loadDashboard() {
        this.showLoading();
        this.hideError();

        try {
            // Load all report modules in parallel
            const [reportsResponse, costSummary, rdsSummary] = await Promise.allSettled([
                fetch('/api/reports/summary'),
                fetch('/api/reports/costs'),
                fetch('/api/reports/rds')
            ]);

            // Process reports list
            if (reportsResponse.status === 'fulfilled' && reportsResponse.value.ok) {
                const reportsData = await reportsResponse.value.json();
                this.reports = reportsData.summaries || [];
                this.updateSystemInfo(reportsData);
            }

            // Process cost module
            if (costSummary.status === 'fulfilled' && costSummary.value.ok) {
                this.costData = await costSummary.value.json();
                this.updateCostModule();
            } else {
                this.setCostModuleError();
            }

            // Process RDS module
            if (rdsSummary.status === 'fulfilled' && rdsSummary.value.ok) {
                this.rdsData = await rdsSummary.value.json();
                this.updateRDSModule();
            } else {
                this.setRDSModuleError();
            }

            this.showDashboard();
            this.hideLoading();

        } catch (error) {
            console.error('Failed to load dashboard:', error);
            this.showError(error.message);
            this.hideLoading();
        }
    }

    updateCostModule() {
        this.setModuleStatus('cost', 'healthy', 'Available');

        // Update cost metrics from report summary
        if (this.costData && this.costData.summary) {
            const costSummary = this.costData.summary.find(s => s.title === 'Total Monthly Cost');
            const appSummary = this.costData.summary.find(s => s.title === 'Applications');
            const avgSummary = this.costData.summary.find(s => s.title === 'Average Cost');

            if (costSummary) {
                document.getElementById('cost-total').textContent = costSummary.value;
            }
            if (appSummary) {
                document.getElementById('cost-apps').textContent = appSummary.value;
            }
            if (avgSummary) {
                document.getElementById('cost-average').textContent = avgSummary.value;
            }
        } else {
            // Fallback: try to get from applications API
            this.loadCostSummaryFallback();
        }
    }

    async loadCostSummaryFallback() {
        try {
            const response = await fetch('/api/applications');
            if (response.ok) {
                const data = await response.json();
                document.getElementById('cost-total').textContent = this.formatCurrency(data.total_cost, data.currency);
                document.getElementById('cost-apps').textContent = data.count.toString();
                const avgCost = data.count > 0 ? data.total_cost / data.count : 0;
                document.getElementById('cost-average').textContent = this.formatCurrency(avgCost, data.currency);
            }
        } catch (error) {
            console.error('Failed to load cost fallback data:', error);
        }
    }

    updateRDSModule() {
        this.setModuleStatus('rds', 'healthy', 'Available');

        // Update RDS metrics from report summary
        if (this.rdsData && this.rdsData.summary) {
            const instancesSummary = this.rdsData.summary.find(s => s.title === 'PostgreSQL Instances');
            const eolSummary = this.rdsData.summary.find(s => s.title === 'EOL Instances');
            const complianceSummary = this.rdsData.summary.find(s => s.title === 'Version Compliance');

            if (instancesSummary) {
                document.getElementById('rds-instances').textContent = instancesSummary.value;
            }
            if (eolSummary) {
                const eolEl = document.getElementById('rds-eol');
                eolEl.textContent = eolSummary.value;
                // Add alert styling if there are EOL instances
                if (parseInt(eolSummary.value) > 0) {
                    eolEl.classList.add('alert');
                } else {
                    eolEl.classList.remove('alert');
                }
            }
            if (complianceSummary) {
                document.getElementById('rds-compliance').textContent = complianceSummary.value;
            }
        } else {
            // Fallback: try to get from RDS summary API
            this.loadRDSSummaryFallback();
        }
    }

    async loadRDSSummaryFallback() {
        try {
            const response = await fetch('/api/rds/summary');
            if (response.ok) {
                const data = await response.json();
                document.getElementById('rds-instances').textContent = data.postgresql_count || '0';
                document.getElementById('rds-eol').textContent = data.eol_instances || '0';
                
                // Calculate compliance percentage
                const total = data.postgresql_count || 0;
                const eol = data.eol_instances || 0;
                const outdated = data.outdated_instances || 0;
                const compliant = total - eol - outdated;
                const compliancePercentage = total > 0 ? ((compliant / total) * 100).toFixed(1) : '0';
                document.getElementById('rds-compliance').textContent = `${compliancePercentage}%`;
            }
        } catch (error) {
            console.error('Failed to load RDS fallback data:', error);
        }
    }

    setCostModuleError() {
        this.setModuleStatus('cost', 'error', 'Unavailable');
        document.getElementById('cost-total').textContent = 'Error';
        document.getElementById('cost-apps').textContent = 'Error';
        document.getElementById('cost-average').textContent = 'Error';
    }

    setRDSModuleError() {
        this.setModuleStatus('rds', 'error', 'Unavailable');
        document.getElementById('rds-instances').textContent = 'Error';
        document.getElementById('rds-eol').textContent = 'Error';
        document.getElementById('rds-compliance').textContent = 'Error';
    }

    setModuleStatus(moduleType, status, text) {
        const statusEl = document.getElementById(`${moduleType}-status`);
        const indicatorEl = statusEl.querySelector('.status-indicator');
        
        // Remove existing status classes
        indicatorEl.classList.remove('loading', 'healthy', 'error');
        indicatorEl.classList.add(status);
        
        // Update text
        statusEl.childNodes[statusEl.childNodes.length - 1].textContent = text;
    }

    updateSystemInfo(data) {
        const countEl = document.getElementById('available-reports-count');
        const lastUpdatedEl = document.getElementById('last-updated');
        
        if (countEl) {
            countEl.textContent = data.count || '0';
        }
        if (lastUpdatedEl) {
            lastUpdatedEl.textContent = new Date().toLocaleString();
        }
    }

    showDashboard() {
        const reportsGrid = document.getElementById('reports-grid');
        const quickActions = document.getElementById('quick-actions');
        const systemInfo = document.getElementById('system-info');
        
        if (reportsGrid) reportsGrid.style.display = 'block';
        if (quickActions) quickActions.style.display = 'block';
        if (systemInfo) systemInfo.style.display = 'block';
    }

    // Applications page methods (keep existing functionality)
    async loadApplications() {
        this.showLoading();
        this.hideError();

        try {
            const response = await fetch('/api/applications');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            this.applications = data.applications || [];
            this.filteredApplications = [...this.applications];
            
            this.updateSummaryCards(data);
            this.renderApplications();
            this.showApplicationsTable();
            this.hideLoading();

        } catch (error) {
            console.error('Failed to load applications:', error);
            this.showError(error.message);
            this.hideLoading();
        }
    }

    updateSummaryCards(data) {
        // Update total cost
        const totalCostEl = document.getElementById('total-cost');
        if (totalCostEl) {
            totalCostEl.textContent = this.formatCurrency(data.total_cost, data.currency);
        }

        // Update application count
        const appCountEl = document.getElementById('app-count');
        if (appCountEl) {
            appCountEl.textContent = data.count.toString();
        }

        // Update average cost
        const avgCostEl = document.getElementById('avg-cost');
        if (avgCostEl && data.count > 0) {
            const avgCost = data.total_cost / data.count;
            avgCostEl.textContent = this.formatCurrency(avgCost, data.currency);
        }
    }

    renderApplications() {
        const tbody = document.getElementById('applications-tbody');
        
        if (!tbody) return;

        // Clear existing content
        tbody.innerHTML = '';

        if (this.filteredApplications.length === 0) {
            this.showNoResults();
            return;
        }

        this.hideNoResults();

        // Sort applications
        this.sortApplications();

        // Render each application
        this.filteredApplications.forEach(app => {
            const row = this.createApplicationRow(app);
            tbody.appendChild(row);
        });
    }

    createApplicationRow(app) {
        const row = document.createElement('tr');
        row.className = 'govuk-table__row';
        
        // Application name with link
        const nameCell = document.createElement('td');
        nameCell.className = 'govuk-table__cell';
        const nameLink = document.createElement('a');
        nameLink.href = `/applications/${encodeURIComponent(app.shortname)}`;
        nameLink.className = 'govuk-link';
        nameLink.textContent = app.name;
        nameCell.appendChild(nameLink);
        
        // Team
        const teamCell = document.createElement('td');
        teamCell.className = 'govuk-table__cell';
        teamCell.textContent = app.team;
        
        // Hosting platform
        const hostingCell = document.createElement('td');
        hostingCell.className = 'govuk-table__cell';
        const hostingTag = document.createElement('span');
        hostingTag.className = `hosting-tag hosting-tag--${app.production_hosted_on.toLowerCase()}`;
        hostingTag.textContent = app.production_hosted_on.toUpperCase();
        hostingCell.appendChild(hostingTag);
        
        // Cost
        const costCell = document.createElement('td');
        costCell.className = 'govuk-table__cell numeric';
        costCell.innerHTML = `<strong>${this.formatCurrency(app.total_cost, app.currency)}</strong>`;
        
        // Service count
        const servicesCell = document.createElement('td');
        servicesCell.className = 'govuk-table__cell';
        servicesCell.textContent = `${app.service_count} services`;
        
        // Actions
        const actionsCell = document.createElement('td');
        actionsCell.className = 'govuk-table__cell';
        
        const viewButton = document.createElement('a');
        viewButton.href = `/applications/${encodeURIComponent(app.shortname)}`;
        viewButton.className = 'govuk-button govuk-button--secondary';
        viewButton.style.fontSize = '14px';
        viewButton.style.padding = '5px 10px';
        viewButton.textContent = 'View Details';
        actionsCell.appendChild(viewButton);
        
        // Append all cells
        row.appendChild(nameCell);
        row.appendChild(teamCell);
        row.appendChild(hostingCell);
        row.appendChild(costCell);
        row.appendChild(servicesCell);
        row.appendChild(actionsCell);
        
        return row;
    }

    handleSearch(searchTerm) {
        const term = searchTerm.toLowerCase().trim();
        
        if (term === '') {
            this.filteredApplications = this.applyFilter(this.applications);
        } else {
            const filtered = this.applications.filter(app => 
                app.name.toLowerCase().includes(term) ||
                app.shortname.toLowerCase().includes(term) ||
                app.team.toLowerCase().includes(term) ||
                app.production_hosted_on.toLowerCase().includes(term)
            );
            this.filteredApplications = this.applyFilter(filtered);
        }
        
        this.renderApplications();
    }

    handleFilter(filter) {
        this.currentFilter = filter;
        this.filteredApplications = this.applyFilter(this.applications);
        
        // Re-apply search if there's a search term
        const searchInput = document.getElementById('search-input');
        if (searchInput && searchInput.value.trim()) {
            this.handleSearch(searchInput.value);
        } else {
            this.renderApplications();
        }
    }

    applyFilter(applications) {
        if (this.currentFilter === 'all') {
            return [...applications];
        }
        
        return applications.filter(app => 
            app.production_hosted_on.toLowerCase() === this.currentFilter.toLowerCase()
        );
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
        this.renderApplications();
    }

    sortApplications() {
        if (!this.currentSort) {
            this.currentSort = { field: 'name', direction: 'asc' };
        }

        this.filteredApplications.sort((a, b) => {
            let aValue, bValue;
            
            switch (this.currentSort.field) {
                case 'name':
                    aValue = a.name.toLowerCase();
                    bValue = b.name.toLowerCase();
                    break;
                case 'cost':
                    aValue = a.total_cost;
                    bValue = b.total_cost;
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

    formatCurrency(amount, currency = 'GBP') {
        return new Intl.NumberFormat('en-GB', {
            style: 'currency',
            currency: currency,
            minimumFractionDigits: 0,
            maximumFractionDigits: 0
        }).format(amount);
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
            errorMessage.textContent = message || 'Failed to load data. Please try again.';
        }
    }

    hideError() {
        const errorState = document.getElementById('error-state');
        if (errorState) {
            errorState.style.display = 'none';
        }
    }

    showApplicationsTable() {
        const container = document.getElementById('applications-container');
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

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new ReportsDashboard();
});

// Add CSS for dashboard modules
const style = document.createElement('style');
style.textContent = `
    /* Report Module Cards */
    .report-module-card {
        border: 2px solid #b1b4b6;
        border-radius: 4px;
        padding: 20px;
        margin-bottom: 20px;
        background: #ffffff;
    }
    
    .report-module-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 15px;
        border-bottom: 1px solid #f3f2f1;
        padding-bottom: 10px;
    }
    
    .report-module-header h2 {
        margin: 0;
    }
    
    .module-status {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        font-weight: bold;
    }
    
    .status-indicator {
        width: 12px;
        height: 12px;
        border-radius: 50%;
        display: inline-block;
    }
    
    .status-indicator.loading {
        background: #ffdd00;
        animation: pulse 1.5s infinite;
    }
    
    .status-indicator.healthy {
        background: #00703c;
    }
    
    .status-indicator.error {
        background: #d4351c;
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }
    
    .report-module-content p {
        color: #505a5f;
        margin-bottom: 20px;
    }
    
    .module-metrics {
        display: grid;
        grid-template-columns: 1fr;
        gap: 10px;
        margin-bottom: 20px;
        padding: 15px;
        background: #f8f8f8;
        border-radius: 4px;
    }
    
    .metric {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .metric-label {
        font-size: 14px;
        color: #505a5f;
    }
    
    .metric-value {
        font-size: 18px;
        font-weight: bold;
        color: #0b0c0c;
    }
    
    .metric-value.alert {
        color: #d4351c;
    }
    
    .module-actions {
        display: flex;
        gap: 15px;
        align-items: center;
    }
    
    .module-actions .govuk-button {
        margin: 0;
    }
    
    /* Action Cards */
    .action-cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 20px;
        margin-top: 20px;
    }
    
    .action-card {
        border: 1px solid #b1b4b6;
        border-radius: 4px;
        padding: 15px;
        background: #ffffff;
    }
    
    .action-card h3 {
        margin-top: 0;
        margin-bottom: 10px;
    }
    
    .action-card p {
        margin: 0;
        color: #505a5f;
    }
    
    /* Hosting tags */
    .hosting-tag {
        padding: 2px 8px;
        border-radius: 3px;
        font-size: 12px;
        font-weight: bold;
        text-transform: uppercase;
    }
    
    .hosting-tag--eks {
        background-color: #1d70b8;
        color: white;
    }
    
    .hosting-tag--heroku {
        background-color: #7a2e8d;
        color: white;
    }
    
    .hosting-tag--gcp {
        background-color: #db4437;
        color: white;
    }
    
    .hosting-tag--aws {
        background-color: #ff9900;
        color: white;
    }
    
    /* Loading styles */
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
    
    /* Error styles */
    .error-container {
        margin: 20px 0;
    }
`;
document.head.appendChild(style);