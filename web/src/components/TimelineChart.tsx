import type { Analysis } from '../types'

export function TimelineChart({ data }: { data: Analysis['timeline'] }) {
  if (!data.length) return <div className="empty-inline">当前范围没有时间序列数据</div>
  const width = 760
  const height = 220
  const padX = 34
  const padY = 24
  const max = Math.max(...data.map((item) => item.additions), 1)
  const point = (value: number, index: number) => {
    const x = padX + (data.length === 1 ? 0 : index / (data.length - 1)) * (width - padX * 2)
    const y = height - padY - value / max * (height - padY * 2)
    return [x, y]
  }
  const line = data.map((item, index) => point(item.additions, index).join(',')).join(' ')
  return (
    <div className="timeline-chart">
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="每月新增代码趋势">
        {[0, 1, 2, 3].map((value) => <line key={value} x1={padX} x2={width - padX} y1={padY + value * 52} y2={padY + value * 52} stroke="#e6eaed" />)}
        <polyline points={line} fill="none" stroke="#008f83" strokeWidth="3" strokeLinejoin="round" strokeLinecap="round" />
        {data.map((item, index) => {
          const [x, y] = point(item.additions, index)
          return <circle key={item.period} cx={x} cy={y} r="4" fill="#fff" stroke="#008f83" strokeWidth="3"><title>{item.period}: +{item.additions.toLocaleString()}</title></circle>
        })}
      </svg>
      <div className="timeline-labels">{data.map((item) => <span key={item.period}>{item.period.slice(2)}</span>)}</div>
    </div>
  )
}

