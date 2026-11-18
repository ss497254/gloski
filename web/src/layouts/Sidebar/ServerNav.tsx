import { serverNavItems } from '@/app/navigation'
import { useServersStore } from '@/shared/store/servers'
import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'
import { NavSection } from './NavSection'

interface ServerNavProps {
  collapsed?: boolean
  serverId?: string
}

export function ServerNav({ collapsed, serverId: serverIdProp }: ServerNavProps) {
  const params = useParams()
  const serverId = serverIdProp ?? params.serverId
  const servers = useServersStore((s) => s.servers)
  const server = servers.find((srv) => srv.id === serverId)

  if (!serverId || !server) return null

  const deviceId = serverId.replace(/^tower-/, '')

  return (
    <NavSection title={server.name} collapsed={collapsed}>
      {serverNavItems.map((item) => (
        <NavItem
          key={item.id}
          to={item.path.replace(':serverId', item.id === 'server-overview' ? deviceId : serverId)}
          icon={item.icon}
          label={item.name}
          end={item.id === 'server-overview'}
          collapsed={collapsed}
        />
      ))}
    </NavSection>
  )
}
