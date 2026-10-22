import { useEffect, useState } from 'react'
import { api } from '../api'
import type { CompareResponse } from '../types'

export default function OverviewTab() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [userCount, setUserCount] = useState(0)
  const [compare, setCompare] = useState<CompareResponse | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const [users, cmp] = await Promise.all([api.users(), api.compare()])
      setUserCount(users.length)
      setCompare(cmp)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) {
    return (
      <div className="bg-white border border-slate-200 rounded-lg p-6 text-center text-slate-500">
        در حال بارگذاری...
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
        <div className="font-medium">خطا در اتصال به بک‌اند</div>
        <div className="text-sm mt-1" dir="ltr">{error}</div>
        <div className="text-xs mt-2 text-red-600">
          مطمئن شو بک‌اند Go در حال اجراست (go run . در ریشه پروژه).
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-sm text-slate-500">کاربران PostgreSQL</div>
          <div className="text-3xl font-bold text-blue-600 mt-1">{userCount}</div>
        </div>
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-sm text-slate-500">اختلاف‌ها</div>
          <div
            className={`text-3xl font-bold mt-1 ${
              compare && compare.differences_count === 0
                ? 'text-green-600'
                : 'text-red-600'
            }`}
          >
            {compare?.differences_count ?? 0}
          </div>
        </div>
        <button
          onClick={load}
          className="bg-white border border-slate-200 rounded-lg p-4 hover:bg-slate-100 transition text-slate-700 font-medium"
        >
          🔄 بروزرسانی
        </button>
      </div>
    </div>
  )
}