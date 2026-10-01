import { Gauge, GitPullRequest as Github, LockKeyhole, Scale, SlidersHorizontal, Tags, UserRoundCog } from 'lucide-react'
import type { Category, Config, Privacy, Weights } from '../types'
import { useEffect, useState } from 'react'
import { api } from '../api'
import { downloadBlob } from '../download'
import { Toggle } from './Toggle'

interface Props {
  config: Config
  effective: Config | null
  onChange: (config: Config) => void
}

export function Settings({ config, effective, onChange }: Props) {
  const [message, setMessage] = useState('')
  const [aliasText, setAliasText] = useState(() => formatAliases(config.aliases))
  useEffect(() => setAliasText(formatAliases(config.aliases)), [config.aliases])
  const set = <K extends keyof Config>(key: K, value: Config[K]) => onChange({ ...config, [key]: value })
  const setWeight = (key: keyof Weights, value: number) => set('weights', { ...config.weights, [key]: value })
  const setCategoryWeight = (key: Category, value: number) => set('categoryWeights', { ...config.categoryWeights, [key]: value })
  const setPrivacy = (key: keyof Privacy, value: boolean) => set('privacy', { ...config.privacy, [key]: value })
  const weightTotal = Object.values(config.weights).reduce((sum, value) => sum + value, 0)
  return (
    <div className="settings-view">
      <header className="settings-header"><div><h2>分析设置</h2><p>配置会写入报告，并参与缓存指纹计算。</p></div><span className={Math.abs(weightTotal - 1) < 0.001 ? 'valid' : 'invalid'}>权重合计 {(weightTotal * 100).toFixed(0)}%</span></header>
      <div className="config-actions">
        <button className="button secondary" onClick={() => downloadBlob(new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' }), 'repolens-config.json')}>导出本机配置</button>
        <label className="button secondary">导入配置<input type="file" accept=".json,application/json" className="file-input" onChange={async (event) => {
          const input = event.currentTarget
          const file = input.files?.[0]
          if (!file) return
          try { if (file.size > 2 * 1024 * 1024) throw new Error('配置文件超过 2 MB'); const result = await api.validateConfig(JSON.parse(await file.text())); onChange(result.config); setMessage('配置已导入并验证') } catch (cause) { setMessage(cause instanceof Error ? cause.message : '无法导入配置') } finally { input.value = '' }
        }} /></label>
        <button className="button secondary" onClick={() => api.defaultConfig().then((defaults) => { onChange(defaults); setMessage('已恢复默认设置') }).catch((cause) => setMessage(cause.message))}>恢复默认</button>
        <span>设置自动保存在此浏览器；配置文件包含身份别名，离线报告会移除完整配置。</span>
      </div>
      {message && <p className="config-message" role="status">{message}</p>}

      <SettingsSection icon={<Scale />} title="综合贡献权重" description="每个维度仍会独立展示，权重只影响综合占比。">
        <Range label="有效工作量" value={config.weights.workload} onChange={(value) => setWeight('workload', value)} color="#008f83" />
        <Range label="当前所有权" value={config.weights.ownership} onChange={(value) => setWeight('ownership', value)} color="#3f6ed8" />
        <Range label="留存贡献" value={config.weights.retention} onChange={(value) => setWeight('retention', value)} color="#d6a11f" />
        <Range label="协作贡献" value={config.weights.collaboration} onChange={(value) => setWeight('collaboration', value)} color="#e7664c" />
      </SettingsSection>

      <SettingsSection icon={<Tags />} title="代码类型权重" description="控制不同类型变更对有效工作量的影响，不改变原始行数。">
        <Range label="源代码" value={config.categoryWeights.source} onChange={(value) => setCategoryWeight('source', value)} color="#008f83" />
        <Range label="测试" value={config.categoryWeights.test} onChange={(value) => setCategoryWeight('test', value)} color="#3f6ed8" />
        <Range label="文档" value={config.categoryWeights.docs} onChange={(value) => setCategoryWeight('docs', value)} color="#d6a11f" />
        <Range label="配置" value={config.categoryWeights.config} onChange={(value) => setCategoryWeight('config', value)} color="#e7664c" />
        <Range label="资源" value={config.categoryWeights.asset} onChange={(value) => setCategoryWeight('asset', value)} color="#8a60a8" />
        <Range label="其他" value={config.categoryWeights.other} onChange={(value) => setCategoryWeight('other', value)} color="#66717d" />
      </SettingsSection>

      <SettingsSection icon={<Gauge />} title="性能与精度" description="平衡模式适合大多数项目，完整模式可能在超大仓库上耗时较长。">
        <div className="settings-grid">
          <label className="field"><span>并行任务数</span><input type="number" min="1" max="32" value={config.parallelism} onChange={(e) => set('parallelism', Number(e.target.value))} /></label>
          <label className="field"><span>单文件上限（MB）</span><input type="number" min="1" max="100" value={Math.round(config.maxFileSizeBytes / 1024 / 1024)} onChange={(e) => set('maxFileSizeBytes', Number(e.target.value) * 1024 * 1024)} /></label>
          <label className="field"><span>归属文件上限</span><input type="number" min="1" max="1000000" value={config.maxOwnershipFiles} onChange={(e) => set('maxOwnershipFiles', Number(e.target.value))} /></label>
          <label className="field"><span>画像文件上限</span><input type="number" min="1" max="1000000" value={config.maxInventoryFiles} onChange={(e) => set('maxInventoryFiles', Number(e.target.value))} /></label>
        </div>
        {effective && <p className="effective-config">实际生效：并行 {effective.parallelism}；单文件 {(effective.maxFileSizeBytes / 1024 / 1024).toFixed(1)} MB；画像最多 {effective.maxInventoryFiles.toLocaleString()} 文件；{effective.enableOwnership ? `归属最多 ${effective.maxOwnershipFiles.toLocaleString()} 文件` : '跳过归属'}；{effective.enableRetention ? '近似留存启用' : '留存停用'}。</p>}
        <div className="toggle-list">
          <Toggle label="当前代码所有权" description="对目标版本中的文件执行并行 blame。" checked={config.enableOwnership} onChange={(value) => set('enableOwnership', value)} />
          <Toggle label="代码留存" description="结合历史新增和当前归属计算留存率。" checked={config.enableRetention} onChange={(value) => set('enableRetention', value)} />
          <Toggle label="重命名检测" description="提高移动文件的准确性，但会增加历史扫描耗时。" checked={config.detectRenames} onChange={(value) => set('detectRenames', value)} />
          <Toggle label="包含 Merge commit" description="可能与原始提交重复，默认关闭。" checked={config.includeMerges} onChange={(value) => set('includeMerges', value)} />
          <Toggle label="包含机器人" description="将依赖更新与自动化账号纳入统计。" checked={config.includeBots} onChange={(value) => set('includeBots', value)} />
        </div>
      </SettingsSection>

      <SettingsSection icon={<Github />} title="GitHub 增强分析" description="连接后可读取 Review、Issue 和协作数据，不会上传本地源代码。">
        <div className="integration-row"><div><strong>GitHub</strong><span>OAuth 接入将在下一里程碑启用</span></div><button className="button secondary" disabled>尚未连接</button></div>
        <Toggle label="协作贡献" description="有可用的 GitHub 数据时纳入综合评分。" checked={config.enableCollaboration} onChange={(value) => set('enableCollaboration', value)} />
      </SettingsSection>

      <SettingsSection icon={<SlidersHorizontal />} title="过滤规则" description="每行一个 Glob 规则，支持 *、** 和 ?。">
        <div className="textareas">
          <label><span>忽略文件</span><textarea value={config.ignoredPatterns.join('\n')} onChange={(e) => set('ignoredPatterns', lines(e.target.value))} /></label>
          <label><span>生成文件</span><textarea value={config.generatedPatterns.join('\n')} onChange={(e) => set('generatedPatterns', lines(e.target.value))} /></label>
          <label><span>第三方代码</span><textarea value={config.vendoredPatterns.join('\n')} onChange={(e) => set('vendoredPatterns', lines(e.target.value))} /></label>
          <label><span>机器人名称片段</span><textarea value={config.botPatterns.join('\n')} onChange={(e) => set('botPatterns', lines(e.target.value))} /></label>
        </div>
      </SettingsSection>

      <SettingsSection icon={<UserRoundCog />} title="贡献者身份归并" description="把历史用户名或邮箱归并到同一身份，每行使用“旧身份 = 统一身份”。">
        <label className="alias-field"><span>身份别名（离开输入框后保存）</span><textarea placeholder="old@example.com = developer@example.com" value={aliasText} onChange={(e) => setAliasText(e.target.value)} onBlur={() => {
          try { const aliases = parseAliases(aliasText); set('aliases', aliases); setMessage('') } catch (cause) { setMessage(cause instanceof Error ? cause.message : '别名格式错误') }
        }} /></label>
      </SettingsSection>

      <SettingsSection icon={<LockKeyhole />} title="默认导出隐私" description="源代码、结构化文件名、邮箱、本机路径和凭据始终不会导出。">
        <div className="toggle-list two-columns">
          <Toggle label="仓库名称" checked={config.privacy.showRepositoryName} onChange={(value) => setPrivacy('showRepositoryName', value)} />
          <Toggle label="贡献者名称" checked={config.privacy.showContributors} onChange={(value) => setPrivacy('showContributors', value)} />
          <Toggle label="提交信息" checked={config.privacy.showCommitMessages} onChange={(value) => setPrivacy('showCommitMessages', value)} />
          <Toggle label="目录名称" checked={config.privacy.showDirectories} onChange={(value) => setPrivacy('showDirectories', value)} />
          <Toggle label="提交 SHA" checked={config.privacy.showCommitSha} onChange={(value) => setPrivacy('showCommitSha', value)} />
          <Toggle label="贡献者头像" checked={config.privacy.showAvatars} onChange={(value) => setPrivacy('showAvatars', value)} />
        </div>
      </SettingsSection>
    </div>
  )
}

function SettingsSection({ icon, title, description, children }: { icon: React.ReactNode; title: string; description: string; children: React.ReactNode }) {
  return <section className="settings-section"><header><span>{icon}</span><div><h3>{title}</h3><p>{description}</p></div></header><div className="settings-content">{children}</div></section>
}

function Range({ label, value, onChange, color }: { label: string; value: number; onChange: (value: number) => void; color: string }) {
  return <label className="range-row"><span>{label}</span><input type="range" min="0" max="1" step="0.05" value={value} style={{ accentColor: color }} onChange={(e) => onChange(Number(e.target.value))} /><b>{Math.round(value * 100)}%</b></label>
}

function lines(value: string) {
  return value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
}

function formatAliases(aliases: Record<string, string>) {
  return Object.entries(aliases).sort(([left], [right]) => left.localeCompare(right)).map(([source, target]) => `${source} = ${target}`).join('\n')
}

function parseAliases(value: string) {
  const aliases: Record<string, string> = {}
  for (const line of lines(value)) {
    const separator = line.indexOf('=')
    if (separator <= 0) throw new Error('每行别名必须使用“旧身份 = 统一身份”')
    const source = line.slice(0, separator).trim().toLowerCase()
    const target = line.slice(separator + 1).trim().toLowerCase()
    if (!source || !target || aliases[source]) throw new Error('别名不能重复或为空')
    aliases[source] = target
  }
  return aliases
}
