import { AppLayout, DeviceLayout } from '@/layouts'
import { RouteErrorBoundary } from '@/shared/components'
import { TowerProvider } from '@/shared/tower'
import { Toaster } from '@/ui/sonner'
import { createBrowserRouter, Navigate, RouterProvider, useLocation, useParams } from 'react-router-dom'

function LegacyRedirect({ prefix }: { prefix: '/fleet' | '/servers' }) {
  const params = useParams<{ deviceId?: string; serverId?: string }>()
  const location = useLocation()
  const id = (params.deviceId ?? params.serverId ?? '').replace(/^tower-/, '')
  const remainder = location.pathname.replace(new RegExp(`^${prefix}/[^/]+`), '')
  if (!id) return <Navigate to="/devices" replace />
  return <Navigate to={`/devices/${id}${remainder}`} replace />
}

const router = createBrowserRouter([
  {
    element: <AppLayout />,
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        path: '/',
        lazy: {
          Component: async () => (await import('@/features/dashboard')).DashboardPage,
        },
      },
      {
        path: '/devices',
        lazy: {
          Component: async () => (await import('@/features/fleet')).FleetPage,
        },
      },
      {
        path: '/devices/:deviceId',
        element: <DeviceLayout />,
        errorElement: <RouteErrorBoundary />,
        children: [
          {
            index: true,
            lazy: {
              Component: async () => (await import('@/features/fleet')).DevicePage,
            },
          },
          {
            path: 'files',
            lazy: {
              Component: async () => (await import('@/features/files')).default,
            },
          },
          {
            path: 'terminal',
            lazy: {
              Component: async () => (await import('@/features/terminal')).default,
            },
          },
          {
            path: 'jobs',
            lazy: {
              Component: async () => (await import('@/features/jobs')).default,
            },
          },
        ],
      },
      {
        path: '/fleet',
        element: <Navigate to="/devices" replace />,
      },
      {
        path: '/fleet/:deviceId',
        element: <LegacyRedirect prefix="/fleet" />,
      },
      {
        path: '/servers/:serverId/*',
        element: <LegacyRedirect prefix="/servers" />,
      },
      {
        path: '/settings',
        lazy: {
          Component: async () => (await import('@/features/dashboard')).SettingsPage,
        },
      },
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
    ],
  },
])

function App() {
  return (
    <TowerProvider>
      <RouterProvider router={router} />
      <Toaster position="bottom-right" richColors />
    </TowerProvider>
  )
}

export default App
