import { ServerProvider } from '@/features/servers'
import { Outlet } from 'react-router-dom'

/**
 * Layout component for server-scoped routes.
 * Wraps all server routes with ServerProvider context.
 */
export function ServerLayout() {
  return (
    <ServerProvider>
      <Outlet />
    </ServerProvider>
  )
}
