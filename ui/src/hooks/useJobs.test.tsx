import { renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { DashboardConnectionContext } from './dashboardConnection';
import { useJobs } from './useJobs';

const mocks = vi.hoisted(() => ({ useSWR: vi.fn() }));

vi.mock('swr', () => ({ default: (...args: unknown[]) => mocks.useSWR(...args) }));
vi.mock('../api/client', () => ({
  getJob: vi.fn(),
  getJobs: vi.fn(),
  getRuntimeConfig: vi.fn(),
}));

const runtimeConfig = {
  sse_enabled: true,
  polling_fallback: true,
  polling_interval: 7,
  page_size: 25,
};

function queryRefreshInterval() {
  const queryCalls = mocks.useSWR.mock.calls.filter((call) => Array.isArray(call[0]) && call[0][0] === 'jobs');
  const queryCall = queryCalls[queryCalls.length - 1];
  return (queryCall?.[2] as { refreshInterval: number } | undefined)?.refreshInterval;
}

describe('useJobs polling fallback', () => {
  it('polls only while SSE is disconnected', () => {
    mocks.useSWR.mockImplementation((key: unknown) => key === 'dashboard-config' ? { data: runtimeConfig } : {});
    let connected = false;
    const wrapper = ({ children }: PropsWithChildren) => (
      <DashboardConnectionContext.Provider value={connected}>{children}</DashboardConnectionContext.Provider>
    );
    const hook = renderHook(() => useJobs({ page: 1 }), { wrapper });

    expect(queryRefreshInterval()).toBe(7000);

    connected = true;
    hook.rerender();
    expect(queryRefreshInterval()).toBe(0);
  });

  it('polls when SSE is disabled and fallback is enabled', () => {
    mocks.useSWR.mockImplementation((key: unknown) => key === 'dashboard-config'
      ? { data: { ...runtimeConfig, sse_enabled: false } }
      : {});
    const wrapper = ({ children }: PropsWithChildren) => (
      <DashboardConnectionContext.Provider value={true}>{children}</DashboardConnectionContext.Provider>
    );

    renderHook(() => useJobs({ page: 1 }), { wrapper });

    expect(queryRefreshInterval()).toBe(7000);
  });
});
