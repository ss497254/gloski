import { deviceNavItems } from '@/app/navigation'
import { useParams } from 'react-router-dom'
import { NavItem } from './NavItem'

export function DeviceNav({ collapsed }: { collapsed?: boolean }) {
  const { deviceId } = useParams<{ deviceId: string }>()
  if (!deviceId) return null

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
