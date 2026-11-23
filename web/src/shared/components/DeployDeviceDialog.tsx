import { deployDevice, getDeployment, listTailnetPeers } from '@/shared/tower/client'
import type { Deployment, TailnetNode } from '@/shared/tower/types'
import { Alert, AlertDescription } from '@/ui/alert'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/ui/dialog'
import { Input } from '@/ui/input'
import { Label } from '@/ui/label'
import { NativeSelect, NativeSelectOption } from '@/ui/native-select'
import { Spinner } from '@/ui/spinner'
import { Rocket, Server } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { toast } from 'sonner'

const DEFAULT_SSH_PORT = '22'
const POLL_INITIAL_DELAY_MS = 700
const POLL_INTERVAL_MS = 1000
const POLL_MAX_ATTEMPTS = 150

interface DeployDeviceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeployed: () => void
}

interface FormState {
  peer: string
  host: string
  name: string
  user: string
  sshPort: string
}

function peerKey(peer: TailnetNode): string {
  return peer.dns_name || peer.ips[0] || peer.hostname
}

function peerHost(peer: TailnetNode): string {
  return peer.ips.find((ip) => ip.includes('.')) ?? peer.ips[0] ?? peer.dns_name
}

function statusLabel(status: Deployment['status']): string {
  switch (status) {
    case 'pending':
    case 'uploading':
      return 'Copying the agent binary to /tmp…'
    case 'starting':
      return 'Starting the agent over SSH…'
    case 'running':
      return 'Agent started. Waiting for it to enroll…'
    default:
      return 'Working…'
  }
}

