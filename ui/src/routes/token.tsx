import { createFileRoute } from "@tanstack/react-router";
import { TokenPage } from "../pages/TokenPage";

export const Route = createFileRoute("/token")({
  component: TokenPage,
});
