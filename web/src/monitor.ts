import type { Analysis, Job } from './types.ts'

interface Callbacks {
  job: (job: Job) => void
  complete: (analysis: Analysis) => void
  error: (message: string) => void
}

export async function monitorJob(
  id: string,
  service: { job: (id: string) => Promise<Job>; analysis: (id: string) => Promise<Analysis> },
  callbacks: Callbacks,
  signal: AbortSignal,
  delay: (ms: number) => Promise<void> = (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
) {
  let failures = 0
  while (!signal.aborted) {
    try {
      const next = await service.job(id)
      if (signal.aborted) return
      if (next.status === 'complete') {
        if (!next.analysisId) throw new Error('完成任务缺少分析结果编号')
        const analysis = await service.analysis(next.analysisId)
        if (signal.aborted) return
        callbacks.complete(analysis)
        callbacks.job(next)
        callbacks.error('')
        return
      }
      callbacks.job(next)
      if (next.status === 'failed') { callbacks.error(next.error || '分析失败'); return }
      if (next.status === 'cancelled') return
      failures = 0
      callbacks.error('')
    } catch (cause) {
      if (signal.aborted) return
      failures++
      const message = cause instanceof Error ? cause.message : '无法获取分析进度'
      if (failures >= 8) { callbacks.error(`${message}；重试已停止，可点击“重新连接”恢复进度。`); return }
      callbacks.error(`${message}；正在重试 (${failures}/8)`)
    }
    await delay(Math.min(550 * 2 ** failures, 5000))
  }
}
