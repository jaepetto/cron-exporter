import createClient from 'openapi-fetch';

import { embeddedConfig } from '../lib/runtime';
import type { components, paths } from './schema';

export type Job = components['schemas']['Job'];
export type JobSearchResult = components['schemas']['JobSearchResult'];
export type CreateJobInput = components['schemas']['CreateJobRequest'];
export type UpdateJobInput = components['schemas']['UpdateJobRequest'];
export type DashboardRuntimeConfig = components['schemas']['DashboardRuntimeConfig'];
export type DashboardEvent = components['schemas']['DashboardEvent'];
export type DashboardError = components['schemas']['DashboardError'];

export interface JobsQuery {
  q?: string;
  name?: string;
  host?: string;
  status?: 'active' | 'maintenance' | 'paused';
  page?: number;
  page_size?: number;
  before?: string;
  after?: string;
}

export class APIError extends Error {
  readonly code: string;
  readonly fields: Record<string, string>;

  constructor(error: DashboardError | undefined, fallback: string) {
    super(error?.message ?? fallback);
    this.name = 'APIError';
    this.code = error?.code ?? 'request_failed';
    this.fields = error?.fields ?? {};
  }
}

export const apiClient = createClient<paths>({
  baseUrl: new URL(embeddedConfig.basePath, window.location.origin).toString().replace(/\/$/, ''),
  credentials: 'same-origin',
  headers: { Accept: 'application/json' },
  fetch: (request) => globalThis.fetch(request),
});

export async function getRuntimeConfig(): Promise<DashboardRuntimeConfig> {
  const { data, error, response } = await apiClient.GET('/api/config');
  if (!data) throw new APIError(error as DashboardError | undefined, `Configuration request failed (${response.status})`);
  return data;
}

export async function getJobs(query: JobsQuery): Promise<JobSearchResult> {
  const { data, error, response } = await apiClient.GET('/api/jobs', { params: { query } });
  if (!data) throw new APIError(error as DashboardError | undefined, `Jobs request failed (${response.status})`);
  return data;
}

export async function getJob(id: number): Promise<Job> {
  const { data, error, response } = await apiClient.GET('/api/jobs/{id}', { params: { path: { id } } });
  if (!data) throw new APIError(error as DashboardError | undefined, `Job request failed (${response.status})`);
  return data;
}

export async function createJob(input: CreateJobInput): Promise<Job> {
  const { data, error, response } = await apiClient.POST('/api/jobs', { body: input });
  if (!data) throw new APIError(error as DashboardError | undefined, `Create request failed (${response.status})`);
  return data;
}

export async function updateJob(id: number, input: UpdateJobInput): Promise<Job> {
  const { data, error, response } = await apiClient.PUT('/api/jobs/{id}', {
    params: { path: { id } },
    body: input,
  });
  if (!data) throw new APIError(error as DashboardError | undefined, `Update request failed (${response.status})`);
  return data;
}

export async function toggleJob(id: number): Promise<Job> {
  const { data, error, response } = await apiClient.POST('/api/jobs/{id}/toggle', { params: { path: { id } } });
  if (!data) throw new APIError(error as DashboardError | undefined, `Toggle request failed (${response.status})`);
  return data;
}

export async function deleteJob(id: number): Promise<void> {
  const { error, response } = await apiClient.DELETE('/api/jobs/{id}', { params: { path: { id } } });
  if (!response.ok) throw new APIError(error as DashboardError | undefined, `Delete request failed (${response.status})`);
}
