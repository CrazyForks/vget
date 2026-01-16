import { createFileRoute } from "@tanstack/react-router";
import { PDFToolsPage } from "@/components/pdf-tools/PDFToolsPage";

export const Route = createFileRoute("/pdf-tools")({
  component: PDFToolsPage,
});
