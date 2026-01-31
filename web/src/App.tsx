import { features, serverFeatures } from '@/app/feature-registry'
import { AppLayout } from '@/layouts'
import { PageLoader } from '@/shared/components'
import { Toaster } from '@/ui/sonner'
import { Suspense } from 'react'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'

// Special pages and providers
import { AddServerPage, ServerProvider } from '@/features/servers'

// Create route objects
const router = createBrowserRouter([
  {
    path: '/servers/add',
    element: <AddServerPage />,
  },
  {
    element: <AppLayout />,
    children: [
      // Global features from registry
      ...features.map((feature) => ({
        path: feature.path,
        element: (
          <Suspense fallback={<PageLoader />}>
            <feature.component />
          </Suspense>
        ),
      })),
      // Server-scoped features from registry - wrapped with ServerProvider
      ...serverFeatures.map((feature) => ({
        path: feature.path,
        element: (
          <ServerProvider>
            <Suspense fallback={<PageLoader />}>
              <feature.component />
            </Suspense>
          </ServerProvider>
        ),
      })),
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
