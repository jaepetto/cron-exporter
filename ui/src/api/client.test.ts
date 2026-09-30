import { afterEach, describe, expect, it, vi } from 'vitest';

import { getJobs } from './client';

afterEach(() => vi.unstubAllGlobals());

describe('dashboard API client', () => {
  it('requests a typed paginated job collection from the dashboard base', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      jobs: [], total_count: 0, page: 1, page_size: 25, total_pages: 0, has_next: false, has_previous: false,
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    vi.stubGlobal('fetch', fetchMock);

    const result = await getJobs({ q: 'backup', page: 1, page_size: 25 });

    expect(result.total_count).toBe(0);
    expect(fetchMock).toHaveBeenCalledOnce();
    const request = fetchMock.mock.calls[0][0] as Request;
    expect(request.url).toContain('/dashboard/api/jobs?');
    expect(request.url).toContain('q=backup');
    expect(request.credentials).toBe('same-origin');
  });

  it('normalizes structured dashboard errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      code: 'validation', message: 'job validation failed', fields: { status: 'invalid' }, timestamp: new Date().toISOString(),
    }), { status: 400, headers: { 'Content-Type': 'application/json' } })));

    await expect(getJobs({ status: 'active' })).rejects.toMatchObject({
      name: 'APIError',
      code: 'validation',
      fields: { status: 'invalid' },
    });
  });
});