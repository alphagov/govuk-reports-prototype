document.addEventListener('DOMContentLoaded', function() {
    loadCostData();
});

async function loadCostData() {
    try {
        const response = await fetch('/api/v1/costs');
        const result = await response.json();
        
        if (response.ok) {
            displayCostSummary(result.data);
            displayServiceBreakdown(result.data.services);
        } else {
            displayError('Failed to load cost data: ' + result.message);
        }
    } catch (error) {
        console.error('Error loading cost data:', error);
        displayError('Failed to load cost data. Please try again later.');
    }
}

function displayCostSummary(data) {
    const costSummaryElement = document.getElementById('cost-summary');
    
    const formatCurrency = (amount, currency) => {
        return new Intl.NumberFormat('en-GB', {
            style: 'currency',
            currency: currency || 'GBP'
        }).format(amount);
    };
    
    const formatDate = (dateString) => {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-GB', {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    };
    
    costSummaryElement.innerHTML = `
        <div class="cost-amount">${formatCurrency(data.total_cost, data.currency)}</div>
        <div class="cost-period">
            Period: ${formatDate(data.period_start)} - ${formatDate(data.period_end)}
        </div>
        <p>Last updated: ${formatDate(data.last_updated)}</p>
    `;
}

function displayServiceBreakdown(services) {
    const serviceBreakdownElement = document.getElementById('service-breakdown');
    
    if (!services || services.length === 0) {
        serviceBreakdownElement.innerHTML = '<p>No service data available.</p>';
        return;
    }
    
    const formatCurrency = (amount, currency) => {
        return new Intl.NumberFormat('en-GB', {
            style: 'currency',
            currency: currency || 'GBP'
        }).format(amount);
    };
    
    const serviceItems = services
        .sort((a, b) => b.amount - a.amount)
        .map(service => `
            <div class="service-item">
                <span class="service-name">${service.service}</span>
                <span class="service-cost">${formatCurrency(service.amount, service.currency)}</span>
            </div>
        `);
    
    serviceBreakdownElement.innerHTML = serviceItems.join('');
}

function displayError(message) {
    const costSummaryElement = document.getElementById('cost-summary');
    const serviceBreakdownElement = document.getElementById('service-breakdown');
    
    const errorHtml = `<div class="error">${message}</div>`;
    
    costSummaryElement.innerHTML = errorHtml;
    serviceBreakdownElement.innerHTML = errorHtml;
}