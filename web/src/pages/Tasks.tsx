import { useState, useEffect, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useServer } from '@/hooks'
import type { Task, SystemdUnit } from '@/lib/types'
import {
  Play,
  Square,
  Trash2,
  RefreshCw,
  Terminal,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
} from 'lucide-react'

export function TasksPage() {
  const { server, api } = useServer()

  const [tasks, setTasks] = useState<Task[]>([])
  const [units, setUnits] = useState<SystemdUnit[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [newCommand, setNewCommand] = useState('')
  const [newCwd, setNewCwd] = useState('')

  const [selectedTask, setSelectedTask] = useState<string | null>(null)
  const [taskLogs, setTaskLogs] = useState<string[]>([])

  const [activeTab, setActiveTab] = useState<'tasks' | 'systemd'>('tasks')
  const [userMode, setUserMode] = useState(true)

  // Redirect if no server
  if (!server || !api) {
    return <Navigate to="/" replace />
  }

  const fetchTasks = useCallback(async () => {
    if (!api) return
    try {
      const data = await api.listTasks()
      setTasks(data.tasks || [])
    } catch (err) {
      setError('Failed to fetch tasks')
    }
  }, [api])

  const fetchSystemd = useCallback(async () => {
    if (!api) return
    try {
      const data = await api.listSystemd(userMode)
      setUnits(data.units || [])
    } catch (err) {
      setError('Failed to fetch systemd units')
    }
  }, [api, userMode])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    await Promise.all([fetchTasks(), fetchSystemd()])
    setLoading(false)
  }, [fetchTasks, fetchSystemd])

  useEffect(() => {
    loadData()
    const interval = setInterval(fetchTasks, 5000)
    return () => clearInterval(interval)
  }, [loadData, fetchTasks])

  useEffect(() => {
    fetchSystemd()
  }, [fetchSystemd])

  const startTask = async () => {
    if (!api || !newCommand.trim()) return

    try {
      await api.startTask(newCommand, newCwd || undefined)
      setNewCommand('')
      setNewCwd('')
      fetchTasks()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start task')
    }
  }

  const stopTask = async (id: string) => {
    if (!api) return
    try {
      await api.stopTask(id)
      fetchTasks()
    } catch (err) {
      setError('Failed to stop task')
    }
  }

  const fetchTaskLogs = async (id: string) => {
    if (!api) return
    try {
      const data = await api.getTaskLogs(id)
      setTaskLogs(data.logs ? [data.logs] : [])
      setSelectedTask(id)
    } catch (err) {
      setError('Failed to fetch logs')
    }
  }

  const systemdAction = async (unit: string, action: string) => {
    if (!api) return
    try {
      await api.systemdAction(unit, action, userMode)
      fetchSystemd()
    } catch (err) {
      setError(`Failed to ${action} ${unit}`)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <Clock className="h-4 w-4 text-blue-500 animate-pulse" />
      case 'completed':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />
      case 'stopped':
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
      default:
        return null
    }
  }

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleTimeString()
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="border-b p-4 flex items-center gap-4">
        <Terminal className="h-5 w-5" />
        <h1 className="text-lg font-semibold">Tasks</h1>

        <div className="flex-1" />

        <div className="flex gap-2">
          <Button
            variant={activeTab === 'tasks' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setActiveTab('tasks')}
          >
            Processes
          </Button>
          <Button
            variant={activeTab === 'systemd' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setActiveTab('systemd')}
          >
            Systemd
          </Button>
        </div>

        <Button variant="ghost" size="icon" onClick={loadData}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive px-4 py-2 text-sm">
          {error}
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        {/* Main content */}
        <div className="flex-1 overflow-auto p-4">
          {activeTab === 'tasks' ? (
            <div className="space-y-4">
              {/* New task form */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Start New Task</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <Input
                    placeholder="Command (e.g., npm run dev)"
                    value={newCommand}
                    onChange={(e) => setNewCommand(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && startTask()}
                  />
                  <div className="flex gap-2">
                    <Input
                      placeholder="Working directory (optional)"
                      value={newCwd}
                      onChange={(e) => setNewCwd(e.target.value)}
                      className="flex-1"
                    />
                    <Button onClick={startTask} disabled={!newCommand.trim()}>
                      <Play className="h-4 w-4 mr-2" />
                      Start
                    </Button>
                  </div>
                </CardContent>
              </Card>

              {/* Task list */}
              <div className="space-y-2">
                {tasks.length === 0 ? (
                  <div className="text-muted-foreground text-center py-8">
                    No tasks running
                  </div>
                ) : (
                  tasks.map((task) => (
                    <Card
                      key={task.id}
                      className={`cursor-pointer transition-colors ${
                        selectedTask === task.id ? 'ring-2 ring-primary' : ''
                      }`}
                      onClick={() => fetchTaskLogs(task.id)}
                    >
                      <CardContent className="p-4">
                        <div className="flex items-center gap-3">
                          {getStatusIcon(task.status)}
                          <div className="flex-1 min-w-0">
                            <div className="font-mono text-sm truncate">
                              {task.command}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              PID: {task.pid} | Started: {formatTime(task.started_at)}
                              {task.cwd && ` | ${task.cwd}`}
                            </div>
                          </div>
                          {task.status === 'running' && (
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={(e) => {
                                e.stopPropagation()
                                stopTask(task.id)
                              }}
                            >
                              <Square className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  ))
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* User/System mode toggle */}
              <div className="flex gap-2">
                <Button
                  variant={userMode ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setUserMode(true)}
                >
                  User Services
                </Button>
                <Button
                  variant={!userMode ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setUserMode(false)}
                >
                  System Services
                </Button>
              </div>

              {/* Unit list */}
              <div className="space-y-2">
                {units.length === 0 ? (
                  <div className="text-muted-foreground text-center py-8">
                    No services found
                  </div>
                ) : (
                  units
                    .filter((u) => u.name.endsWith('.service'))
                    .map((unit) => (
                      <Card key={unit.name}>
                        <CardContent className="p-4">
                          <div className="flex items-center gap-3">
                            <div
                              className={`h-2 w-2 rounded-full ${
                                unit.active_state === 'active'
                                  ? 'bg-green-500'
                                  : unit.active_state === 'failed'
                                    ? 'bg-red-500'
                                    : 'bg-gray-400'
                              }`}
                            />
                            <div className="flex-1 min-w-0">
                              <div className="font-mono text-sm truncate">
                                {unit.name}
                              </div>
                              <div className="text-xs text-muted-foreground truncate">
                                {unit.description}
                              </div>
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {unit.sub_state}
                            </div>
                            <div className="flex gap-1">
                              {unit.active_state === 'active' ? (
                                <>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => systemdAction(unit.name, 'restart')}
                                  >
                                    <RefreshCw className="h-3 w-3" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => systemdAction(unit.name, 'stop')}
                                  >
                                    <Square className="h-3 w-3" />
                                  </Button>
                                </>
                              ) : (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => systemdAction(unit.name, 'start')}
                                >
                                  <Play className="h-3 w-3" />
                                </Button>
                              )}
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))
                )}
              </div>
            </div>
          )}
        </div>

        {/* Logs panel */}
        {selectedTask && (
          <div className="w-1/2 border-l flex flex-col">
            <div className="p-3 border-b flex items-center justify-between">
              <span className="font-medium text-sm">Task Logs</span>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setSelectedTask(null)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-auto p-4 bg-muted/30">
              <pre className="text-xs font-mono whitespace-pre-wrap">
                {taskLogs.length > 0 ? taskLogs.join('\n') : 'No output yet'}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
