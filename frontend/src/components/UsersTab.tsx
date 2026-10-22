import { useEffect, useState } from 'react'
import { api } from '../api'
import type { User } from '../types'

type Source = 'postgres' | 'yugabyte'

interface FormState {
  id: number | null
  first_name: string
  last_name: string
  email: string
  age: string
}

const emptyForm: FormState = {
  id: null,
  first_name: '',
  last_name: '',
  email: '',
  age: '',
}

export default function UsersTab() {
  const [source, setSource] = useState<Source>('postgres')
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [showForm, setShowForm] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const data =
        source === 'postgres'
          ? await api.users()
          : await api.usersYugabyte()
      setUsers(data)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [source])

  const openCreate = () => {
    setForm(emptyForm)
    setShowForm(true)
    setMessage(null)
  }

  const openEdit = (u: User) => {
    setForm({
      id: u.id,
      first_name: u.first_name,
      last_name: u.last_name,
      email: u.email,
      age: String(u.age),
    })
    setShowForm(true)
    setMessage(null)
  }

  const closeForm = () => {
    setShowForm(false)
    setForm(emptyForm)
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    setMessage(null)
    try {
      const age = Number(form.age)
      if (!Number.isFinite(age) || age < 1 || age > 120) {
        throw new Error('سن باید بین ۱ تا ۱۲۰ باشد')
      }
      if (form.id !== null) {
        await api.updateUser(form.id, {
          first_name: form.first_name,
          last_name: form.last_name,
          age,
        })
        setMessage('کاربر بروزرسانی شد و در صف همگام‌سازی قرار گرفت.')
      } else {
        await api.createUser({
          first_name: form.first_name,
          last_name: form.last_name,
          email: form.email,
          age,
        })
        setMessage('کاربر ساخته شد و در صف همگام‌سازی قرار گرفت.')
      }
      closeForm()
      await load()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  const remove = async (u: User) => {
    if (!confirm(`کاربر ${u.email} حذف شود؟`)) return
    setError(null)
    setMessage(null)
    try {
      await api.deleteUser(u.id)
      setMessage('کاربر حذف شد و در صف همگام‌سازی قرار گرفت.')
      await load()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const isPostgres = source === 'postgres'

  return (
    <div className="space-y-4">
      <div className="bg-white border border-slate-200 rounded-lg p-3 flex items-center gap-2">
        <span className="text-sm text-slate-600">نمایش از:</span>
        <button
          onClick={() => setSource('postgres')}
          className={`px-3 py-1 text-sm rounded transition ${
            isPostgres
              ? 'bg-blue-600 text-white'
              : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
          }`}
        >
          PostgreSQL
        </button>
        <button
          onClick={() => setSource('yugabyte')}
          className={`px-3 py-1 text-sm rounded transition ${
            !isPostgres
              ? 'bg-purple-600 text-white'
              : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
          }`}
        >
          YugabyteDB
        </button>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="font-medium text-slate-900">
          کاربران {isPostgres ? 'PostgreSQL' : 'YugabyteDB'} ({users.length})
        </h2>
        <div className="flex gap-2">
          <button
            onClick={load}
            className="text-sm text-slate-600 hover:underline"
          >
            بروزرسانی
          </button>
          {isPostgres && (
            <button
              onClick={openCreate}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition text-sm font-medium"
            >
              + افزودن کاربر
            </button>
          )}
        </div>
      </div>

      {!isPostgres && (
        <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 text-purple-800 text-xs">
          YugabyteDB یک replica است. تغییرات فقط از طریق PostgreSQL و صف همگام‌سازی اعمال می‌شوند.
        </div>
      )}

      {message && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-3 text-green-800 text-sm">
          {message}
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm" dir="ltr">
          {error}
        </div>
      )}

      {showForm && isPostgres && (
        <form
          onSubmit={submit}
          className="bg-white border border-slate-200 rounded-lg p-4 space-y-3"
        >
          <h3 className="font-medium text-slate-900">
            {form.id !== null ? 'ویرایش کاربر' : 'کاربر جدید'}
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-slate-500 block mb-1">نام</label>
              <input
                required
                maxLength={100}
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                className="border border-slate-300 rounded px-3 py-2 w-full text-sm"
              />
            </div>
            <div>
              <label className="text-xs text-slate-500 block mb-1">نام خانوادگی</label>
              <input
                required
                maxLength={100}
                value={form.last_name}
                onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                className="border border-slate-300 rounded px-3 py-2 w-full text-sm"
              />
            </div>
            {form.id === null && (
              <div className="md:col-span-2">
                <label className="text-xs text-slate-500 block mb-1">ایمیل</label>
                <input
                  required
                  type="email"
                  dir="ltr"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  className="border border-slate-300 rounded px-3 py-2 w-full text-sm font-mono"
                />
              </div>
            )}
            <div>
              <label className="text-xs text-slate-500 block mb-1">سن</label>
              <input
                required
                type="number"
                min={1}
                max={120}
                value={form.age}
                onChange={(e) => setForm({ ...form, age: e.target.value })}
                className="border border-slate-300 rounded px-3 py-2 w-full text-sm"
              />
            </div>
          </div>
          <div className="flex gap-2 pt-2">
            <button
              type="submit"
              disabled={submitting}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition text-sm font-medium disabled:opacity-50"
            >
              {submitting ? 'در حال ذخیره...' : 'ذخیره'}
            </button>
            <button
              type="button"
              onClick={closeForm}
              className="border border-slate-300 text-slate-700 px-4 py-2 rounded-lg hover:bg-slate-50 transition text-sm"
            >
              انصراف
            </button>
          </div>
        </form>
      )}

      <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
        {loading ? (
          <div className="p-6 text-center text-slate-500">در حال بارگذاری...</div>
        ) : users.length === 0 ? (
          <div className="p-6 text-center text-slate-500">هیچ کاربری وجود ندارد.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-slate-600">
                <tr>
                  <th className="px-4 py-2 text-right">ID</th>
                  <th className="px-4 py-2 text-right">نام</th>
                  <th className="px-4 py-2 text-right">نام خانوادگی</th>
                  <th className="px-4 py-2 text-right">ایمیل</th>
                  <th className="px-4 py-2 text-right">سن</th>
                  {isPostgres && (
                    <th className="px-4 py-2 text-right">عملیات</th>
                  )}
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} className="border-t border-slate-100 hover:bg-slate-50">
                    <td className="px-4 py-2 text-slate-500">{u.id}</td>
                    <td className="px-4 py-2">{u.first_name}</td>
                    <td className="px-4 py-2">{u.last_name}</td>
                    <td className="px-4 py-2 font-mono text-xs" dir="ltr">{u.email}</td>
                    <td className="px-4 py-2">{u.age}</td>
                    {isPostgres && (
                      <td className="px-4 py-2">
                        <div className="flex gap-2">
                          <button
                            onClick={() => openEdit(u)}
                            className="text-xs text-blue-600 hover:underline"
                          >
                            ویرایش
                          </button>
                          <button
                            onClick={() => remove(u)}
                            className="text-xs text-red-600 hover:underline"
                          >
                            حذف
                          </button>
                        </div>
                      </td>
                    )}
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