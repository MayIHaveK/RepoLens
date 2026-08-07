import { Download, FileCode2, ShieldCheck, X } from 'lucide-react'
import { useState } from 'react'
import type { Privacy } from '../types'
import { Toggle } from './Toggle'

interface Props {
  initial: Privacy
  onClose: () => void
  onExport: (privacy: Privacy) => Promise<void>
}

export function ExportDialog({ initial, onClose, onExport }: Props) {
  const [privacy, setPrivacy] = useState(initial)
  const [busy, setBusy] = useState(false)
  const set = (key: keyof Privacy, value: boolean) => setPrivacy((current) => ({ ...current, [key]: value }))
  const submit = async () => {
    setBusy(true)
    try { await onExport(privacy); onClose() } finally { setBusy(false) }
  }
  return (
    <div className="modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="export-title">
        <header><div className="modal-icon"><FileCode2 /></div><div><h2 id="export-title">导出离线报告</h2><p>生成包含交互图表的单文件 HTML。</p></div><button className="icon-button" title="关闭" onClick={onClose}><X /></button></header>
        <div className="privacy-banner"><ShieldCheck /><span><strong>固定保护项</strong>源代码、文件名、邮箱、本机路径和凭据不会写入报告。</span></div>
        <div className="modal-options">
          <Toggle label="显示仓库名称" checked={privacy.showRepositoryName} onChange={(value) => set('showRepositoryName', value)} />
          <Toggle label="显示贡献者名称" checked={privacy.showContributors} onChange={(value) => set('showContributors', value)} />
          <Toggle label="显示提交信息" checked={privacy.showCommitMessages} onChange={(value) => set('showCommitMessages', value)} />
          <Toggle label="显示目录名称" checked={privacy.showDirectories} onChange={(value) => set('showDirectories', value)} />
          <Toggle label="显示 Commit SHA" checked={privacy.showCommitSha} onChange={(value) => set('showCommitSha', value)} />
          <Toggle label="显示贡献者头像" checked={privacy.showAvatars} onChange={(value) => set('showAvatars', value)} />
        </div>
        <footer><button className="button secondary" onClick={onClose}>取消</button><button className="button primary" disabled={busy} onClick={submit}><Download />{busy ? '正在生成' : '导出 HTML'}</button></footer>
      </div>
    </div>
  )
}

