import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'
import { NavSection } from './NavSection'
import { serverNavItems } from '@/app/navigation'
import { useServersStore } from '@/features/servers'

interface ServerNavProps {
  collapsed?: boolean
}

export function ServerNav({ collapsed }: ServerNavProps) {
  const { serverId } = useParams()
  const servers = useServersStore((s) => s.servers)
  const server = servers.find((srv) => srv.id === serverId)

  if (!serverId || !server) return null

  return (
    <NavSection title={server.name} collapsed={collapsed}>
      {serverNavItems.map((item) => (
        <NavItem
          key={item.id}
          to={item.path.replace(':serverId', serverId)}
          icon={item.icon}
          label={item.name}
          end={item.id === 'server-overview'}
          collapsed={collapsed}
        />
      ))}
    </NavSection>
  )
}
