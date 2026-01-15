import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Folder, Globe } from "lucide-react";

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto px-4 py-4 flex items-center gap-4">
          <Link to="/" className="p-2 rounded-lg hover:bg-muted transition-colors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-xl font-semibold">Settings</h1>
        </div>
      </header>

      {/* Settings Content */}
      <main className="container mx-auto px-4 py-8 max-w-2xl">
        {/* Output Directory */}
        <div className="space-y-6">
          <div className="flex items-start justify-between p-4 rounded-lg border border-border">
            <div className="flex items-start gap-3">
              <Folder className="h-5 w-5 text-muted-foreground mt-0.5" />
              <div>
                <h3 className="font-medium">Download Location</h3>
                <p className="text-sm text-muted-foreground mt-1">
                  ~/Downloads/vget
                </p>
              </div>
            </div>
            <button className="px-3 py-1.5 text-sm rounded-lg border border-input hover:bg-muted transition-colors">
              Change
            </button>
          </div>

          {/* Language */}
          <div className="flex items-start justify-between p-4 rounded-lg border border-border">
            <div className="flex items-start gap-3">
              <Globe className="h-5 w-5 text-muted-foreground mt-0.5" />
              <div>
                <h3 className="font-medium">Language</h3>
                <p className="text-sm text-muted-foreground mt-1">English</p>
              </div>
            </div>
            <button className="px-3 py-1.5 text-sm rounded-lg border border-input hover:bg-muted transition-colors">
              Change
            </button>
          </div>
        </div>
      </main>
    </div>
  );
}
