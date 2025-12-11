export interface UITranslations {
  download_to: string;
  edit: string;
  save: string;
  cancel: string;
  paste_url: string;
  download: string;
  adding: string;
  jobs: string;
  total: string;
  no_downloads: string;
  paste_hint: string;
  queued: string;
  downloading: string;
  completed: string;
  failed: string;
  cancelled: string;
  settings: string;
  language: string;
  format: string;
  quality: string;
  twitter_auth: string;
  server_port: string;
  max_concurrent: string;
  api_key: string;
  webdav_servers: string;
  add: string;
  delete: string;
  name: string;
  url: string;
  username: string;
  password: string;
  no_webdav_servers: string;
  clear_history: string;
  clear_all: string;
}

export interface ServerTranslations {
  no_config_warning: string;
  run_init_hint: string;
}

export const defaultTranslations: UITranslations = {
  download_to: "Download to:",
  edit: "Edit",
  save: "Save",
  cancel: "Cancel",
  paste_url: "Paste URL to download...",
  download: "Download",
  adding: "Adding...",
  jobs: "Jobs",
  total: "total",
  no_downloads: "No downloads yet",
  paste_hint: "Paste a URL above to get started",
  queued: "queued",
  downloading: "downloading",
  completed: "completed",
  failed: "failed",
  cancelled: "cancelled",
  settings: "Settings",
  language: "Language",
  format: "Format",
  quality: "Quality",
  twitter_auth: "Twitter Auth",
  server_port: "Server Port",
  max_concurrent: "Max Concurrent",
  api_key: "API Key",
  webdav_servers: "WebDAV Servers",
  add: "Add",
  delete: "Delete",
  name: "Name",
  url: "URL",
  username: "Username",
  password: "Password",
  no_webdav_servers: "No WebDAV servers configured",
  clear_history: "Clear",
  clear_all: "Clear All",
};

export const defaultServerTranslations: ServerTranslations = {
  no_config_warning: "No config file found. Using default settings.",
  run_init_hint: "Run 'vget init' to configure vget interactively.",
};
