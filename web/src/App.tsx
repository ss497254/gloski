import { AppLayout, ServerLayout } from '@/layouts'
import { RouteErrorBoundary } from '@/shared/components'
import { TowerProvider } from '@/shared/tower'
import { Toaster } from '@/ui/sonner'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'

// The dashboard is master-only. Device tools are mounted under internal
// /servers/<virtual-device-id> routes but are reached from the Device page.
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
        path: '/fleet',
        lazy: {
          Component: async () => (await import('@/features/fleet')).FleetPage,
        },
      },
      {
        path: '/fleet/:deviceId',
        lazy: {
          Component: async () => (await import('@/features/fleet')).DevicePage,
        },
      },
      {
        path: '/settings',
        lazy: {
          Component: async () => (await import('@/features/dashboard')).SettingsPage,
        },
      },
      {
        path: '/servers/:serverId',
        element: <ServerLayout />,
        errorElement: <RouteErrorBoundary />,
        children: [
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
          {
            path: 'downloads',
            lazy: {
              Component: async () => (await import('@/features/downloads')).default,
            },
          },
          {
            path: 'search',
            lazy: {
              Component: async () => (await import('@/features/search')).default,
            },
          },
        ],
      },
      {
        path: '*',
        element: <Navigate to="/fleet" replace />,
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
