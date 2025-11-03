// Dashboard JavaScript functionality

// Auto-refresh functionality
let refreshInterval;

function startAutoRefresh(intervalSeconds) {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }

    refreshInterval = setInterval(() => {
        refreshJobList();
    }, intervalSeconds * 1000);
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

// Refresh job list via HTMX-style fetch
function refreshJobList() {
    fetch('/dashboard/api/jobs')
        .then(response => response.json())
        .then(data => {
            updateJobTable(data.jobs);
        })
        .catch(error => {
            console.error('Error refreshing job list:', error);
        });
}

// Update job table with new data
function updateJobTable(jobs) {
    const tbody = document.querySelector('#jobs-table tbody');
    if (!tbody) return;

    const currentJobs = new Map();
    jobs.forEach(job => {
        currentJobs.set(job.id, job);
    });

    // Update existing rows or add new ones
    const rows = tbody.querySelectorAll('tr[data-job-id]');
    rows.forEach(row => {
        const jobId = parseInt(row.getAttribute('data-job-id'));
        const job = currentJobs.get(jobId);

        if (job) {
            updateJobRow(row, job);
            currentJobs.delete(jobId);
        } else {
            row.remove();
        }
    });

    // Add new jobs
    currentJobs.forEach(job => {
        const newRow = createJobRow(job);
        tbody.appendChild(newRow);
    });
}

// Update a job row with new data
function updateJobRow(row, job) {
    const statusCell = row.querySelector('.job-status');
    const lastReportedCell = row.querySelector('.job-last-reported');

    if (statusCell) {
        statusCell.innerHTML = getStatusBadge(job.status);
    }

    if (lastReportedCell) {
        lastReportedCell.textContent = formatTimeAgo(job.last_reported_at);
    }
}

// Create a new job row
function createJobRow(job) {
    const row = document.createElement('tr');
    row.setAttribute('data-job-id', job.id);

    row.innerHTML = `
        <td><strong>${escapeHtml(job.name)}</strong></td>
        <td>${escapeHtml(job.host)}</td>
        <td class="job-status">${getStatusBadge(job.status)}</td>
        <td class="job-last-reported">${formatTimeAgo(job.last_reported_at)}</td>
        <td>
            <a href="/dashboard/jobs/${job.id}" class="btn btn-sm btn-primary">View</a>
            <a href="/dashboard/jobs/${job.id}/edit" class="btn btn-sm btn-secondary">Edit</a>
        </td>
    `;

    return row;
}

// Get status badge HTML
function getStatusBadge(status) {
    const badgeClass = status === 'active' ? 'success' :
                      status === 'maintenance' ? 'warning' :
                      status === 'paused' ? 'secondary' : 'danger';
    return `<span class="badge badge-${badgeClass}">${status}</span>`;
}

// Format time ago
function formatTimeAgo(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? 's' : ''} ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Form validation
function validateJobForm() {
    const form = document.getElementById('job-form');
    if (!form) return true;

    const name = form.querySelector('[name="name"]').value.trim();
    const host = form.querySelector('[name="host"]').value.trim();
    const threshold = form.querySelector('[name="automatic_failure_threshold"]').value;

    if (!name) {
        showError('Job name is required');
        return false;
    }

    if (!host) {
        showError('Host is required');
        return false;
    }

    if (!threshold || threshold < 1) {
        showError('Automatic failure threshold must be at least 1 second');
        return false;
    }

    return true;
}

// Show error message
function showError(message) {
    // Simple alert for now, can be enhanced with toast notifications
    alert(message);
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Get refresh interval from page config
    const refreshIntervalEl = document.getElementById('refresh-interval');
    const refreshInterval = refreshIntervalEl ? parseInt(refreshIntervalEl.value) : 5;

    // Start auto-refresh for job list page
    if (document.getElementById('jobs-table')) {
        startAutoRefresh(refreshInterval);
    }

    // Form validation
    const jobForm = document.getElementById('job-form');
    if (jobForm) {
        jobForm.addEventListener('submit', function(e) {
            if (!validateJobForm()) {
                e.preventDefault();
            }
        });
    }
});

// Clean up on page unload
window.addEventListener('beforeunload', function() {
    stopAutoRefresh();
});
