import { serverNavItems } from '@/app/navigation'
import { useServersStore } from '@/shared/store/servers'
import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'
import { NavSection } from './NavSection'

interface ServerNavProps {
  collapsed?: boolean
}

export function ServerNav({ collapsed }: ServerNavProps) {
  const { serverId } = useParams()
  const servers = useServersStore((s) => s.servers)
  const server = servers.find((srv) => srv.id === serverId)

  if (!serverId || !server) return null

  const isDevice = server.managedBy === 'tower'
  const deviceId = serverId.replace(/^tower-/, '')
  const items = isDevice ? serverNavItems.filter((item) => item.id !== 'metrics') : serverNavItems

  return (
    <NavSection title={server.name} collapsed={collapsed}>
      {items.map((item) => {
        const to =
          item.id === 'server-overview'
            ? isDevice
              ? `/fleet/${deviceId}`
              : `/servers/${serverId}`
            : item.path.replace(':serverId', serverId)
        return (
          <NavItem
            key={item.id}
            to={to}
            icon={item.icon}
            label={item.name}
            end={item.id === 'server-overview'}
            collapsed={collapsed}
          />
        )
      })}
    </NavSection>
  )
}
