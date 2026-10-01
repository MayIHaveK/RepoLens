import { useMemo, useState } from 'react'
import type { Analysis, Category, FileMetric } from '../types'

const format = (value: number) => new Intl.NumberFormat('zh-CN').format(value)
const categories: Record<Category, string> = { source: '源代码', test: '测试', docs: '文档', config: '配置', asset: '资源', other: '其他' }
const reasons: Record<string, string> = { ignored: '忽略规则', generated: '生成文件', vendored: '第三方代码', binary: '二进制扩展名', oversized: '超出大小限制', symlink: '符号链接', 'binary-or-encoding': '二进制或非 UTF-8', 'inventory-limit': '画像文件上限' }
const kinds: Record<string, string> = { 'go-import': 'Go 外部导入', 'go-package': 'Go 包导入', import: '代码导入', 'style-import': '样式引用', 'asset-reference': '页面资源', 'config-extends': '配置继承', 'config-reference': '配置引用' }
type Tab = 'languages' | 'files' | 'directories' | 'configurations' | 'dependencies'

export function ProjectView({ analysis }: { analysis: Analysis }) {
  const project = analysis.project
  const [tab, setTab] = useState<Tab>('languages')
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [status, setStatus] = useState('')
  const [sort, setSort] = useState('lines')
  const [page, setPage] = useState(0)
  const [selected, setSelected] = useState<FileMetric | null>(null)
  const match = (value: string) => value.toLowerCase().includes(query.toLowerCase())
  const files = useMemo(() => (project?.files || []).filter((file) => file.path.toLowerCase().includes(query.toLowerCase()) && (!category || file.category === category) && (!status || file.status === status)).sort((left, right) => {
    if (sort === 'path') return left.path.localeCompare(right.path)
    const difference = sort === 'commits' ? right.commits - left.commits : sort === 'bytes' ? right.bytes - left.bytes : right.lines.total - left.lines.total
    return difference || left.path.localeCompare(right.path)
  }), [project, query, category, status, sort])
  if (!project) return <section className="data-section"><div className="empty-inline">这份旧报告没有项目画像，请重新分析仓库。新分析会统计代码、配置和文件关系。</div></section>
  const allDependencies = project.dependencies || []
  const cycles = project.cycles || []
  const directories = (project.directories || []).filter((item) => match(item.path))
  const configurations = (project.configurations || []).filter((item) => match(item.path) || (item.keys || []).some(match))
  const dependencies = allDependencies.filter((item) => match(item.from) || match(item.to))
  const languages = (project.languages || []).filter((item) => match(item.language))
  const total = tab === 'files' ? files.length : tab === 'directories' ? directories.length : tab === 'configurations' ? configurations.length : tab === 'dependencies' ? dependencies.length : languages.length
  const start = page * 100
  const fromFile = selected ? allDependencies.filter((edge) => edge.from === selected.path || edge.to === selected.path || (edge.kind === 'go-package' && (edge.from === directoryOf(selected.path) || edge.to === directoryOf(selected.path)))) : []
  const selectTab = (value: Tab) => { setTab(value); setPage(0); setSelected(null) }
  return <>
    <div className="metric-band">
      <ProjectMetric label="文本总行数" value={project.totals.total} detail={`${format(project.textFiles)} / ${format(project.totalFiles)} 个文件已读取`} />
      <ProjectMetric label="有效源代码行" value={project.sourceCodeLines} detail="源代码与测试的非空、非纯注释行" />
      <ProjectMetric label="纯注释 / 空行" value={project.totals.comment} detail={`${format(project.totals.blank)} 行空白`} />
      <ProjectMetric label="配置 / 本地关系" value={(project.configurations || []).length} detail={`${format(allDependencies.filter((item) => item.resolved).length)} 条已解析关系`} />
    </div>
    <div className="scope-banner"><strong>统计范围</strong><span>{project.scope}。历史日期范围：{analysis.config.effective?.since || '不限'} 至 {analysis.config.effective?.until || '不限'}。</span></div>
    <section className="data-section project-section">
      <div className="project-tabs" role="tablist" aria-label="项目分析视图">
        {([['languages', '语言构成'], ['files', '文件明细'], ['directories', '目录模块'], ['configurations', '配置分析'], ['dependencies', '多文件关系']] as const).map(([value, label]) => <button key={value} role="tab" aria-selected={tab === value} className={tab === value ? 'active' : ''} onClick={() => selectTab(value)}>{label}</button>)}
      </div>
      <div className="project-toolbar">
        <input type="search" aria-label="筛选项目数据" placeholder="搜索语言、路径或配置键…" value={query} onChange={(event) => { setQuery(event.target.value); setPage(0) }} />
        {tab === 'files' && <>
          <select aria-label="文件分类" value={category} onChange={(event) => { setCategory(event.target.value); setPage(0) }}><option value="">全部分类</option>{Object.entries(categories).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select>
          <select aria-label="读取状态" value={status} onChange={(event) => { setStatus(event.target.value); setPage(0) }}><option value="">全部状态</option><option value="analyzed">已读取</option><option value="skipped">已跳过</option></select>
          <select aria-label="文件排序" value={sort} onChange={(event) => { setSort(event.target.value); setPage(0) }}><option value="lines">按总行数</option><option value="commits">按变更次数</option><option value="bytes">按文件大小</option><option value="path">按路径</option></select>
        </>}
        <span>{format(total)} 项</span>
      </div>
      <div className="table-scroll">
        {tab === 'languages' && <table><thead><tr><th>语言 / 格式</th><th>文件</th><th>总行数</th><th>代码 / 内容</th><th>注释</th><th>空白</th><th>总行占比</th></tr></thead><tbody>{languages.slice(start, start + 100).map((item) => <tr key={item.language}><td><b>{item.language}</b></td><td>{format(item.files)}</td><td>{format(item.lines.total)}</td><td>{format(item.lines.code)}</td><td>{format(item.lines.comment)}</td><td>{format(item.lines.blank)}</td><td>{(project.totals.total ? item.lines.total / project.totals.total * 100 : 0).toFixed(1)}%</td></tr>)}</tbody></table>}
        {tab === 'files' && <table><thead><tr><th>文件</th><th>分类</th><th>语言</th><th>总行</th><th>代码 / 内容</th><th>大小</th><th>历史变更</th><th>读取状态</th></tr></thead><tbody>{files.slice(start, start + 100).map((file) => <tr key={file.path}><td><button className="path-link" onClick={() => setSelected(file)}>{file.path}</button></td><td>{categories[file.category] || '其他'}</td><td>{file.language}</td><td>{file.status === 'analyzed' ? format(file.lines.total) : '—'}</td><td>{file.status === 'analyzed' ? format(file.lines.code) : '—'}</td><td>{format(file.bytes)} B</td><td>{format(file.commits)} 次</td><td>{file.status === 'analyzed' ? '已读取' : reasons[file.reason || ''] || '已跳过'}</td></tr>)}</tbody></table>}
        {tab === 'directories' && <table><thead><tr><th>目录 / 模块</th><th>文件</th><th>总行数</th><th>代码 / 内容</th><th>注释</th><th>空白</th></tr></thead><tbody>{directories.slice(start, start + 100).map((item) => <tr key={item.path}><td><button className="path-link" onClick={() => { setQuery(item.path === '.' ? '' : item.path + '/'); selectTab('files') }}>{item.path === '.' ? '整个仓库' : item.path}</button></td><td>{format(item.files)}</td><td>{format(item.lines.total)}</td><td>{format(item.lines.code)}</td><td>{format(item.lines.comment)}</td><td>{format(item.lines.blank)}</td></tr>)}</tbody></table>}
        {tab === 'configurations' && <table><thead><tr><th>配置文件</th><th>格式</th><th>检查范围</th><th>依赖 / 脚本</th><th>顶层键</th><th>发现</th></tr></thead><tbody>{configurations.slice(start, start + 100).map((item) => <tr key={item.path}><td>{item.path}</td><td>{item.format}</td><td>{item.parseStatus === 'valid' ? 'JSON 语法有效' : item.parseStatus === 'invalid' ? '语法无效' : '结构检查'}</td><td>{item.dependencies} / {item.scripts}</td><td className="wrap-cell">{item.keys?.join('、') || '—'}</td><td className="wrap-cell">{item.issues?.join('；') || '未发现结构问题'}</td></tr>)}</tbody></table>}
        {tab === 'dependencies' && <table><thead><tr><th>来源文件 / 包</th><th>引用目标</th><th>关系</th><th>解析状态</th></tr></thead><tbody>{dependencies.slice(start, start + 100).map((item) => <tr key={`${item.from}:${item.to}:${item.kind}`}><td>{item.from}</td><td>{item.to}</td><td>{kinds[item.kind] || item.kind}</td><td>{item.resolved ? '仓库内已解析' : '未解析 / 外部引用'}</td></tr>)}</tbody></table>}
      </div>
      {total === 0 && <p className="empty-inline">没有符合条件的数据。</p>}
      <div className="pagination"><button className="button secondary" disabled={page === 0} onClick={() => setPage((value) => value - 1)}>上一页</button><span>{page + 1} / {Math.max(1, Math.ceil(total / 100))}</span><button className="button secondary" disabled={(page + 1) * 100 >= total} onClick={() => setPage((value) => value + 1)}>下一页</button></div>
    </section>
    {selected && <section className="data-section file-detail"><div className="section-title"><h2>{selected.path}</h2><button className="button secondary" onClick={() => setSelected(null)}>收起</button></div><p>历史新增 +{format(selected.additions)} / 删除 −{format(selected.deletions)} · {format(selected.commits)} 次变更 · 最后变更 {selected.lastChangedAt ? new Date(selected.lastChangedAt).toLocaleString('zh-CN') : '范围内无记录'}</p><p>代码 / 内容 {format(selected.lines.code)} · 注释 {format(selected.lines.comment)} · 空白 {format(selected.lines.blank)}</p><ul>{fromFile.map((edge) => <li key={`${edge.from}:${edge.to}:${edge.kind}`}>{edge.from} → {edge.to}（{kinds[edge.kind] || edge.kind}）</li>)}</ul>{!fromFile.length && <p>未识别出静态引用；这不代表运行时没有依赖。</p>}</section>}
    {cycles.length > 0 && <section className="data-section analysis-notes"><h3>循环引用组</h3><p>下列文件或包处于同一强连通组；列表不是调用顺序。</p><ul>{cycles.map((group) => <li key={group.join('|')}>{group.join(' · ')}</li>)}</ul></section>}
    <details className="data-section analysis-notes" open><summary>覆盖范围与分析说明</summary><p>当前文件跳过：{Object.entries(project.skipped).map(([reason, count]) => `${reasons[reason] || reason} ${format(count)}`).join('；') || '无'}</p><p>历史变更跳过：{Object.entries(project.historySkipped || {}).map(([reason, count]) => `${reasons[reason] || reason} ${format(count)}`).join('；') || '无'}</p><ul>{project.notes.map((note) => <li key={note}>{note}</li>)}</ul></details>
  </>
}

function ProjectMetric({ label, value, detail }: { label: string; value: number; detail: string }) {
  return <div className="metric"><div><span>{label}</span><strong>{format(value)}</strong><small>{detail}</small></div></div>
}
function directoryOf(value: string) { const index = value.lastIndexOf('/'); return index < 0 ? '.' : value.slice(0, index) }
