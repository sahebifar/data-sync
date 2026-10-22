import { useEffect, useState } from 'react'
import { api } from '../api'
import type { CompareResponse } from '../types'

const statusInfo: Record<string, { label: string; color: string }> = {
  missing_in_yugabyte: { label: 'فقط در PostgreSQL', color: 'bg-orange-100 text-orange-800' },
  missing_in_postgres: { label: 'فقط در YugabyteDB', color: 'bg-purple-100 text-purple-800' },
  data_mismatch: { label: 'داده ناسازگار', color: 'bg-red-100 text-red-800' },
}

export default function CompareTab() {
  const [data, setData] = useState<CompareResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const compare = async () => {
    setLoading(true)
    setError(null)
    setActionMsg(null)
    try {
      setData(await api.compare())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const syncMissing = async () => {
    setActionMsg(null)
    try {
      const r = await api.syncMissing()
      setActionMsg(`درج شد: ${r.inserted}، رد شد: ${r.skipped}، خطا: ${r.failed}`)
      await compare()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const repair = async (deleteExtra: boolean) => {
    setActionMsg(null)
    try {
      const r = await api.repair(deleteExtra)
      setActionMsg(
        `درج: ${r.inserted}، بروزرسانی: ${r.updated}، بدون تغییر: ${r.unchanged}، حذف: ${r.deleted}، خطا: ${r.failed}`,
      )
      await compare()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  useEffect(() => {
    compare()
  }, [])

  return (
    <div className="space-y-4">
      <div className="bg-white border border-slate-200 rounded-lg p-4 flex flex-wrap gap-2">
        <button
          onClick={compare}
          disabled={loading}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition disabled:opacity-50 text-sm font-medium"
        >
          {loading ? 'در حال مقایسه...' : 'مقایسه'}
        </button>
        <button
          onClick={syncMissing}
          className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition text-sm font-medium"
        >
          انتقال کاربران مفقود
        </button>
        <button
          onClick={() => repair(false)}
          className="bg-orange-600 text-white px-4 py-2 rounded-lg hover:bg-orange-700 transition text-sm font-medium"
        >
          تعمیر (بدون حذف)
        </button>
        <button
          onClick={() => repair(true)}
          className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition text-sm font-medium"
        >
          تعمیر کامل (با حذف)
        </button>
      </div>

      {actionMsg && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-3 text-green-800 text-sm">
          {actionMsg}
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm" dir="ltr">
          {error}
        </div>
      )}

      {data && (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <div className="px-4 py-3 border-b border-slate-200">
            <h2 className="font-medium text-slate-900">
              اختلاف‌ها ({data.differences_count})
            </h2>
          </div>

          {data.differences.length === 0 ? (
            <div className="p-6 text-center text-green-600 font-medium">
              ✅ دیتابیس‌ها کاملاً یکسان هستند.
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {data.differences.map((d, i) => {
                const info = statusInfo[d.status] ?? {
                  label: d.status,
                  color: 'bg-slate-100 text-slate-800',
                }
                return (
                  <div key={i} className="px-4 py-3">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`text-xs font-medium px-2 py-1 rounded ${info.color}`}>
                        {info.label}
                      </span>
                      <span className="font-mono text-sm" dir="ltr">{d.email}</span>
                    </div>

                    {d.different_fields && (
                      <div className="text-xs text-slate-500 mt-2">
                        فیلدهای متفاوت: {d.different_fields.join(', ')}
                      </div>
                    )}

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-2 mt-2 text-xs">
                      {d.postgres_user && (
                        <div className="bg-blue-50 rounded p-2">
                          <div className="font-medium text-blue-700 mb-1">PostgreSQL</div>
                          <div dir="ltr">
                            {d.postgres_user.first_name} {d.postgres_user.last_name} ({d.postgres_user.age})
                          </div>
                        </div>
                      )}
                      {d.yugabyte_user && (
                        <div className="bg-purple-50 rounded p-2">
                          <div className="font-medium text-purple-700 mb-1">YugabyteDB</div>
                          <div dir="ltr">
                            {d.yugabyte_user.first_name} {d.yugabyte_user.last_name} ({d.yugabyte_user.age})
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}