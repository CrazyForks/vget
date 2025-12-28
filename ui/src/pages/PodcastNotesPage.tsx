import { useApp } from "../context/AppContext";

export function PodcastNotesPage() {
  const { t } = useApp();

  return (
    <div className="max-w-3xl mx-auto flex flex-col gap-4">
      <h1 className="text-xl font-medium text-zinc-900 dark:text-white">
        {t.ai_podcast_notes}
      </h1>
      <p className="text-zinc-500 dark:text-zinc-400">
        {t.coming_soon}
      </p>
    </div>
  );
}
