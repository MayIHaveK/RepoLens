import { AlertTriangle, ArrowDown, ArrowUp, Code2, GitCommitHorizontal, ShieldCheck, Users } from 'lucide-react'
import type { Analysis, Category } from '../types'
import { DonutChart } from './DonutChart'
import { TimelineChart } from './TimelineChart'

const format = (value: number) => new Intl.NumberFormat('zh-CN').format(value)
const categoryLabels: Record<Category, string> = { source: '源代码', test: '测试', docs: '文档', config: '配置', asset: '资源', other: '其他' }
const categoryColors: Record<Category, string> = { source: '#008f83', test: '#3f6ed8', docs: '#d6a11f', config: '#e7664c', asset: '#8a60a8', other: '#66717d' }

export function Overview({ analysis }: { analysis: Analysis }) {
  const maximum = Math.max(...analysis.categories.map((item) => item.additions), 1)
  return (
    <>
      <div className="metric-band">
        <Metric icon={<Users />} label="贡献者" value={format(analysis.summary.contributors)} detail={`${format(analysis.summary.commits)} 次有效提交`} />
        <Metric icon={<ArrowUp />} label="有效新增" value={`+${format(analysis.summary.additions)}`} detail={`删除 ${format(analysis.summary.deletions)} 行`} tone="green" />
        <Metric icon={<Code2 />} label="当前归属" value={format(analysis.summary.ownedLines)} detail={`覆盖 ${analysis.summary.coveragePercent.toFixed(1)}% 文件`} tone="blue" />
        <Metric icon={<ShieldCheck />} label="分析耗时" value={duration(analysis.durationMs)} detail={analysis.config.profile === 'balanced' ? '平衡模式' : analysis.config.profile} tone="yellow" />
      </div>

      {analysis.project && <div className="metric-band">
        <Metric icon={<Code2 />} label="目标版本文本总行" value={format(analysis.project.totals.total)} detail={`${format(analysis.project.textFiles)} 个已读取文本文件`} />
        <Metric icon={<Code2 />} label="有效源代码行" value={format(analysis.project.sourceCodeLines)} detail="源代码与测试，排除纯注释和空行" tone="green" />
        <Metric icon={<Code2 />} label="注释 / 空白" value={format(analysis.project.totals.comment)} detail={`${format(analysis.project.totals.blank)} 行空白`} tone="blue" />
        <Metric icon={<ShieldCheck />} label="配置文件" value={format(analysis.project.configurations.length)} detail="在项目分析中查看结构与关系" tone="yellow" />
      </div>}

      {analysis.warnings && analysis.warnings.length > 0 && (
        <div className="warning-strip"><AlertTriangle size={17} /><ul>{analysis.warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul></div>
      )}

      <div className="dashboard-grid">
        <section className="data-section contribution-section">
          <div className="section-title"><div><h2>综合贡献分布</h2><p>根据有可用数据的维度归一化</p></div><span className="section-note">合计 {analysis.contributors.reduce((sum, item) => sum + item.compositeShare, 0).toFixed(1)}%</span></div>
          <DonutChart contributors={analysis.contributors} />
        </section>
        <section className="data-section category-section">
          <div className="section-title"><div><h2>代码类型构成</h2><p>文件数取当前已读取文本；新增取筛选后的历史</p></div></div>
          <div className="category-list">
            {analysis.categories.map((category) => (
              <div className="category-row" key={category.category}>
                <div><i style={{ background: categoryColors[category.category] }} /><span>{categoryLabels[category.category]}</span><b>{format(category.files)} 文件</b></div>
                <div className="category-track"><span style={{ width: `${category.additions / maximum * 100}%`, background: categoryColors[category.category] }} /></div>
                <strong>+{format(category.additions)}</strong>
              </div>
            ))}
          </div>
        </section>
      </div>

      <section className="data-section table-section">
        <div className="section-title"><div><h2>贡献者排名</h2><p>各维度独立展示，综合占比不替代原始数据</p></div><span className="section-note">按综合占比排序</span></div>
        <ContributorTable analysis={analysis} limit={6} />
      </section>
      <details className="data-section analysis-notes"><summary>分析配置与可复现信息</summary><p>程序 {analysis.programVersion || '旧版未记录'} · 算法 {analysis.algorithmVersion || '旧版未记录'} · 配置 {analysis.config.fingerprint}</p>{analysis.config.effectiveWeights && <p>实际评分权重：工作量 {(analysis.config.effectiveWeights.workload * 100).toFixed(1)}% · 所有权 {(analysis.config.effectiveWeights.ownership * 100).toFixed(1)}% · 留存 {(analysis.config.effectiveWeights.retention * 100).toFixed(1)}% · 协作 {(analysis.config.effectiveWeights.collaboration * 100).toFixed(1)}%</p>}{analysis.config.effective && <pre>{JSON.stringify(analysis.config.effective, null, 2)}</pre>}</details>
    </>
  )
}

export function Contributors({ analysis }: { analysis: Analysis }) {
  return (
    <section className="data-section table-section full-view">
      <div className="section-title"><div><h2>全部贡献者</h2><p>{analysis.repository.ref} 分支 · {format(analysis.summary.commits)} 次提交</p></div></div>
      <ContributorTable analysis={analysis} />
    </section>
  )
}

export function Trends({ analysis }: { analysis: Analysis }) {
  return (
    <>
      <section className="data-section full-view">
        <div className="section-title"><div><h2>贡献趋势</h2><p>按月聚合的有效新增代码</p></div><span className="section-note">{analysis.timeline.length} 个周期</span></div>
        <TimelineChart data={analysis.timeline} />
      </section>
      <section className="data-section table-section">
        <div className="section-title"><div><h2>月度明细</h2><p>提交活动与代码变化</p></div></div>
        <table><thead><tr><th>月份</th><th>提交</th><th>新增</th><th>删除</th><th>净变化</th></tr></thead><tbody>
          {[...analysis.timeline].reverse().map((item) => <tr key={item.period}><td><b>{item.period}</b></td><td>{format(item.commits)}</td><td className="positive">+{format(item.additions)}</td><td className="negative">-{format(item.deletions)}</td><td>{format(item.additions - item.deletions)}</td></tr>)}
        </tbody></table>
      </section>
    </>
  )
}

function ContributorTable({ analysis, limit }: { analysis: Analysis; limit?: number }) {
  return (
    <div className="table-scroll"><table><thead><tr><th>贡献者</th><th>综合占比</th><th>工作量</th><th>当前归属</th><th>留存率</th><th>提交</th><th>代码变化</th></tr></thead><tbody>
      {analysis.contributors.slice(0, limit).map((person, index) => (
        <tr key={person.id}>
          <td><div className="person-cell"><span className={`rank rank-${Math.min(index + 1, 4)}`}>{index + 1}</span><span><b>{person.name}</b><small>{person.firstCommitAt ? new Date(person.firstCommitAt).toLocaleDateString('zh-CN') + ' 加入' : '历史贡献者'}</small></span></div></td>
          <td><div className="share-cell"><span className="share-track"><i style={{ width: `${person.compositeShare}%` }} /></span><b>{person.compositeShare.toFixed(1)}%</b></div></td>
          <td>{person.workloadShare.toFixed(1)}%</td><td>{analysis.config.ownershipEnabled ? person.ownershipShare.toFixed(1) + '%' : '—'}</td><td>{analysis.config.effective?.enableRetention === false ? '—' : person.retentionRate.toFixed(1) + '%'}</td>
          <td><span className="icon-number"><GitCommitHorizontal size={14} />{format(person.commits)}</span></td>
          <td><span className="positive"><ArrowUp size={13} />{format(person.additions)}</span><span className="negative"><ArrowDown size={13} />{format(person.deletions)}</span></td>
        </tr>
      ))}
    </tbody></table></div>
  )
}

function Metric({ icon, label, value, detail, tone = 'neutral' }: { icon: React.ReactNode; label: string; value: string; detail: string; tone?: string }) {
  return <div className={`metric metric-${tone}`}><div className="metric-icon">{icon}</div><div><span>{label}</span><strong>{value}</strong><small>{detail}</small></div></div>
}

function duration(milliseconds: number) {
  if (milliseconds < 1000) return `${milliseconds} ms`
  if (milliseconds < 60000) return `${(milliseconds / 1000).toFixed(1)} 秒`
  return `${(milliseconds / 60000).toFixed(1)} 分`
}

