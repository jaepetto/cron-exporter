import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { JobFormPage } from './JobFormPage';

const createJob = vi.fn();
vi.mock('../api/client', () => ({
  APIError: class APIError extends Error { fields = {}; },
  createJob: (...args: unknown[]) => createJob(...args),
  updateJob: vi.fn(),
}));
vi.mock('../hooks/useJobs', () => ({
  useJob: () => ({ data: undefined, isLoading: false, error: undefined }),
}));

describe('JobFormPage', () => {
  beforeEach(() => createJob.mockReset());

  it('rejects labels that are not a string-valued object', async () => {
    const user = userEvent.setup();
    render(<MemoryRouter><JobFormPage mode="create" /></MemoryRouter>);
    fireEvent.change(screen.getByLabelText('Labels (JSON)'), { target: { value: '[]' } });

    await user.click(screen.getByRole('button', { name: 'Save job' }));

    expect(screen.getByText('Enter a JSON object whose values are strings.')).toBeInTheDocument();
    expect(createJob).not.toHaveBeenCalled();
  });

  it('submits normalized job data and navigates to detail', async () => {
    const user = userEvent.setup();
    createJob.mockResolvedValue({ id: 12 });
    render(
      <MemoryRouter initialEntries={['/jobs/new']}>
        <Routes>
          <Route path="/jobs/new" element={<JobFormPage mode="create" />} />
          <Route path="/jobs/:id" element={<h1>Saved job</h1>} />
        </Routes>
      </MemoryRouter>,
    );

    await user.type(screen.getByLabelText('Job name'), ' nightly-backup ');
    await user.type(screen.getByLabelText('Host'), ' db01 ');
    fireEvent.change(screen.getByLabelText('Labels (JSON)'), { target: { value: '{"env":"prod"}' } });
    await user.click(screen.getByRole('button', { name: 'Save job' }));

    expect(createJob).toHaveBeenCalledWith(expect.objectContaining({ job_name: 'nightly-backup', host: 'db01', labels: { env: 'prod' } }));
    expect(await screen.findByRole('heading', { name: 'Saved job' })).toBeInTheDocument();
  });
});