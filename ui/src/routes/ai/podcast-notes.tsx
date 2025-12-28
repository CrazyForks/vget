import { createFileRoute } from "@tanstack/react-router";
import { PodcastNotesPage } from "../../pages/PodcastNotesPage";

export const Route = createFileRoute("/ai/podcast-notes")({
  component: PodcastNotesPage,
});
