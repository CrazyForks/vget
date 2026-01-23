/**
 * Docker API Service
 * Communicates with vget-server running in Docker for YouTube downloads
 */

const DEFAULT_DOCKER_SERVER_URL = "http://localhost:8080";
const JWT_STORAGE_KEY = "docker_server_jwt";
const JWT_EXPIRY_KEY = "docker_server_jwt_expiry";

export interface DockerServerConfig {
  url: string;
  apiKey?: string;
}

export interface DockerHealthResponse {
  code: number;
  data: {
    status: string;
    version: string;
  };
  message: string;
}

export interface DockerAuthStatusResponse {
  code: number;
  data: {
    api_key_configured: boolean;
  };
  message: string;
}

export interface DockerTokenResponse {
  code: number;
  data: {
    jwt: string;
  } | null;
  message: string;
}

export interface DockerDownloadJob {
  id: string;
  url: string;
  status: "queued" | "downloading" | "completed" | "failed" | "cancelled";
  progress: number;
  downloaded: number;
  total: number;
  filename: string;
  error: string | null;
}

export interface DockerDownloadResponse {
  code: number;
  data: {
    id: string;
    status: string;
  };
  message: string;
}

export interface DockerJobStatusResponse {
  code: number;
  data: DockerDownloadJob;
  message: string;
}

/**
 * Check if a URL is a YouTube URL
 */
export function isYouTubeUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    const host = parsed.hostname.toLowerCase();
    return (
      host === "youtube.com" ||
      host === "www.youtube.com" ||
      host === "m.youtube.com" ||
      host === "youtu.be" ||
      host === "www.youtu.be" ||
      host === "music.youtube.com"
    );
  } catch {
    return false;
  }
}

/**
 * Get the Docker server URL from localStorage or use default
 */
export function getDockerServerUrl(): string {
  const stored = localStorage.getItem("docker_server_url");
  return stored || DEFAULT_DOCKER_SERVER_URL;
}

/**
 * Set the Docker server URL
 */
export function setDockerServerUrl(url: string): void {
  localStorage.setItem("docker_server_url", url);
  // Clear cached JWT when URL changes
  localStorage.removeItem(JWT_STORAGE_KEY);
  localStorage.removeItem(JWT_EXPIRY_KEY);
}

/**
 * Get the stored JWT token
 */
function getStoredJwt(): string | null {
  const jwt = localStorage.getItem(JWT_STORAGE_KEY);
  const expiry = localStorage.getItem(JWT_EXPIRY_KEY);

  if (!jwt || !expiry) return null;

  // Check if token is expired (with 1 hour buffer)
  const expiryTime = parseInt(expiry, 10);
  if (Date.now() > expiryTime - 3600000) {
    // Token expired or expiring soon
    localStorage.removeItem(JWT_STORAGE_KEY);
    localStorage.removeItem(JWT_EXPIRY_KEY);
    return null;
  }

  return jwt;
}

/**
 * Store JWT token
 */
function storeJwt(jwt: string): void {
  localStorage.setItem(JWT_STORAGE_KEY, jwt);
  // API tokens are valid for 1 year, but we'll refresh more frequently
  // Set expiry to 30 days from now
  const expiry = Date.now() + 30 * 24 * 60 * 60 * 1000;
  localStorage.setItem(JWT_EXPIRY_KEY, expiry.toString());
}

/**
 * Get the manually configured JWT token (for settings UI)
 */
export function getDockerJwtToken(): string {
  return localStorage.getItem(JWT_STORAGE_KEY) || "";
}

/**
 * Set JWT token manually (from settings UI)
 */
export function setDockerJwtToken(token: string): void {
  if (token) {
    storeJwt(token);
  } else {
    localStorage.removeItem(JWT_STORAGE_KEY);
    localStorage.removeItem(JWT_EXPIRY_KEY);
  }
}

/**
 * Check if the Docker server requires authentication
 */
export async function checkAuthRequired(): Promise<boolean> {
  try {
    const baseUrl = getDockerServerUrl();
    const response = await fetch(`${baseUrl}/api/auth/status`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });

    if (!response.ok) return false;

    const data: DockerAuthStatusResponse = await response.json();
    return data.code === 200 && data.data.api_key_configured;
  } catch {
    return false;
  }
}

/**
 * Generate a new JWT token from the server
 */
export async function generateJwtToken(): Promise<string | null> {
  try {
    const baseUrl = getDockerServerUrl();
    const response = await fetch(`${baseUrl}/api/auth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });

    if (!response.ok) return null;

    const data: DockerTokenResponse = await response.json();
    if (data.code === 201 && data.data?.jwt) {
      storeJwt(data.data.jwt);
      return data.data.jwt;
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * Get authorization headers for API requests
 */
function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  // Use stored JWT token if available
  const jwt = getStoredJwt();
  if (jwt) {
    headers["Authorization"] = `Bearer ${jwt}`;
  }

  return headers;
}

/**
 * Check if the Docker server is running and healthy
 */
export async function checkDockerHealth(): Promise<boolean> {
  try {
    const baseUrl = getDockerServerUrl();
    const response = await fetch(`${baseUrl}/api/health`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });

    if (!response.ok) return false;

    const data: DockerHealthResponse = await response.json();
    return data.code === 200 && data.data.status === "ok";
  } catch {
    return false;
  }
}

/**
 * Start a download via the Docker server
 */
export async function startDockerDownload(
  url: string,
  filename?: string
): Promise<DockerDownloadResponse> {
  const baseUrl = getDockerServerUrl();
  const response = await fetch(`${baseUrl}/api/download`, {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify({ url, filename }),
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("Authentication required. Please configure JWT token in Settings → Sites → Docker Server.");
    }
    const errorText = await response.text();
    throw new Error(`Docker server error: ${errorText}`);
  }

  return response.json();
}

/**
 * Get the status of a download job
 */
export async function getDockerJobStatus(
  jobId: string
): Promise<DockerJobStatusResponse> {
  const baseUrl = getDockerServerUrl();
  const response = await fetch(`${baseUrl}/api/status/${jobId}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw new Error(`Failed to get job status: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get all download jobs from the Docker server
 */
export async function getDockerJobs(): Promise<DockerDownloadJob[]> {
  const baseUrl = getDockerServerUrl();
  const response = await fetch(`${baseUrl}/api/jobs`, {
    method: "GET",
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw new Error(`Failed to get jobs: ${response.statusText}`);
  }

  const data = await response.json();
  return data.data.jobs || [];
}

/**
 * Cancel a download job on the Docker server
 */
export async function cancelDockerJob(jobId: string): Promise<void> {
  const baseUrl = getDockerServerUrl();
  const response = await fetch(`${baseUrl}/api/jobs/${jobId}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw new Error(`Failed to cancel job: ${response.statusText}`);
  }
}
