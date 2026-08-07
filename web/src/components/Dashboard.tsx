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

      {analysis.warnings && analysis.warnings.length > 0 && (
        <div className="warning-strip"><AlertTriangle size={17} /><span>{analysis.warnings[0]}</span></div>
      )}

      <div className="dashboard-grid">
        <section className="data-section contribution-section">
          <div className="section-title"><div><h2>综合贡献分布</h2><p>根据当前权重归一化计算</p></div><span className="section-note">总计 100%</span></div>
          <DonutChart contributors={analysis.contributors} />
        </section>
        <section className="data-section category-section">
          <div className="section-title"><div><h2>代码类型构成</h2><p>已过滤生成、依赖和二进制文件</p></div></div>
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
          <td>{person.workloadShare.toFixed(1)}%</td><td>{person.ownershipShare.toFixed(1)}%</td><td>{person.retentionRate.toFixed(1)}%</td>
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

