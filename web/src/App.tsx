import { Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AppLayout } from '@/layouts'
import { PageLoader } from '@/shared/components'
import { Toaster } from '@/ui/sonner'
import { features, serverFeatures } from '@/app/feature-registry'

// Special pages not in registry
import { AddServerPage } from '@/features/servers'

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

            {/* Server-scoped features from registry */}
            {serverFeatures.map((feature) => (
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
