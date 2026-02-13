import { AppLayout, ServerLayout } from '@/layouts'
import { RouteErrorBoundary } from '@/shared/components'
import { Toaster } from '@/ui/sonner'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'

// Create route configuration for SPA
const router = createBrowserRouter([
  {
    element: <AppLayout />,
    errorElement: <RouteErrorBoundary />,
    children: [
      // Dashboard
      {
        path: '/',
        lazy: {
          Component: async () => (await import('@/features/dashboard')).default,
        },
      },
      // Settings
      {
        path: '/settings',
        lazy: {
          Component: async () => (await import('@/features/settings')).default,
        },
      },
      // Servers - parent route grouping all server-related pages
      {
        path: '/servers',
        children: [
          // Servers list (index route)
          {
            index: true,
            lazy: {
              Component: async () => (await import('@/features/servers')).default,
            },
          },
          // Add server page
          {
            path: 'add',
            lazy: {
              Component: async () => (await import('@/features/servers')).AddServerPage,
            },
          },
          // Server-scoped features - wrapped with ServerLayout (provides ServerProvider context)
          {
            path: ':serverId',
            element: <ServerLayout />,
            errorElement: <RouteErrorBoundary />,
            children: [
              // Server overview (index route)
              {
                index: true,
                lazy: {
                  Component: async () => (await import('@/features/servers')).ServerDetailPage,
                },
              },
              // Server sub-pages
              {
                path: 'files',
                lazy: {
                  Component: async () => (await import('@/features/files')).default,
                },
              },
              {
                path: 'metrics',
                lazy: {
                  Component: async () => (await import('@/features/servers')).ServerMetricsPage,
                },
              },
              {
                path: 'search',
                lazy: {
                  Component: async () => (await import('@/features/search')).default,
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
            ],
          },
        ],
      },
      // Catch all - redirect to dashboard
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
    ],
  },
])

function App() {
  return (
    <>
      <RouterProvider router={router} />
      <Toaster position="bottom-right" richColors />
    </>
  )
}

export default App
