import { useApp } from "../context/AppContext";

export function WebDAVPage() {
  const { t } = useApp();

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">{t.webdav_browser}</h1>

      <div className="bg-zinc-100 dark:bg-zinc-800 rounded-lg p-8 text-center">
        <p className="text-zinc-500 dark:text-zinc-400">{t.coming_soon}</p>
      </div>
    </div>
  );
}
