import { create } from "zustand";
import { invoke } from "@tauri-apps/api/core";

export type AuthStatus = "logged_out" | "checking" | "logged_in";

export interface SiteAuthStatus {
  status: AuthStatus;
  username?: string;
  avatar?: string;
}

export interface QRSession {
  url: string;
  qrcode_key: string;
}

export interface QRPollResult {
  status: number;
  status_text: string;
  username?: string;
}

// QR Status codes from Bilibili API
export const QR_WAITING = 86101;
export const QR_SCANNED = 86090;
export const QR_EXPIRED = 86038;
export const QR_CONFIRMED = 0;

interface AuthState {
  // Sidebar visibility
  isOpen: boolean;
  activeTab: "bilibili" | "xiaohongshu";

  // Site-specific auth status
  bilibili: SiteAuthStatus;
  xiaohongshu: SiteAuthStatus;

  // Actions
  open: (tab?: "bilibili" | "xiaohongshu") => void;
  close: () => void;
  setTab: (tab: "bilibili" | "xiaohongshu") => void;
  checkAuthStatus: () => Promise<void>;
  setBilibiliStatus: (status: SiteAuthStatus) => void;
  setXiaohongshuStatus: (status: SiteAuthStatus) => void;
  logout: (site: "bilibili" | "xiaohongshu") => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  isOpen: false,
  activeTab: "bilibili",

  bilibili: { status: "logged_out" },
  xiaohongshu: { status: "logged_out" },

  open: (tab) => {
    set({ isOpen: true });
    if (tab) {
      set({ activeTab: tab });
    }
    // Check auth status when opening
    get().checkAuthStatus();
  },

  close: () => set({ isOpen: false }),

  setTab: (tab) => set({ activeTab: tab }),

  checkAuthStatus: async () => {
    // Check Bilibili status
    set((state) => ({
      bilibili: { ...state.bilibili, status: "checking" },
      xiaohongshu: { ...state.xiaohongshu, status: "checking" },
    }));

    try {
      const bilibiliStatus = await invoke<SiteAuthStatus>("bilibili_check_status");
      set({ bilibili: bilibiliStatus });
    } catch {
      set({ bilibili: { status: "logged_out" } });
    }

    try {
      const xhsStatus = await invoke<SiteAuthStatus>("xhs_check_status");
      set({ xiaohongshu: xhsStatus });
    } catch {
      set({ xiaohongshu: { status: "logged_out" } });
    }
  },

  setBilibiliStatus: (status) => set({ bilibili: status }),

  setXiaohongshuStatus: (status) => set({ xiaohongshu: status }),

  logout: async (site) => {
    if (site === "bilibili") {
      await invoke("bilibili_logout");
      set({ bilibili: { status: "logged_out" } });
    } else {
      await invoke("xhs_logout");
      set({ xiaohongshu: { status: "logged_out" } });
    }
  },
}));

// Helper functions for Bilibili QR login
export async function generateBilibiliQR(): Promise<QRSession> {
  return invoke<QRSession>("bilibili_qr_generate");
}

export async function pollBilibiliQR(qrcodeKey: string): Promise<QRPollResult> {
  return invoke<QRPollResult>("bilibili_qr_poll", { qrcodeKey });
}

export async function saveBilibiliCookie(cookie: string): Promise<void> {
  return invoke("bilibili_save_cookie", { cookie });
}

// Helper for XHS login window
export async function openXhsLoginWindow(): Promise<void> {
  return invoke("xhs_open_login_window");
}
