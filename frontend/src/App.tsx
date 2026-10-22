import { useState } from 'react'
import OverviewTab from './components/OverviewTab'
import UsersTab from './components/UsersTab'
import CompareTab from './components/CompareTab'
import OutboxTab from './components/OutboxTab'

type Tab = 'overview' | 'users' | 'compare' | 'outbox'

const tabs: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'نمای کلی' },
  { id: 'users', label: 'کاربران' },
  { id: 'compare', label: 'مقایسه' },
  { id: 'outbox', label: 'صف همگام‌سازی' },
]

export default function App() {
  const [tab, setTab] = useState<Tab>('overview')

  return (
    <div className="min-h-screen bg-slate-50" dir="rtl">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-6xl mx-auto px-4 pt-4">
          <h1 className="text-xl font-bold text-slate-900">
            Data Sync Dashboard
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            مقایسه و همگام‌سازی PostgreSQL و YugabyteDB
          </p>
          <nav className="flex gap-1 mt-4 -mb-px">
            {tabs.map((t) => (
              <button
                key={t.id}
                onClick={() => setTab(t.id)}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition ${
                  tab === t.id
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-slate-500 hover:text-slate-700'
                }`}
              >
                {t.label}
              </button>
            ))}
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-6">
        {tab === 'overview' && <OverviewTab />}
        {tab === 'users' && <UsersTab />}
        {tab === 'compare' && <CompareTab />}
        {tab === 'outbox' && <OutboxTab />}
      </main>
    </div>
  )
}