import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { JobSearchResult } from '../api/client';
import { JobsPage } from './JobsPage';

const useJobs = vi.fn();
const useRuntimeConfig = vi.fn();
vi.mock('../hooks/useJobs', () => ({
  useJobs: (...args: unknown[]) => useJobs(...args),
  useRuntimeConfig: (...args: unknown[]) => useRuntimeConfig(...args),
}));

const result: JobSearchResult = {
  jobs: [{
    id: 7,
    job_name: 'nightly-backup',
    host: 'db01',
    automatic_failure_threshold: 3600,
    labels: { env: 'prod' },
    status: 'active',
    last_reported_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }],
  total_count: 1,
  page: 1,
  page_size: 25,
  total_pages: 1,
  has_next: false,
  has_previous: false,
};

describe('JobsPage', () => {
  beforeEach(() => {
    useJobs.mockReset();
    useRuntimeConfig.mockReturnValue({ data: { page_size: 25 } });
  });

  it('renders job data in desktop and mobile representations', () => {
    useJobs.mockReturnValue({ data: result, isLoading: false, error: undefined, mutate: vi.fn() });
    render(<MemoryRouter><JobsPage /></MemoryRouter>);

    expect(screen.getByRole('heading', { name: 'Jobs' })).toBeInTheDocument();
    expect(screen.getAllByText('nightly-backup')).toHaveLength(2);
    expect(screen.getByText('Matching jobs').previousSibling).toHaveTextContent('1');
    expect(screen.getAllByText('env:prod')).not.toHaveLength(0);
  });

  it('renders a useful empty state', () => {
    useJobs.mockReturnValue({ data: { ...result, jobs: [], total_count: 0 }, isLoading: false, error: undefined, mutate: vi.fn() });
    render(<MemoryRouter><JobsPage /></MemoryRouter>);

    expect(screen.getByRole('heading', { name: 'No jobs found' })).toBeInTheDocument();
  });

  it('uses the configured page size', () => {
    useRuntimeConfig.mockReturnValue({ data: { page_size: 40 } });
    useJobs.mockReturnValue({ data: result, isLoading: false, error: undefined, mutate: vi.fn() });
    render(<MemoryRouter><JobsPage /></MemoryRouter>);

    expect(useJobs).toHaveBeenCalledWith(expect.objectContaining({ page_size: 40 }));
  });
});
