import { formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import type { FleetDevice } from '@/shared/tower'
import { cn } from '@/shared/lib/utils'
import { ChevronRight, WifiOff } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Sparkline } from './Sparkline'

interface MiniBtopProps {
  device: FleetDevice
}

function rate(value = 0): string {
  return `${formatBytes(value)}/s`
}

function CoreBars({ cores }: { cores?: number[] }) {
  if (!cores?.length) return <div className="h-5" />
  return (
    <div className="flex h-5 items-end gap-[2px]" title="Per-core CPU usage">
      {cores.slice(0, 32).map((core, index) => (
        <span
          key={index}
          className={cn(
            'flex-1 rounded-[1px]',
            core > 85 ? 'bg-red-400' : core > 60 ? 'bg-amber-400' : 'bg-emerald-400/80'
          )}
          style={{ height: `${Math.max(8, Math.min(100, core))}%` }}
        />
      ))}
    </div>
  )
}

export function MiniBtop({ device }: MiniBtopProps) {
  const stats = device.stats
  const online = device.status === 'online'
  const cpu = stats?.cpu.usage_percent ?? 0
  const mem = stats?.memory.used_percent ?? 0
  const rx = stats?.network.rx_rate ?? 0
  const tx = stats?.network.tx_rate ?? 0

  const [cpuHistory, setCpuHistory] = useState<number[]>([])
  const [netHistory, setNetHistory] = useState<number[]>([])
  const lastStamp = useRef<string>('')

  /* eslint-disable react-hooks/set-state-in-effect -- roll the history forward per received stats frame */
  useEffect(() => {
    if (!stats) return
    const stamp = `${stats.uptime}-${device.last_seen}`
    if (stamp === lastStamp.current) return
    lastStamp.current = stamp
    setCpuHistory((history) => [...history.slice(-59), cpu])
    setNetHistory((history) => [...history.slice(-59), rx + tx])
  }, [cpu, device.last_seen, rx, stats, tx])
  /* eslint-enable react-hooks/set-state-in-effect */

  return (
    <Link
      to={`/fleet/${device.id}`}
      className={cn(
        'group block overflow-hidden rounded-lg border bg-[#0b0f14] font-mono text-[#c9d1d9] shadow-sm transition-colors',
        online ? 'border-white/10 hover:border-emerald-400/40' : 'border-white/5 opacity-75 hover:border-white/20'
      )}
    >
      <div className="flex items-center gap-2 border-b border-white/[0.06] px-3 py-2">
        <span
          className={cn(
            'h-2 w-2 shrink-0 rounded-full',
            online ? 'bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]' : 'bg-white/25'
          )}
        />
        <span className="truncate text-sm font-semibold text-white/90">{device.name}</span>
        <span className="ml-auto text-[10px] uppercase tracking-wider text-white/35">
          {online ? 'online' : 'offline'}
        </span>
        <ChevronRight className="h-3.5 w-3.5 text-white/20 group-hover:text-white/50" />
      </div>

      {!stats ? (
        <div className="flex h-40 flex-col items-center justify-center gap-2 text-xs text-white/30">
          <WifiOff className="h-4 w-4" />
          {online ? 'waiting for stats…' : 'no recent heartbeat'}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-x-4 gap-y-3 p-3">
          <div className="col-span-2 space-y-1.5">
            <div className="flex items-baseline justify-between">
              <span className="text-[10px] uppercase tracking-widest text-white/40">cpu</span>
              <span className="text-lg font-semibold leading-none text-emerald-400">{cpu.toFixed(0)}%</span>
            </div>
            <Sparkline values={cpuHistory} max={100} height={22} color="#34d399" />
            <CoreBars cores={stats.cpu.per_core} />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline justify-between">
              <span className="text-[10px] uppercase tracking-widest text-white/40">mem</span>
              <span className={cn('text-sm font-semibold', mem > 85 ? 'text-red-400' : 'text-cyan-300')}>
                {mem.toFixed(0)}%
              </span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-sm bg-white/10">
              <div
                className={cn('h-full', mem > 85 ? 'bg-red-400' : mem > 65 ? 'bg-amber-400' : 'bg-cyan-400')}
                style={{ width: `${Math.min(100, mem)}%` }}
              />
            </div>
            <p className="text-[10px] text-white/35">
              {formatBytes(stats.memory.used)} / {formatBytes(stats.memory.total)}
            </p>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline justify-between">
              <span className="text-[10px] uppercase tracking-widest text-white/40">net</span>
              <span className="text-[10px] text-white/50">
                <span className="text-emerald-300">↓{rate(rx)}</span> <span className="text-sky-300">↑{rate(tx)}</span>
              </span>
            </div>
            <Sparkline values={netHistory} height={20} color="#38bdf8" />
            <p className="truncate text-[10px] text-white/35">
              {(stats.network.interfaces ?? [])
                .filter((iface) => iface.up)
                .slice(0, 2)
                .map((iface) => iface.name)
                .join(' · ') || 'no interfaces'}
            </p>
          </div>

          <div className="col-span-2 flex items-center justify-between border-t border-white/[0.06] pt-2 text-[10px] text-white/35">
            <span>
              load {stats.load_avg.load1.toFixed(2)} · up {formatUptime(stats.uptime)} · {stats.processes} proc
            </span>
            <span>{formatRelativeTime(device.last_seen)}</span>
          </div>
        </div>
      )}
    </Link>
  )
}
