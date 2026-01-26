import { useState, useEffect, useCallback } from "react";
import clsx from "clsx";
import prettyBytes from "pretty-bytes";
import { useApp } from "../context/AppContext";
import {
  fetchHistory,
  deleteHistoryRecord,
  clearAllHistory,
  type HistoryRecord,
  type HistoryStats,
} from "../utils/apis";
import {
  FaCheck,
  FaXmark,
  FaTrash,
  FaChevronLeft,
  FaChevronRight,
} from "react-icons/fa6";

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}

function formatDate(unixTimestamp: number): string {
  const date = new Date(unixTimestamp * 1000);
  return date.toLocaleString();
}

function extractFilename(record: HistoryRecord): string {
  if (record.filename) {
    // Get just the filename from path
    const parts = record.filename.split("/");
    return parts[parts.length - 1];
  }
  // Fallback to URL
  try {
    const url = new URL(record.url);
    return url.pathname.split("/").pop() || record.url;
  } catch {
    return record.url;
  }
}

export function HistoryPage() {
  const { isConnected, t } = useApp();

  const [records, setRecords] = useState<HistoryRecord[]>([]);
  const [stats, setStats] = useState<HistoryStats | null>(null);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [clearing, setClearing] = useState(false);

  const limit = 20;
  const totalPages = Math.ceil(total / limit);

  const loadHistory = useCallback(async () => {
    if (!isConnected) return;
    setLoading(true);
    try {
      const res = await fetchHistory(limit, page * limit);
      if (res.code === 200) {
        setRecords(res.data.records || []);
        setTotal(res.data.total);
        setStats(res.data.stats);
      }
    } catch (err) {
      console.error("Failed to load history:", err);
    } finally {
      setLoading(false);
    }
  }, [isConnected, page]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  const handleDelete = async (id: string) => {
    setDeleting(id);
    try {
      const res = await deleteHistoryRecord(id);
      if (res.code === 200) {
        loadHistory();
      }
    } catch (err) {
      console.error("Failed to delete record:", err);
    } finally {
      setDeleting(null);
    }
  };

  const handleClearAll = async () => {
    if (!confirm("Clear all download history?")) return;
    setClearing(true);
    try {
      const res = await clearAllHistory();
      if (res.code === 200) {
        setPage(0);
        loadHistory();
      }
    } catch (err) {
      console.error("Failed to clear history:", err);
    } finally {
      setClearing(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto flex flex-col gap-4">
      <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
        <h1 className="text-lg font-semibold text-zinc-900 dark:text-white">
          {t.history_title}
        </h1>

        {stats && (
          <div className="flex gap-4 text-sm text-zinc-500 dark:text-zinc-400">
            <span className="flex items-center gap-1">
              <FaCheck className="text-green-500" />
              {stats.completed}
            </span>
            <span className="flex items-center gap-1">
              <FaXmark className="text-red-500" />
              {stats.failed}
            </span>
            <span>{prettyBytes(stats.total_bytes)}</span>
          </div>
        )}

        <div className="ml-auto">
          <button
            onClick={handleClearAll}
            disabled={!isConnected || total === 0 || clearing}
            className="px-3 py-1.5 border border-zinc-300 dark:border-zinc-700 rounded text-xs text-zinc-500 hover:border-red-500 hover:text-red-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {clearing ? "..." : t.history_clear_all}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-zinc-400 dark:text-zinc-600">
          {t.loading}
        </div>
      ) : records.length === 0 ? (
        <div className="text-center py-12 text-zinc-400 dark:text-zinc-600">
          <p>{t.history_empty}</p>
          <p className="text-sm mt-2">{t.history_empty_hint}</p>
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-2">
            {records.map((record) => (
              <div
                key={record.id}
                className={clsx(
                  "flex items-center gap-3 px-4 py-3 rounded-lg border transition-colors",
                  record.status === "completed"
                    ? "border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900"
                    : "border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/30",
                )}
              >
                <div className="shrink-0">
                  {record.status === "completed" ? (
                    <FaCheck className="text-green-500" />
                  ) : (
                    <FaXmark className="text-red-500" />
                  )}
                </div>

                <div className="flex-1 min-w-0">
                  <div className="truncate text-sm font-medium text-zinc-900 dark:text-white">
                    {extractFilename(record)}
                  </div>
                  <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                    <span>{formatDate(record.completed_at)}</span>
                    {record.size_bytes > 0 && (
                      <span>{prettyBytes(record.size_bytes)}</span>
                    )}
                    <span>{formatDuration(record.duration_seconds)}</span>
                  </div>
                  {record.error && (
                    <div className="text-xs text-red-500 mt-1 truncate">
                      {record.error}
                    </div>
                  )}
                </div>

                <button
                  onClick={() => handleDelete(record.id)}
                  disabled={deleting === record.id}
                  className="shrink-0 p-2 text-zinc-400 hover:text-red-500 disabled:opacity-50 transition-colors"
                  title={t.delete}
                >
                  <FaTrash className="text-sm cursor-pointer" />
                </button>
              </div>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-4 mt-4">
              <button
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
                className="p-2 text-zinc-500 hover:text-zinc-900 dark:hover:text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <FaChevronLeft />
              </button>
              <span className="text-sm text-zinc-500 dark:text-zinc-400">
                {page + 1} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                disabled={page >= totalPages - 1}
                className="p-2 text-zinc-500 hover:text-zinc-900 dark:hover:text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <FaChevronRight />
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
