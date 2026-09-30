import { Activity, ListChecks, Moon, Plus, Sun, SunMoon } from 'lucide-react';
import { NavLink, Outlet } from 'react-router-dom';

import { DashboardConnectionContext } from '../hooks/dashboardConnection';
import { useDashboardEvents } from '../hooks/useDashboardEvents';
import { type Theme, useTheme } from '../hooks/useTheme';
import { embeddedConfig } from '../lib/runtime';
import { SelectField } from './ui';

const themeOptions: Array<{ value: Theme; label: string }> = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
];

export function Layout() {
  const events = useDashboardEvents();
  const { theme, setTheme } = useTheme();
  const ThemeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : SunMoon;
  const connected = events.enabled && events.connected;

  return (
    <DashboardConnectionContext.Provider value={connected}>
      <div className="app-shell">
        <aside className="sidebar">
          <div className="brand-block">
            <span className="brand-mark"><Activity size={19} /></span>
            <div><strong>{embeddedConfig.title}</strong><span>Operations</span></div>
          </div>
          <nav className="primary-nav" aria-label="Primary navigation">
            <NavLink to="/jobs"><ListChecks size={18} />Jobs</NavLink>
            <NavLink to="/jobs/new"><Plus size={18} />New job</NavLink>
          </nav>
          <div className="sidebar-footer">
            <div className="connection-state"><span className={connected ? 'connection-dot online' : 'connection-dot'} />{events.enabled ? (connected ? 'Live' : 'Reconnecting') : 'Polling'}</div>
            <div className="theme-control">
              <ThemeIcon size={17} />
              <SelectField id="theme" value={theme} onValueChange={(value) => setTheme(value as Theme)} options={themeOptions} placeholder="Theme" />
            </div>
          </div>
        </aside>
        <main className="main-content"><Outlet /></main>
      </div>
    </DashboardConnectionContext.Provider>
  );
}
