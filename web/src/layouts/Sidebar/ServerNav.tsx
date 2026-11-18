import { deviceNavItems } from '@/app/navigation'
import { useServersStore } from '@/shared/store/servers'
import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'

interface ServerNavProps {
  collapsed?: boolean
  serverId?: string
}

export function ServerNav({ collapsed, serverId: serverIdProp }: ServerNavProps) {
  const params = useParams()
  const serverId = serverIdProp ?? params.serverId ?? (params.deviceId ? `tower-${params.deviceId}` : undefined)
  const servers = useServersStore((s) => s.servers)
  const server = servers.find((srv) => srv.id === serverId)

  if (!serverId || !server) return null
  const deviceId = serverId.replace(/^tower-/, '')

  return (
    <div className="space-y-1">
      {deviceNavItems.map((item) => (
        <NavItem
          key={item.id}
          to={item.path.replace(':deviceId', deviceId)}
          icon={item.icon}
          label={item.name}
          end={item.id === 'overview'}
          collapsed={collapsed}
        />
      ))}
    </div>
  )
}
