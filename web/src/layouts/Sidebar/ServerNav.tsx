import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'
import { NavSection } from './NavSection'
import { serverFeatures } from '@/app/feature-registry'
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
      {serverFeatures.map((feature) => (
        <NavItem
          key={feature.id}
          to={feature.path.replace(':serverId', serverId)}
          icon={feature.icon}
          label={feature.name}
          end={feature.id === 'server-overview'}
          collapsed={collapsed}
        />
      ))}
    </NavSection>
  )
}
