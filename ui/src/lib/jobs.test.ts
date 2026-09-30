import { describe, expect, it } from 'vitest';

import type { Job } from '../api/client';
import { formatRelativeTime, formatThreshold, isJobOverdue } from './jobs';

const job: Job = {
  id: 1,
  job_name: 'backup',
  host: 'db01',
  automatic_failure_threshold: 3600,
  labels: {},
  status: 'active',
  last_reported_at: '2026-01-01T00:00:00Z',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('job formatting', () => {
  it('marks only active silent jobs overdue', () => {
    const now = Date.parse('2026-01-01T02:00:00Z');
    expect(isJobOverdue(job, now)).toBe(true);
    expect(isJobOverdue({ ...job, status: 'maintenance' }, now)).toBe(false);
    expect(isJobOverdue({ ...job, last_reported_at: '2026-01-01T01:30:00Z' }, now)).toBe(false);
  });

  it('formats relative time at useful boundaries', () => {
    const now = Date.parse('2026-01-03T00:00:00Z');
    expect(formatRelativeTime('2026-01-02T23:59:30Z', now)).toBe('just now');
    expect(formatRelativeTime('2026-01-02T23:30:00Z', now)).toBe('30m ago');
    expect(formatRelativeTime('2026-01-02T20:00:00Z', now)).toBe('4h ago');
    expect(formatRelativeTime('2026-01-01T00:00:00Z', now)).toBe('2d ago');
  });

  it('formats thresholds using the largest exact unit', () => {
    expect(formatThreshold(172800)).toBe('2d');
    expect(formatThreshold(7200)).toBe('2h');
    expect(formatThreshold(300)).toBe('5m');
    expect(formatThreshold(45)).toBe('45s');
  });
});