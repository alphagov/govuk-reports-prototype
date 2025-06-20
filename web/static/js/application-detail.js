// GOV.UK Application Detail JavaScript
// Handles loading and displaying application details and service costs

class ApplicationDetail {
    constructor() {
        this.applicationName = window.APPLICATION_NAME || this.getApplicationNameFromURL();
        this.applicationData = null;
        this.servicesData = [];
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadApplicationDetail();
    }

    setupEventListeners() {
        // Retry button
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                this.loadApplicationDetail();
            });
        }
    }

    getApplicationNameFromURL() {
        const pathParts = window.location.pathname.split('/');
        return pathParts[pathParts.length - 1] || '';
    }

    async loadApplicationDetail() {
        if (!this.applicationName) {
            this.showError('No application name provided');
            return;
        }

        this.showLoading();
        this.hideError();

        try {
            // Load application details and services in parallel
            const [appResponse, servicesResponse] = await Promise.all([
                fetch(`/api/applications/${encodeURIComponent(this.applicationName)}`),
                fetch(`/api/applications/${encodeURIComponent(this.applicationName)}/services`)
            ]);

            if (!appResponse.ok) {
                if (appResponse.status === 404) {
                    throw new Error('Application not found');
                } else {
                    throw new Error(`HTTP ${appResponse.status}: ${appResponse.statusText}`);
                }
            }

            if (!servicesResponse.ok) {
                throw new Error(`Failed to load services: HTTP ${servicesResponse.status}`);
            }

            this.applicationData = await appResponse.json();
            const servicesData = await servicesResponse.json();
            this.servicesData = servicesData.services || [];

            this.renderApplicationDetail();
            this.renderServicesTable();
            this.renderCostChart();
            this.showApplicationDetail();
            this.hideLoading();

        } catch (error) {
            console.error('Failed to load application detail:', error);
            this.showError(error.message);
            this.hideLoading();
        }
    }

    renderApplicationDetail() {
        if (!this.applicationData) return;

        const app = this.applicationData;

        // Update page title
        document.title = `${app.name} - GOV.UK Reports Dashboard`;
        
        // Update breadcrumb
        const breadcrumbAppName = document.getElementById('breadcrumb-app-name');
        if (breadcrumbAppName) {
            breadcrumbAppName.textContent = app.name;
        }

        // Update application header
        const appTitle = document.getElementById('app-title');
        if (appTitle) {
            appTitle.textContent = app.name;
        }

        // Update info cards
        this.updateElement('total-cost', this.formatCurrency(app.total_cost, app.currency));
        this.updateElement('team', app.team);
        this.updateElement('hosting', app.production_hosted_on.toUpperCase());
        this.updateElement('service-count', `${app.service_count} services`);

        // Update links
        this.updateLinks(app.links);
    }

    updateLinks(links) {
        if (!links) return;

        // Repository link
        const repoLink = document.getElementById('repo-link');
        if (repoLink && links.repo_url) {
            repoLink.href = links.repo_url;
            repoLink.style.display = 'inline';
        }

        // Sentry link
        const sentryLink = document.getElementById('sentry-link');
        if (sentryLink && links.sentry_url) {
            sentryLink.href = links.sentry_url;
            sentryLink.style.display = 'inline';
        }
    }

    renderServicesTable() {
        const tbody = document.getElementById('services-tbody');
        if (!tbody || !this.servicesData.length) return;

        // Clear existing content
        tbody.innerHTML = '';

        // Sort services by cost (highest first)
        const sortedServices = [...this.servicesData].sort((a, b) => b.cost - a.cost);

        // Render each service
        sortedServices.forEach(service => {
            const row = this.createServiceRow(service);
            tbody.appendChild(row);
        });
    }

    createServiceRow(service) {
        const row = document.createElement('tr');
        row.className = 'govuk-table__row';

        // Service name
        const nameCell = document.createElement('td');
        nameCell.className = 'govuk-table__cell';
        nameCell.innerHTML = `<strong>${service.service_name}</strong>`;

        // Cost
        const costCell = document.createElement('td');
        costCell.className = 'govuk-table__cell numeric';
        costCell.innerHTML = `<strong>${this.formatCurrency(service.cost, service.currency)}</strong>`;

        // Percentage
        const percentageCell = document.createElement('td');
        percentageCell.className = 'govuk-table__cell numeric';
        percentageCell.textContent = `${service.percentage.toFixed(1)}%`;

        // Period
        const periodCell = document.createElement('td');
        periodCell.className = 'govuk-table__cell';
        const startDate = new Date(service.start_date).toLocaleDateString('en-GB');
        const endDate = new Date(service.end_date).toLocaleDateString('en-GB');
        periodCell.textContent = `${startDate} - ${endDate}`;

        // Append all cells
        row.appendChild(nameCell);
        row.appendChild(costCell);
        row.appendChild(percentageCell);
        row.appendChild(periodCell);

        return row;
    }

    renderCostChart() {
        const chartContainer = document.getElementById('cost-chart');
        const legendContainer = document.getElementById('chart-legend');
        
        if (!chartContainer || !this.servicesData.length) return;

        // Clear existing content
        chartContainer.innerHTML = '';
        legendContainer.innerHTML = '';

        // Sort services by cost for chart
        const sortedServices = [...this.servicesData]
            .sort((a, b) => b.cost - a.cost)
            .slice(0, 8); // Show top 8 services

        const maxCost = Math.max(...sortedServices.map(s => s.cost));
        const colors = [
            '#1d70b8', '#005ea5', '#003078', '#4c2c92',
            '#7a2e8d', '#b10e1e', '#d4351c', '#f47738'
        ];

        // Create chart bars
        sortedServices.forEach((service, index) => {
            const bar = document.createElement('div');
            bar.className = 'chart-bar';
            
            const height = (service.cost / maxCost) * 100;
            bar.style.height = `${height}%`;
            bar.style.backgroundColor = colors[index % colors.length];
            
            // Add label
            const label = document.createElement('div');
            label.className = 'chart-bar-label';
            label.textContent = this.truncateText(service.service_name.replace('Amazon ', ''), 10);
            bar.appendChild(label);
            
            // Add tooltip on hover
            bar.title = `${service.service_name}: ${this.formatCurrency(service.cost)} (${service.percentage.toFixed(1)}%)`;
            
            chartContainer.appendChild(bar);

            // Add legend item
            const legendItem = document.createElement('div');
            legendItem.className = 'legend-item';
            
            const colorBox = document.createElement('div');
            colorBox.className = 'legend-color';
            colorBox.style.backgroundColor = colors[index % colors.length];
            
            const legendText = document.createElement('span');
            legendText.textContent = `${service.service_name} (${service.percentage.toFixed(1)}%)`;
            
            legendItem.appendChild(colorBox);
            legendItem.appendChild(legendText);
            legendContainer.appendChild(legendItem);
        });
    }

    updateElement(id, content) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = content;
        }
    }

    formatCurrency(amount, currency = 'GBP') {
        return new Intl.NumberFormat('en-GB', {
            style: 'currency',
            currency: currency,
            minimumFractionDigits: 0,
            maximumFractionDigits: 2
        }).format(amount);
    }

    truncateText(text, maxLength) {
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength - 3) + '...';
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
            errorMessage.textContent = message || 'Failed to load application details. Please try again.';
        }
    }

    hideError() {
        const errorState = document.getElementById('error-state');
        if (errorState) {
            errorState.style.display = 'none';
        }
    }

    showApplicationDetail() {
        const appDetails = document.getElementById('app-details');
        if (appDetails) {
            appDetails.style.display = 'block';
        }
    }
}

// Initialize application detail when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new ApplicationDetail();
});