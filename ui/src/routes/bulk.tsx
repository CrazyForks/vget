import { createFileRoute } from "@tanstack/react-router";
import { BulkDownloadPage } from "../pages/BulkDownloadPage";

export const Route = createFileRoute("/bulk")({
  component: BulkDownloadPage,
});
