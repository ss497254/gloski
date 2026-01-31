import { AppLayout, ServerLayout } from '@/layouts'
import { RouteErrorBoundary } from '@/shared/components'
import { Toaster } from '@/ui/sonner'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'

// Special pages
import { AddServerPage } from '@/features/servers'

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
        errorElement: <RouteErrorBoundary />,
      },
      // Settings
      {
        path: '/settings',
        lazy: {
          Component: async () => (await import('@/features/settings')).default,
        },
        errorElement: <RouteErrorBoundary />,
      },
      // Servers - parent route grouping all server-related pages
      {
        path: '/servers',
        errorElement: <RouteErrorBoundary />,
        children: [
          // Servers list (index route)
          {
            index: true,
            lazy: {
              Component: async () => (await import('@/features/servers')).default,
            },
            errorElement: <RouteErrorBoundary />,
          },
          // Add server page
          {
            path: 'add',
            element: <AddServerPage />,
            errorElement: <RouteErrorBoundary />,
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
                errorElement: <RouteErrorBoundary />,
              },
              // Server sub-pages
              {
                path: 'files',
                lazy: {
                  Component: async () => (await import('@/features/files')).default,
                },
                errorElement: <RouteErrorBoundary />,
              },
              {
                path: 'search',
                lazy: {
                  Component: async () => (await import('@/features/search')).default,
                },
                errorElement: <RouteErrorBoundary />,
              },
              {
                path: 'terminal',
                lazy: {
                  Component: async () => (await import('@/features/terminal')).default,
                },
                errorElement: <RouteErrorBoundary />,
              },
              {
                path: 'jobs',
                lazy: {
                  Component: async () => (await import('@/features/jobs')).default,
                },
                errorElement: <RouteErrorBoundary />,
              },
              {
                path: 'downloads',
                lazy: {
                  Component: async () => (await import('@/features/downloads')).default,
                },
                errorElement: <RouteErrorBoundary />,
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
