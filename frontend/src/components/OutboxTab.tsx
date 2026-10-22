import { useEffect, useState } from 'react'
import { api } from '../api'
import type { OutboxItem } from '../types'

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  processing: 'bg-blue-100 text-blue-800',
  completed: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
}

const statusLabels: Record<string, string> = {
  pending: 'در انتظار',
  processing: 'در حال پردازش',
  completed: 'تکمیل شده',
  failed: 'ناموفق',
}

export default function OutboxTab() {
  const [items, setItems] = useState<OutboxItem[]>([])
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const r = await api.outbox(status || undefined)
      setItems(r.data)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const retry = async (id: number) => {
    try {
      await api.retryOutbox(id)
      await load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  useEffect(() => {
    load()
  }, [status])

  return (
    <div className="space-y-4">
      <div className="bg-white border border-slate-200 rounded-lg p-4 flex flex-wrap items-center gap-2">
        <label className="text-sm text-slate-600">فیلتر وضعیت:</label>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="border border-slate-300 rounded px-3 py-1 text-sm"
        >
          <option value="">همه</option>
          <option value="pending">در انتظار</option>
          <option value="processing">در حال پردازش</option>
          <option value="completed">تکمیل شده</option>
          <option value="failed">ناموفق</option>
        </select>
        <button
          onClick={load}
          className="bg-blue-600 text-white px-4 py-1 rounded text-sm hover:bg-blue-700"
        >
          بروزرسانی
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm" dir="ltr">
          {error}
        </div>
      )}

      <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-200">
          <h2 className="font-medium text-slate-900">
            صف همگام‌سازی ({items.length})
          </h2>
        </div>

        {loading ? (
          <div className="p-6 text-center text-slate-500">در حال بارگذاری...</div>
        ) : items.length === 0 ? (
          <div className="p-6 text-center text-slate-500">هیچ رکوردی وجود ندارد.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-slate-600">
                <tr>
                  <th className="px-3 py-2 text-right">ID</th>
                  <th className="px-3 py-2 text-right">عملیات</th>
                  <th className="px-3 py-2 text-right">ایمیل</th>
                  <th className="px-3 py-2 text-right">وضعیت</th>
                  <th className="px-3 py-2 text-right">تلاش‌ها</th>
                  <th className="px-3 py-2 text-right">آخرین خطا</th>
                  <th className="px-3 py-2 text-right"></th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.id} className="border-t border-slate-100 hover:bg-slate-50">
                    <td className="px-3 py-2 text-slate-500">{it.id}</td>
                    <td className="px-3 py-2">{it.operation}</td>
                    <td className="px-3 py-2 font-mono text-xs" dir="ltr">{it.entity_key}</td>
                    <td className="px-3 py-2">
                      <span className={`text-xs px-2 py-1 rounded ${statusColors[it.status] ?? 'bg-slate-100'}`}>
                        {statusLabels[it.status] ?? it.status}
                      </span>
                    </td>
                    <td className="px-3 py-2">{it.attempts}</td>
                    <td className="px-3 py-2 text-xs text-red-600 max-w-xs truncate" dir="ltr" title={it.last_error ?? ''}>
                      {it.last_error ?? '—'}
                    </td>
                    <td className="px-3 py-2">
                      {it.status === 'failed' && (
                        <button
                          onClick={() => retry(it.id)}
                          className="text-xs text-blue-600 hover:underline"
                        >
                          تلاش مجدد
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}