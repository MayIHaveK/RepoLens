import type { Analysis, Config, Job, Privacy } from './types'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options)
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(payload.error || response.statusText)
  }
  return response.json() as Promise<T>
}

export const api = {
  defaultConfig: () => request<Config>('/api/config/default'),
  recent: () => request<Analysis[]>('/api/analyses'),
  createJob: (repositoryPath: string, config: Config) =>
    request<Job>('/api/jobs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repositoryPath, config }),
    }),
  job: (id: string) => request<Job>(`/api/jobs/${id}`),
  analysis: (id: string) => request<Analysis>(`/api/analyses/${id}`),
  export: async (analysisId: string, privacy: Privacy) => {
    const response = await fetch('/api/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ analysisId, privacy }),
    })
    if (!response.ok) throw new Error('无法生成离线报告')
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'repolens-report.html'
    anchor.click()
    URL.revokeObjectURL(url)
  },
}

