import type { Contributor } from '../types'

const colors = ['#008f83', '#e7664c', '#d6a11f', '#3f6ed8', '#8a60a8', '#66717d']

export function DonutChart({ contributors }: { contributors: Contributor[] }) {
  const radius = 70
  const circumference = 2 * Math.PI * radius
  let offset = 0
  return (
    <div className="donut-wrap">
      <svg className="donut" width="184" height="184" viewBox="0 0 184 184" role="img" aria-label="综合贡献占比">
        <circle cx="92" cy="92" r={radius} fill="none" stroke="#e8ecee" strokeWidth="18" />
        {contributors.slice(0, 6).map((person, index) => {
          const length = circumference * person.compositeShare / 100
          const element = (
            <circle key={person.id} cx="92" cy="92" r={radius} fill="none" stroke={colors[index]}
              strokeWidth="18" strokeDasharray={`${length} ${circumference - length}`}
              strokeDashoffset={-offset} transform="rotate(-90 92 92)" />
          )
          offset += length
          return element
        })}
        <text x="92" y="86" textAnchor="middle" className="donut-value">{contributors.length}</text>
        <text x="92" y="108" textAnchor="middle" className="donut-label">位贡献者</text>
      </svg>
      <div className="donut-legend">
        {contributors.slice(0, 5).map((person, index) => (
          <div key={person.id}><i style={{ background: colors[index] }} /><span>{person.name}</span><b>{person.compositeShare.toFixed(1)}%</b></div>
        ))}
      </div>
    </div>
  )
}