export function DeployDeviceDialog({ open, onOpenChange, onDeployed }: DeployDeviceDialogProps) {
  const [peers, setPeers] = useState<TailnetNode[]>([])
  const [peersAvailable, setPeersAvailable] = useState(false)
  const [peersError, setPeersError] = useState<string | null>(null)
  const [loadingPeers, setLoadingPeers] = useState(false)
  const [form, setForm] = useState<FormState>({
    peer: '',
    host: '',
    name: '',
    user: '',
    sshPort: DEFAULT_SSH_PORT,
  })
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<'idle' | 'deploying' | 'failed'>('idle')
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const deploymentId = deployment?.id

  useEffect(() => {
    if (!open) return
    let cancelled = false

    const bootstrap = async () => {
      setLoadingPeers(true)
      setError(null)
      setStatus('idle')
      setDeployment(null)

      let response: Awaited<ReturnType<typeof listTailnetPeers>> | null = null
      try {
        response = await listTailnetPeers()
      } catch (err) {
        if (!cancelled) setPeersError(err instanceof Error ? err.message : 'Could not reach the tailnet API')
      }
      if (cancelled) return

      const availablePeers = response?.peers ?? []
      setPeers(availablePeers)
      setPeersAvailable(Boolean(response?.available))
      setPeersError(response?.error ?? null)

      const firstPeer = availablePeers.find((peer) => peer.online && !peer.self)
      setForm({
        peer: firstPeer ? peerKey(firstPeer) : '',
        host: firstPeer ? peerHost(firstPeer) : '',
        name: firstPeer?.hostname ?? '',
        user: localStorage.getItem('gloski-ssh-user') ?? '',
        sshPort: DEFAULT_SSH_PORT,
      })
      setLoadingPeers(false)
    }

    void bootstrap()
    return () => {
      cancelled = true
    }
  }, [open])

  useEffect(() => {
    if (!deploymentId || status !== 'deploying') return
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null
    let attempts = 0

    const poll = async () => {
      try {
        const latest = await getDeployment(deploymentId)
        if (cancelled) return
        setDeployment(latest)
        if (latest.status === 'running') {
          localStorage.setItem('gloski-ssh-user', form.user)
          toast.success(`Agent deployed to ${latest.name}`)
          onDeployed()
          onOpenChange(false)
          return
        }
        if (latest.status === 'failed') {
          setError(latest.error || 'Deployment failed')
          setStatus('failed')
          return
        }
        attempts += 1
        if (attempts > POLL_MAX_ATTEMPTS) {
          setError('Deployment timed out waiting for the agent. Check the master logs.')
          setStatus('failed')
          return
        }
        timer = setTimeout(poll, POLL_INTERVAL_MS)
      } catch (err) {
        if (cancelled) return
        setError(err instanceof Error ? err.message : 'Failed to read deployment status')
        setStatus('failed')
      }
    }

    timer = setTimeout(poll, POLL_INITIAL_DELAY_MS)
    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [deploymentId, status, form.user, onDeployed, onOpenChange])

  const update = (field: keyof FormState, value: string) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const choosePeer = (key: string) => {
    const peer = peers.find((item) => peerKey(item) === key)
    setForm((current) => ({
      ...current,
      peer: key,
      host: peer ? peerHost(peer) : '',
      name: peer?.hostname ?? '',
    }))
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!form.host.trim() || !form.user.trim() || status === 'deploying') {
      setError('Host and SSH user are required')
      return
    }

    setError(null)
    setStatus('deploying')
    try {
      const created = await deployDevice({
        name: form.name.trim() || form.host.trim(),
        host: form.host.trim(),
        user: form.user.trim(),
        ssh_port: Number(form.sshPort) || 22,
      })
      setDeployment(created)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start deployment')
      setStatus('failed')
    }
  }

  const busy = status === 'deploying' || loadingPeers

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Rocket className="h-4 w-4 text-success" />
            Deploy an agent
          </DialogTitle>
          <DialogDescription>
            The master installs the agent over SSH and keeps it running: if the agent stops or the device reboots, the
            master installs it again.
          </DialogDescription>
        </DialogHeader>

        {!peersAvailable && peersError && (
          <Alert variant="warning">
            <AlertDescription>Tailnet discovery unavailable ({peersError}). Enter the device address manually.</AlertDescription>
          </Alert>
        )}

        <form onSubmit={submit} className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground" htmlFor="deploy-peer">
                Tailnet device
              </Label>
              <NativeSelect
                id="deploy-peer"
                className="w-full"
                value={form.peer}
                onChange={(event) => choosePeer(event.target.value)}
                disabled={busy}
              >
                <NativeSelectOption value="">Manual entry</NativeSelectOption>
                {peers.map((peer) => (
                  <NativeSelectOption key={peerKey(peer)} value={peerKey(peer)}>
                    {peer.hostname} · {peer.os} · {peer.online ? 'online' : 'offline'}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground" htmlFor="deploy-host">
                SSH host
              </Label>
              <Input
                id="deploy-host"
                placeholder="100.x.y.z or device.ts.net"
                value={form.host}
                onChange={(event) => {
                  update('host', event.target.value)
                  if (form.peer) update('peer', '')
                }}
                disabled={busy}
              />
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground" htmlFor="deploy-user">
                SSH user
              </Label>
              <Input
                id="deploy-user"
                placeholder="root"
                value={form.user}
                onChange={(event) => update('user', event.target.value)}
                disabled={busy}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground" htmlFor="deploy-name">
                Display name
              </Label>
              <Input
                id="deploy-name"
                placeholder="defaults to hostname"
                value={form.name}
                onChange={(event) => update('name', event.target.value)}
                disabled={busy}
              />
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground" htmlFor="deploy-ssh-port">
                SSH port
              </Label>
              <Input
                id="deploy-ssh-port"
                inputMode="numeric"
                value={form.sshPort}
                onChange={(event) => update('sshPort', event.target.value)}
                disabled={busy}
              />
            </div>
          </div>

          <p className="text-[11px] text-muted-foreground">
            The master chooses the agent's gRPC endpoint automatically. SSH keys or an agent are read from the machine
            running the master.
          </p>

          {deployment && status === 'deploying' && (
            <Alert role="status" className="text-xs">
              <Spinner className="text-success" />
              <AlertDescription>{statusLabel(deployment.status)}</AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertDescription className="font-mono text-xs">{error}</AlertDescription>
            </Alert>
          )}

          <DialogFooter className="pt-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={status === 'deploying'}>
              Cancel
            </Button>
            <Button type="submit" disabled={busy || !form.host.trim() || !form.user.trim()}>
              {status === 'deploying' ? (
                <Spinner />
              ) : (
                <Server className="mr-2 h-4 w-4" />
              )}
              {status === 'deploying' ? 'Deploying…' : 'Deploy agent'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
