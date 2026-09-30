import { ArrowLeft, ArrowRight, Plus, Search, Server, TimerOff } from 'lucide-react';
import { useDeferredValue } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

import type { Job, JobsQuery } from '../api/client';
import { useJobs } from '../hooks/useJobs';
import { formatRelativeTime, formatThreshold, isJobOverdue } from '../lib/jobs';
import { Button, ErrorState, LoadingState, SelectField, StatusBadge } from '../components/ui';

const statusOptions = [
  { value: 'all', label: 'All states' },
  { value: 'active', label: 'Active' },
  { value: 'maintenance', label: 'Maintenance' },
  { value: 'paused', label: 'Paused' },
];

export function JobsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const search = searchParams.get('q') ?? '';
  const deferredSearch = useDeferredValue(search);
  const statusValue = searchParams.get('status') ?? 'all';
  const page = Math.max(1, Number(searchParams.get('page') ?? '1') || 1);
  const query: JobsQuery = {
    q: deferredSearch || undefined,
    status: statusValue === 'all' ? undefined : statusValue as JobsQuery['status'],
    page,
    page_size: 25,
  };
  const jobs = useJobs(query);

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams);
    if (!value || value === 'all') next.delete(key);
    else next.set(key, value);
    next.delete('page');
    setSearchParams(next, { replace: true });
  };
  const setPage = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    next.set('page', String(nextPage));
    setSearchParams(next);
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <div><p className="eyebrow">Fleet status</p><h1>Jobs</h1></div>
        <Link className="button button-primary" to="/jobs/new"><Plus size={17} />New job</Link>
      </header>

      <section className="summary-strip" aria-label="Job summary">
        <SummaryMetric value={jobs.data?.total_count ?? 0} label="Matching jobs" />
        <SummaryMetric value={countStatus(jobs.data?.jobs, 'active')} label="Active on page" accent="healthy" />
        <SummaryMetric value={countOverdue(jobs.data?.jobs)} label="Overdue on page" accent="danger" />
        <SummaryMetric value={countStatus(jobs.data?.jobs, 'maintenance') + countStatus(jobs.data?.jobs, 'paused')} label="Suppressed on page" accent="warning" />
      </section>

      <div className="toolbar" role="search">
        <label className="search-field">
          <Search size={17} />
          <span className="sr-only">Search jobs</span>
          <input value={search} onChange={(event) => setFilter('q', event.target.value)} placeholder="Search name, host, or label" />
        </label>
        <SelectField id="status-filter" value={statusValue} onValueChange={(value) => setFilter('status', value)} options={statusOptions} placeholder="Status" />
      </div>

      {jobs.isLoading && <LoadingState label="Loading jobs" />}
      {jobs.error && <ErrorState message={jobs.error instanceof Error ? jobs.error.message : 'Unable to load jobs'} retry={() => void jobs.mutate()} />}
      {jobs.data && jobs.data.jobs.length === 0 && (
        <div className="empty-state"><TimerOff size={28} /><h2>No jobs found</h2><p>Adjust the filters or register a new job.</p></div>
      )}
      {jobs.data && jobs.data.jobs.length > 0 && <JobCollection jobs={jobs.data.jobs} />}

      {jobs.data && jobs.data.total_pages > 1 && (
        <nav className="pagination" aria-label="Job pages">
          <Button variant="secondary" disabled={!jobs.data.has_previous} onClick={() => setPage(page - 1)}><ArrowLeft size={16} />Previous</Button>
          <span>Page {jobs.data.page} of {jobs.data.total_pages}</span>
          <Button variant="secondary" disabled={!jobs.data.has_next} onClick={() => setPage(page + 1)}>Next<ArrowRight size={16} /></Button>
        </nav>
      )}
    </div>
  );
}

function SummaryMetric({ value, label, accent = 'neutral' }: { value: number; label: string; accent?: string }) {
  return <div className={`summary-metric metric-${accent}`}><strong>{value}</strong><span>{label}</span></div>;
}

function JobCollection({ jobs }: { jobs: Job[] }) {
  return (
    <div className="job-collection">
      <div className="job-table-wrap">
        <table className="job-table">
          <thead><tr><th>Job</th><th>Host</th><th>Status</th><th>Last report</th><th>Threshold</th><th><span className="sr-only">Open</span></th></tr></thead>
          <tbody>{jobs.map((job) => <JobRow key={job.id} job={job} />)}</tbody>
        </table>
      </div>
      <div className="job-mobile-list">{jobs.map((job) => <JobMobileRow key={job.id} job={job} />)}</div>
    </div>
  );
}

function JobRow({ job }: { job: Job }) {
  const overdue = isJobOverdue(job);
  return (
    <tr>
      <td><Link className="job-name" to={`/jobs/${job.id}`}>{job.job_name}</Link><LabelList labels={job.labels} /></td>
      <td><span className="host"><Server size={15} />{job.host}</span></td>
      <td><StatusBadge status={job.status} overdue={overdue} /></td>
      <td className="mono subtle">{formatRelativeTime(job.last_reported_at)}</td>
      <td className="mono subtle">{formatThreshold(job.automatic_failure_threshold)}</td>
      <td><Link className="row-link" to={`/jobs/${job.id}`} aria-label={`Open ${job.job_name}`}><ArrowRight size={17} /></Link></td>
    </tr>
  );
}

function JobMobileRow({ job }: { job: Job }) {
  return (
    <Link className="job-mobile-row" to={`/jobs/${job.id}`}>
      <div><strong>{job.job_name}</strong><span><Server size={14} />{job.host}</span></div>
      <StatusBadge status={job.status} overdue={isJobOverdue(job)} />
      <span className="mono subtle">{formatRelativeTime(job.last_reported_at)}</span>
      <ArrowRight size={17} />
    </Link>
  );
}

function LabelList({ labels }: { labels: Record<string, string> }) {
  const entries = Object.entries(labels).slice(0, 3);
  if (entries.length === 0) return null;
  return <div className="label-list">{entries.map(([key, value]) => <span key={key}>{key}:{value}</span>)}</div>;
}

function countStatus(jobs: Job[] | undefined, status: string) {
  return jobs?.filter((job) => job.status === status).length ?? 0;
}

function countOverdue(jobs: Job[] | undefined) {
  return jobs?.filter((job) => isJobOverdue(job)).length ?? 0;
}