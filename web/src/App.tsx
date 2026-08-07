import { useEffect, useState } from 'react'
import { BarChart3, Clock3, Download, FlaskConical, FolderGit2, GitPullRequest as Github, LayoutDashboard, Play, RotateCcw, Settings as SettingsIcon, SlidersHorizontal, Users, XCircle } from 'lucide-react'
import { api } from './api'
import { Contributors, Overview, Trends } from './components/Dashboard'
import { ExportDialog } from './components/ExportDialog'
import { Settings } from './components/Settings'
import { demoAnalysis } from './demo'
import type { Analysis, Config, Job } from './types'

type View = 'overview' | 'contributors' | 'trends' | 'settings'

export default function App() {
  const [config, setConfig] = useState<Config | null>(null)
  const [analysis, setAnalysis] = useState<Analysis | null>(null)
  const [recent, setRecent] = useState<Analysis[]>([])
  const [repositoryPath, setRepositoryPath] = useState(() => localStorage.getItem('repolens.repository') || '')
  const [view, setView] = useState<View>('overview')
  const [job, setJob] = useState<Job | null>(null)
  const [error, setError] = useState('')
  const [exportOpen, setExportOpen] = useState(false)

  useEffect(() => {
    Promise.all([api.defaultConfig(), api.recent()])
      .then(([defaults, history]) => {
        setConfig(defaults)
        setRecent(history)
        if (history.length) setAnalysis(history[0])
      })
      .catch((cause) => setError(cause instanceof Error ? cause.message : '无法连接本地服务'))
  }, [])

  useEffect(() => {
    if (!job || job.status === 'complete' || job.status === 'failed') return
    const timer = window.setTimeout(async () => {
      try {
        const next = await api.job(job.id)
        setJob(next)
        if (next.status === 'complete' && next.analysisId) {
          const result = await api.analysis(next.analysisId)
          setAnalysis(result)
          setView('overview')
          setRecent((items) => [result, ...items.filter((item) => item.id !== result.id)].slice(0, 8))
        }
        if (next.status === 'failed') setError(next.error || '分析失败')
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : '无法获取分析进度')
      }
    }, 550)
    return () => window.clearTimeout(timer)
  }, [job])

  const startAnalysis = async () => {
    if (!config || !repositoryPath.trim()) return
    setError('')
    localStorage.setItem('repolens.repository', repositoryPath)
    try {
      const created = await api.createJob(repositoryPath.trim(), config)
      setJob(created)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '无法创建分析任务')
    }
  }

  const showDemo = () => {
    setAnalysis(demoAnalysis)
    setJob(null)
    setError('')
    setView('overview')
  }

  if (!config) {
    return <div className="boot"><div className="brand-mark">R</div><strong>RepoLens</strong><span>{error || '正在启动本地分析服务'}</span></div>
  }

  const running = job?.status === 'queued' || job?.status === 'running'
  const isDemo = analysis?.id === demoAnalysis.id

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand"><span className="brand-mark">R</span><span>Repo<b>Lens</b></span></div>
        <nav className="main-tabs" aria-label="主视图">
          <Tab icon={<LayoutDashboard />} label="概览" active={view === 'overview'} onClick={() => setView('overview')} disabled={!analysis} />
          <Tab icon={<Users />} label="贡献者" active={view === 'contributors'} onClick={() => setView('contributors')} disabled={!analysis} />
          <Tab icon={<BarChart3 />} label="趋势" active={view === 'trends'} onClick={() => setView('trends')} disabled={!analysis} />
          <Tab icon={<SettingsIcon />} label="设置" active={view === 'settings'} onClick={() => setView('settings')} />
        </nav>
        <div className="top-actions">
          <span className="local-status"><i />仅本机</span>
          <button className="icon-button" title="GitHub 连接状态" disabled><Github /></button>
          <button className="button secondary export-button" disabled={!analysis || isDemo} onClick={() => setExportOpen(true)}><Download />导出</button>
        </div>
      </header>

      <aside className="sidebar">
        <section className="scan-panel">
          <div className="panel-label"><FolderGit2 /><span>本地仓库</span></div>
          <label className="path-field"><input value={repositoryPath} onChange={(event) => setRepositoryPath(event.target.value)} placeholder="C:\Projects\repository" aria-label="仓库路径" /></label>
          <div className="two-fields">
            <label><span>目标分支</span><input value={config.ref} onChange={(event) => setConfig({ ...config, ref: event.target.value })} /></label>
            <label><span>开始时间</span><input type="date" value={config.since || ''} onChange={(event) => setConfig({ ...config, since: event.target.value })} /></label>
          </div>
          <span className="field-label">性能配置</span>
          <div className="segmented">
            {(['fast', 'balanced', 'thorough'] as const).map((profile) => <button key={profile} className={config.profile === profile ? 'active' : ''} onClick={() => setConfig({ ...config, profile })}>{profile === 'fast' ? '快速' : profile === 'balanced' ? '平衡' : '完整'}</button>)}
          </div>
          <button className="button primary analyze-button" disabled={running || !repositoryPath.trim()} onClick={startAnalysis}><Play />{running ? '正在分析' : '开始分析'}</button>
          {running && job && <div className="job-progress"><div><span>{job.progress.message}</span><b>{job.progress.percent.toFixed(0)}%</b></div><span className="progress-track"><i style={{ width: `${job.progress.percent}%` }} /></span></div>}
          {error && <div className="error-message"><XCircle /><span>{error}</span></div>}
        </section>

        <section className="recent-panel">
          <header><span>最近分析</span>{recent.length > 0 && <button title="刷新列表" className="icon-button small" onClick={() => api.recent().then(setRecent)}><RotateCcw /></button>}</header>
          {recent.length ? recent.map((item) => <button key={item.id} className={`recent-item ${analysis?.id === item.id ? 'active' : ''}`} onClick={() => { setAnalysis(item); setView('overview') }}><span className="repo-icon">{item.repository.name.slice(0, 1).toUpperCase()}</span><span><b>{item.repository.name}</b><small>{item.repository.ref} · {new Date(item.generatedAt).toLocaleDateString('zh-CN')}</small></span></button>) : <p className="recent-empty">完成一次分析后会显示在这里。</p>}
        </section>

        <button className="sidebar-settings" onClick={() => setView('settings')}><SlidersHorizontal /><span><b>分析与隐私设置</b><small>权重、过滤和性能</small></span></button>
      </aside>

      <main className="workspace">
        {view === 'settings' ? <Settings config={config} onChange={setConfig} /> : analysis ? (
          <>
            <header className="workspace-header">
              <div><div className="scope-row"><h1>{analysis.repository.name}</h1>{isDemo && <span className="demo-badge">演示数据</span>}<span className="mode-badge">{analysis.mode === 'git-only' ? 'Git-only' : 'GitHub enhanced'}</span></div><p>{analysis.repository.ref} · {analysis.repository.commitSha.slice(0, 10)} · {analysis.summary.files.toLocaleString()} 个文件</p></div>
              <div className="header-facts"><span><Clock3 />{new Date(analysis.generatedAt).toLocaleString('zh-CN')}</span><span><FlaskConical />配置 {analysis.config.fingerprint.slice(0, 8)}</span></div>
            </header>
            {view === 'overview' && <Overview analysis={analysis} />}
            {view === 'contributors' && <Contributors analysis={analysis} />}
            {view === 'trends' && <Trends analysis={analysis} />}
          </>
        ) : <EmptyState onDemo={showDemo} />}
      </main>

      {exportOpen && analysis && <ExportDialog initial={config.privacy} onClose={() => setExportOpen(false)} onExport={(privacy) => api.export(analysis.id, privacy)} />}
    </div>
  )
}

function Tab({ icon, label, active, onClick, disabled }: { icon: React.ReactNode; label: string; active: boolean; onClick: () => void; disabled?: boolean }) {
  return <button className={active ? 'active' : ''} onClick={onClick} disabled={disabled}>{icon}{label}</button>
}

function EmptyState({ onDemo }: { onDemo: () => void }) {
  return <div className="empty-state"><div className="empty-visual"><span /><span /><span /><BarChart3 /></div><h1>选择一个 Git 仓库开始分析</h1><p>RepoLens 在本机读取提交历史，不会上传源代码。大型仓库首次分析可能需要一些时间，后续会直接使用缓存。</p><button className="button secondary" onClick={onDemo}><FlaskConical />查看演示报告</button></div>
}
