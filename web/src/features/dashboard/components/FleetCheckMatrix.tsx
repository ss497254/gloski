import { devicePath } from '@/app/navigation'
import { StatusDot, toneSurface } from '@/shared/components'
import { statusTone } from '@/shared/lib/status'
import { cn, formatDateTime } from '@/shared/lib/utils'
import type { CheckBlock, CheckMatrix, CheckMatrixRow } from '@/shared/tower'
import { Card } from '@/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/ui/table'
import { useState } from 'react'
import { Link } from 'react-router-dom'

/** One coloured strip per group; a block is one check on one device. */
export function FleetCheckMatrix({ matrix }: { matrix: CheckMatrix }) {
  const [open, setOpen] = useState<string | null>(null)
  if (matrix.rows.length === 0) return null

  const key = (row: CheckMatrixRow, block: CheckBlock) => `${row.device_id}:${block.check_id}`

  return (
    <Card className="mt-5 gap-0 overflow-hidden py-0">
      <header className="border-b border-border px-4 py-3">
        <h2 className="text-sm font-semibold text-foreground">Fleet checks</h2>
        <p className="mt-0.5 text-[11px] text-muted-foreground">
          Red first. Grey means skipped, stale or never run — not a failure. Click a block for details.
        </p>
      </header>
      <div className="overflow-x-auto">
        <Table className="text-xs">
          <TableHeader className="bg-muted/40 hover:bg-muted/40">
            <TableRow>
              <TableHead className="min-w-36 px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                Device
              </TableHead>
              {matrix.groups.map((group) => (
                <TableHead key={group} className="px-3 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                  {group}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className="divide-y divide-border/60">
            {matrix.rows.map((row) => {
              const blocks = row.cells.flatMap((cell) => cell.checks)
              return (
                <FragmentRow
                  key={row.device_id}
                  row={row}
                  blocks={blocks}
                  open={open}
                  keyFor={key}
                  onToggle={(next) => setOpen((current) => (current === next ? null : next))}
                  colSpan={matrix.groups.length + 1}
                />
              )
            })}
          </TableBody>
        </Table>
      </div>
    </Card>
  )
}

function FragmentRow({
  row,
  blocks,
  open,
  keyFor,
  onToggle,
  colSpan,
}: {
  row: CheckMatrixRow
  blocks: CheckBlock[]
  open: string | null
  keyFor: (row: CheckMatrixRow, block: CheckBlock) => string
  onToggle: (key: string) => void
  colSpan: number
}) {
  return (
    <>
      <TableRow className="hover:bg-muted/40">
        <TableCell className="px-4 py-2.5">
          <Link to={devicePath(row.device_id)} className="flex items-center gap-2 hover:text-primary">
            <StatusDot status={row.device_status} className="size-1.5" />
            <span className="truncate font-medium">{row.device_name}</span>
          </Link>
        </TableCell>
        {row.cells.map((cell) => (
          <TableCell key={cell.group} className="px-3 py-2.5">
            {cell.checks.length === 0 ? (
              <span className="text-muted-foreground/40">—</span>
            ) : (
              <div className="flex flex-wrap gap-1">
                {cell.checks.map((block) => {
                  const blockKey = keyFor(row, block)
                  const tone = block.stale ? 'muted' : statusTone(block.status)
                  return (
                    <button
                      key={blockKey}
                      type="button"
                      title={`${block.check_id}: ${block.summary}${block.stale ? ' (stale)' : ''}`}
                      aria-label={block.check_id}
                      aria-expanded={open === blockKey}
                      onClick={() => onToggle(blockKey)}
                      className={cn(
                        'size-3 rounded-sm transition-transform hover:scale-125',
                        toneSurface[tone],
                        block.stale && 'opacity-50',
                        open === blockKey && 'ring-1 ring-foreground',
                      )}
                    />
                  )
                })}
              </div>
            )}
          </TableCell>
        ))}
      </TableRow>
      {open && blocks.some((block) => keyFor(row, block) === open) && (
        <TableRow className="bg-muted/20">
          <TableCell colSpan={colSpan} className="px-4 py-3">
            <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {blocks.map((block) => (
                <div key={keyFor(row, block)} className="flex items-start gap-2 text-xs">
                  <StatusDot status={block.stale ? 'unknown' : block.status} className="mt-1 size-1.5" />
                  <div className="min-w-0">
                    <p className="font-mono text-foreground/90">
                      {block.check_id}
                      {block.stale && <span className="ml-1 text-muted-foreground">stale</span>}
                    </p>
                    <p className="text-muted-foreground">{block.summary}</p>
                    {block.details?.map((detail) => (
                      <p key={detail.key} className="text-muted-foreground/80">
                        {detail.key}: {detail.value}
                      </p>
                    ))}
                    {block.status !== 'unknown' && (
                      <p className="text-[10px] text-muted-foreground/60">ran {formatDateTime(block.ran_at)}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  )
}
