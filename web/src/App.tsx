import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { useServersStore } from '@/stores/servers'
import { Layout } from '@/components/Layout'
import { PageLoader } from '@/components/common'
import { Toaster } from '@/components/ui/sonner'

// Lazy load pages for code splitting
const AddServerPage = lazy(() =>
  import('@/pages/AddServer').then((m) => ({ default: m.AddServerPage }))
)
const DashboardPage = lazy(() =>
  import('@/pages/Dashboard').then((m) => ({ default: m.DashboardPage }))
)
const ServerDetailPage = lazy(() =>
  import('@/pages/ServerDetail').then((m) => ({ default: m.ServerDetailPage }))
)
const FilesPage = lazy(() =>
  import('@/pages/Files').then((m) => ({ default: m.FilesPage }))
)
const SearchPage = lazy(() =>
  import('@/pages/Search').then((m) => ({ default: m.SearchPage }))
)
const TerminalPage = lazy(() =>
  import('@/pages/Terminal').then((m) => ({ default: m.TerminalPage }))
)
const TasksPage = lazy(() =>
  import('@/pages/Tasks').then((m) => ({ default: m.TasksPage }))
)

function AppLayout() {
  return (
    <Layout>
      <Suspense fallback={<PageLoader />}>
        <Outlet />
      </Suspense>
    </Layout>
  )
}

function RequireServers() {
  const { servers } = useServersStore()

  // If no servers configured, redirect to add server page
  if (servers.length === 0) {
    return <Navigate to="/add-server" replace />
  }

  return <Outlet />
}

function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          {/* Add server (no layout) */}
          <Route path="/add-server" element={<AddServerPage />} />

          {/* Main app with layout */}
          <Route element={<AppLayout />}>
            {/* Routes that require at least one server */}
            <Route element={<RequireServers />}>
              <Route path="/" element={<DashboardPage />} />
              
              {/* Server-scoped routes */}
              <Route path="/servers/:serverId" element={<ServerDetailPage />} />
              <Route path="/servers/:serverId/files" element={<FilesPage />} />
              <Route path="/servers/:serverId/search" element={<SearchPage />} />
              <Route path="/servers/:serverId/terminal" element={<TerminalPage />} />
              <Route path="/servers/:serverId/tasks" element={<TasksPage />} />
            </Route>
          </Route>

          {/* Catch all - redirect to dashboard */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
      <Toaster position="bottom-right" richColors />
    </BrowserRouter>
  )
}

export default App
