// GOV.UK Reports Dashboard JavaScript
// Handles search, filtering, sorting, and data loading

class CostDashboard {
    constructor() {
        this.applications = [];
        this.filteredApplications = [];
        this.currentSort = { field: 'name', direction: 'asc' };
        this.currentFilter = 'all';
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadApplications();
    }

    setupEventListeners() {
        // Search functionality
        const searchInput = document.getElementById('search-input');
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

        // Retry button
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                this.loadApplications();
            });
        }
    }

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
        const noResults = document.getElementById('no-results');
        
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
            errorMessage.textContent = message || 'Failed to load applications. Please try again.';
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

// Utility functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new CostDashboard();
});

// Add some CSS for hosting tags dynamically
const style = document.createElement('style');
style.textContent = `
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
`;
document.head.appendChild(style);