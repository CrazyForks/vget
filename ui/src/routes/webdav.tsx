import { createFileRoute } from "@tanstack/react-router";
import { WebDAVPage } from "../pages/WebDAVPage";

export const Route = createFileRoute("/webdav")({
  component: WebDAVPage,
});
