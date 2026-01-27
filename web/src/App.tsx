import { features, serverFeatures } from '@/app/feature-registry'
import { AppLayout } from '@/layouts'
import { PageLoader } from '@/shared/components'
import { Toaster } from '@/ui/sonner'
import { Suspense } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

// Special pages and providers
import { AddServerPage, ServerProvider } from '@/features/servers'

function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          {/* Add server (no layout) */}
          <Route path="/servers/add" element={<AddServerPage />} />

          {/* Main app with layout */}
          <Route element={<AppLayout />}>
            {/* Global features from registry */}
            {features.map((feature) => (
              <Route
                key={feature.id}
                path={feature.path}
                element={
                  <Suspense fallback={<PageLoader />}>
                    <feature.component />
                  </Suspense>
                }
              />
            ))}

            {/* Server-scoped features from registry - wrapped with ServerProvider */}
            {serverFeatures.map((feature) => (
              <Route
                key={feature.id}
                path={feature.path}
                element={
                  <ServerProvider>
                    <Suspense fallback={<PageLoader />}>
                      <feature.component />
                    </Suspense>
                  </ServerProvider>
                }
              />
            ))}
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
