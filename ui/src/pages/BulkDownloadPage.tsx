import { useApp } from "../context/AppContext";

export function BulkDownloadPage() {
  const { t } = useApp();

  return (
    <div className="flex flex-col items-center justify-center h-full">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-zinc-800 dark:text-zinc-100 mb-2">
          {t.bulk_download}
        </h1>
        <p className="text-zinc-500 dark:text-zinc-400">{t.coming_soon}</p>
      </div>
    </div>
  );
}
