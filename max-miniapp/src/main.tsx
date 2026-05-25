import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './pages/App';
import { AdminApp } from './pages/AdminApp';
import 'leaflet/dist/leaflet.css';
import './styles.css';

const isAdminView = typeof window !== 'undefined' && /\/admin\/?$/.test(window.location.pathname);

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    {isAdminView ? <AdminApp /> : <App />}
  </React.StrictMode>
);
