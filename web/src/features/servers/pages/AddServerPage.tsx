import { cn } from '@/shared/lib/utils'
import { checkServerHealth } from '@/shared/services/api'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Input } from '@/ui/input'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/ui/tooltip'
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  HelpCircle,
  Key,
  Link as LinkIcon,
  Loader2,
  Server,
  Tag,
  XCircle,
} from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useServersStore } from '../stores/servers'

type ConnectionStatus = 'idle' | 'checking' | 'success' | 'error' | 'warning'

interface FormState {
  url: string
  name: string
  apiKey: string
}

interface ValidationErrors {
  url?: string
  name?: string
  apiKey?: string
}

export function AddServerPage() {
  const navigate = useNavigate()
  const servers = useServersStore((s) => s.servers)
  const addServer = useServersStore((s) => s.addServer)
  const updateServer = useServersStore((s) => s.updateServer)

  const [form, setForm] = useState<FormState>({ url: '', name: '', apiKey: '' })
  const [errors, setErrors] = useState<ValidationErrors>({})
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('idle')
  const [connectionMessage, setConnectionMessage] = useState('')
  const [loading, setLoading] = useState(false)

  const hasServers = servers.length > 0

  const updateField = (field: keyof FormState, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    setErrors((prev) => ({ ...prev, [field]: undefined }))
    // Reset connection status when URL changes
    if (field === 'url') {
      setConnectionStatus('idle')
      setConnectionMessage('')
    }
  }

  const validateUrl = (url: string): string | undefined => {
    if (!url.trim()) return 'Server URL is required'
    try {
      const parsed = new URL(url)
      if (!['http:', 'https:'].includes(parsed.protocol)) {
        return 'URL must use http or https protocol'
      }
    } catch {
      return 'Invalid URL format'
    }
    return undefined
  }

  const testConnection = async () => {
    const serverUrl = form.url.trim().replace(/\/$/, '')
    const urlError = validateUrl(serverUrl)
    if (urlError) {
      setErrors((prev) => ({ ...prev, url: urlError }))
      return
    }

    setConnectionStatus('checking')
    setConnectionMessage('Testing connection...')

    try {
      await checkServerHealth(serverUrl)
      setConnectionStatus('success')
      setConnectionMessage('Server is reachable')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Connection failed'
      if (message.includes('401') || message.includes('Unauthorized')) {
        setConnectionStatus('warning')
        setConnectionMessage('Server requires authentication')
      } else {
        setConnectionStatus('error')
        setConnectionMessage(message)
      }
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const serverUrl = form.url.trim().replace(/\/$/, '')
    const urlError = validateUrl(serverUrl)

    if (urlError) {
      setErrors({ url: urlError })
      return
    }

    const serverName = form.name.trim() || new URL(serverUrl).hostname

    // Check for duplicate URLs
    const existingServer = servers.find((s) => s.url.replace(/\/$/, '').toLowerCase() === serverUrl.toLowerCase())
    if (existingServer) {
      setErrors({ url: `Server already exists as "${existingServer.name}"` })
      return
    }

    setLoading(true)

    try {
      // Test connection if not already tested
      if (connectionStatus !== 'success') {
        await checkServerHealth(serverUrl)
      }

      const serverId = addServer(serverUrl, serverName, form.apiKey.trim() || undefined)
      updateServer(serverId, { status: 'online' })

      toast.success(`Server "${serverName}" added successfully`)
      navigate('/')
    } catch (err) {
      if (form.apiKey.trim()) {
        // If API key provided, add server anyway (might be auth issue we can't test)
        const serverId = addServer(serverUrl, serverName, form.apiKey.trim())
        updateServer(serverId, { status: 'connecting' })
        toast.success(`Server "${serverName}" added`, {
          description: 'Connection will be verified when you access the server.',
        })
        navigate('/')
      } else {
        const message = err instanceof Error ? err.message : 'Connection failed'
        setConnectionStatus('error')
        setConnectionMessage(message)
        setErrors({ url: 'Cannot connect to server. Add an API key to continue anyway.' })
      }
    } finally {
      setLoading(false)
    }
  }

  const statusIndicator = {
    idle: null,
    checking: (
      <Badge variant="secondary" className="gap-1.5">
        <Loader2 className="h-3 w-3 animate-spin" />
        Checking...
      </Badge>
    ),
    success: (
      <Badge className="gap-1.5 bg-emerald-500/10 text-emerald-600 border-emerald-500/20">
        <CheckCircle2 className="h-3 w-3" />
        Connected
      </Badge>
    ),
    warning: (
      <Badge className="gap-1.5 bg-yellow-500/10 text-yellow-600 border-yellow-500/20">
        <AlertTriangle className="h-3 w-3" />
        Auth Required
      </Badge>
    ),
    error: (
      <Badge variant="destructive" className="gap-1.5">
        <XCircle className="h-3 w-3" />
        Failed
      </Badge>
    ),
  }

  return (
    <TooltipProvider>
      <div className="min-h-screen flex items-center justify-center bg-linear-to-b from-background to-muted/30 p-4">
        <Card className="w-full max-w-lg shadow-lg">
          <CardHeader className="text-center pb-2">
            <div className="mx-auto mb-4 h-14 w-14 rounded-2xl bg-primary/10 flex items-center justify-center">
              <Server className="h-7 w-7 text-primary" />
            </div>
            <CardTitle className="text-2xl">{hasServers ? 'Add Server' : 'Welcome to Gloski'}</CardTitle>
            <CardDescription className="text-base">
              {hasServers ? 'Connect to another Gloski server' : 'Add your first server to get started'}
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-4">
            <form onSubmit={handleSubmit} className="space-y-5">
              {/* Server URL */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium flex items-center gap-2">
                    <LinkIcon className="h-4 w-4 text-muted-foreground" />
                    Server URL
                  </label>
                  {statusIndicator[connectionStatus]}
                </div>
                <div className="flex gap-2">
                  <Input
                    type="url"
                    placeholder="https://server.example.com:8080"
                    value={form.url}
                    onChange={(e) => updateField('url', e.target.value)}
                    disabled={loading}
                    autoFocus
                    className={cn(errors.url && 'border-destructive focus-visible:ring-destructive')}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={testConnection}
                    disabled={loading || connectionStatus === 'checking' || !form.url.trim()}
                  >
                    {connectionStatus === 'checking' ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Test'}
                  </Button>
                </div>
                {errors.url ? (
                  <p className="text-sm text-destructive flex items-start gap-1.5">
                    <XCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                    <span className="whitespace-pre-line">{errors.url}</span>
                  </p>
                ) : connectionMessage && connectionStatus !== 'idle' ? (
                  <div
                    className={cn(
                      'text-sm flex items-start gap-1.5',
                      connectionStatus === 'success' && 'text-emerald-600',
                      connectionStatus === 'warning' && 'text-yellow-600',
                      connectionStatus === 'error' && 'text-destructive'
                    )}
                  >
                    {connectionStatus === 'success' && <CheckCircle2 className="h-3.5 w-3.5 shrink-0 mt-0.5" />}
                    {connectionStatus === 'warning' && <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />}
                    {connectionStatus === 'error' && <XCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />}
                    <span className="whitespace-pre-line">{connectionMessage}</span>
                  </div>
                ) : null}
              </div>

              {/* Server Name */}
              <div className="space-y-2">
                <label className="text-sm font-medium flex items-center gap-2">
                  <Tag className="h-4 w-4 text-muted-foreground" />
                  Display Name
                  <span className="text-muted-foreground font-normal">(optional)</span>
                </label>
                <Input
                  placeholder="My Production Server"
                  value={form.name}
                  onChange={(e) => updateField('name', e.target.value)}
                  disabled={loading}
                />
                <p className="text-xs text-muted-foreground">
                  A friendly name to identify this server. Defaults to hostname if empty.
                </p>
              </div>

              {/* API Key */}
              <div className="space-y-2">
                <label className="text-sm font-medium flex items-center gap-2">
                  <Key className="h-4 w-4 text-muted-foreground" />
                  API Key
                  <span className="text-muted-foreground font-normal">(recommended)</span>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <HelpCircle className="h-3.5 w-3.5 text-muted-foreground cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent side="top" className="max-w-xs">
                      <p>
                        Set <code className="bg-muted px-1 rounded">GLOSKI_API_KEY</code> on your server and enter the
                        same value here for authentication.
                      </p>
                    </TooltipContent>
                  </Tooltip>
                </label>
                <Input
                  type="password"
                  placeholder="Enter your API key"
                  value={form.apiKey}
                  onChange={(e) => updateField('apiKey', e.target.value)}
                  disabled={loading}
                />
                {connectionStatus === 'warning' && !form.apiKey && (
                  <p className="text-xs text-yellow-600 flex items-center gap-1.5">
                    <AlertTriangle className="h-3 w-3 shrink-0" />
                    This server requires authentication. Enter your API key above.
                  </p>
                )}
              </div>

              {/* Submit */}
              <div className="pt-2 space-y-3">
                <Button type="submit" className="w-full h-11" disabled={loading}>
                  {loading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Connecting...
                    </>
                  ) : (
                    <>
                      <Server className="h-4 w-4 mr-2" />
                      Add Server
                    </>
                  )}
                </Button>

                {hasServers && (
                  <Button
                    type="button"
                    variant="ghost"
                    className="w-full"
                    onClick={() => navigate('/')}
                    disabled={loading}
                  >
                    <ArrowLeft className="h-4 w-4 mr-2" />
                    Back to Dashboard
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </TooltipProvider>
  )
}
