import { useContext } from 'react';
import useSWR from 'swr';

import { getJob, getJobs, getRuntimeConfig, type DashboardRuntimeConfig, type JobsQuery } from '../api/client';
import { DashboardConnectionContext } from './dashboardConnection';

export function useRuntimeConfig() {
  return useSWR('dashboard-config', getRuntimeConfig, { revalidateOnFocus: false });
}

export function useJobs(query: JobsQuery) {
  const runtime = useRuntimeConfig();
  const connected = useContext(DashboardConnectionContext);
  const refreshInterval = getRefreshInterval(runtime.data, connected);
  return useSWR(['jobs', query], () => getJobs(query), { keepPreviousData: true, refreshInterval });
}

export function useJob(id: number | null) {
  const runtime = useRuntimeConfig();
  const connected = useContext(DashboardConnectionContext);
  const refreshInterval = getRefreshInterval(runtime.data, connected);
  return useSWR(id ? ['job', id] : null, () => getJob(id as number), { refreshInterval });
}

function getRefreshInterval(runtime: DashboardRuntimeConfig | undefined, connected: boolean) {
  if (!runtime?.polling_fallback || (runtime.sse_enabled && connected)) return 0;
  return runtime.polling_interval * 1000;
}
