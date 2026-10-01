import { test } from 'node:test'
import assert from 'node:assert/strict'
import { monitorJob } from './monitor.ts'
import type { Analysis, Job } from './types.ts'

const complete = { id: 'job', status: 'complete', analysisId: 'analysis' } as Job
const analysis = { id: 'analysis' } as Analysis

test('retries transient progress and result failures before marking complete', async () => {
  let requests = 0, resultRequests = 0
  const states: string[] = []
  const reports: Analysis[] = []
  await monitorJob('job', {
    job: async () => { if (++requests === 1) throw new Error('temporary'); return complete },
    analysis: async () => { if (++resultRequests === 1) throw new Error('temporary result'); return analysis },
  }, { job: (value) => states.push(value.status), complete: (value) => reports.push(value), error: () => {} }, new AbortController().signal, async () => {})
  assert.equal(requests, 3)
  assert.equal(resultRequests, 2)
  assert.deepEqual(states, ['complete'])
  assert.deepEqual(reports, [analysis])
})

test('limits retries and allows explicit reconnect', async () => {
  let requests = 0, message = ''
  await monitorJob('job', { job: async () => { requests++; throw new Error('offline') }, analysis: async () => analysis },
    { job: () => assert.fail('unexpected state'), complete: () => assert.fail('unexpected result'), error: (value) => { message = value } }, new AbortController().signal, async () => {})
  assert.equal(requests, 8)
  assert.match(message, /重新连接/)
})

test('aborted request never delivers stale analysis', async () => {
  const controller = new AbortController()
  await monitorJob('job', { job: async () => { controller.abort(); return complete }, analysis: async () => assert.fail('stale result read') },
    { job: () => assert.fail('stale job'), complete: () => assert.fail('stale result'), error: () => assert.fail('stale error') }, controller.signal, async () => {})
})
