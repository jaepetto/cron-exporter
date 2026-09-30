import { useEffect, useState } from 'react';

export type Theme = 'light' | 'dark' | 'system';

const themeStorageKey = 'cronmetrics-theme';

function readTheme(): Theme {
  const stored = localStorage.getItem(themeStorageKey);
  return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
}

function applyTheme(theme: Theme) {
  const dark = theme === 'dark' || (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', dark);
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readTheme);

  useEffect(() => {
    applyTheme(theme);
    localStorage.setItem(themeStorageKey, theme);
    const media = matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => theme === 'system' && applyTheme(theme);
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, [theme]);

  return { theme, setTheme };
}
