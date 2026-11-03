import { Route, Routes } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
import { DashboardPage } from '@/features/dashboard/DashboardPage';
import { VillagesPage } from '@/features/villages/VillagesPage';
import { SessionsPage } from '@/features/sessions/SessionsPage';
import { MonitoringPage } from '@/features/monitoring/MonitoringPage';
import { ModerationPage } from '@/features/moderation/ModerationPage';
import { SettingsPage } from '@/features/settings/SettingsPage';
import { LoginPage } from '@/features/authentication/LoginPage';
import { AccessDeniedPage } from '@/features/authentication/AccessDeniedPage';
import { NotFoundPage } from '@/components/feedback/NotFoundPage';

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<DashboardPage />} />
        <Route path="villages" element={<VillagesPage />} />
        <Route path="sessions" element={<SessionsPage />} />
        <Route path="monitoring" element={<MonitoringPage />} />
        <Route path="moderation" element={<ModerationPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="access-denied" element={<AccessDeniedPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
