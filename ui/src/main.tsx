import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { SWRConfig } from 'swr';

import { App } from './App';
import { embeddedConfig } from './lib/runtime';
import './styles.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SWRConfig value={{ revalidateOnFocus: true, shouldRetryOnError: true, errorRetryCount: 3 }}>
      <BrowserRouter basename={embeddedConfig.basePath}><App /></BrowserRouter>
    </SWRConfig>
  </StrictMode>,
);