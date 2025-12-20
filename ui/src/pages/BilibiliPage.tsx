import { useState, useEffect } from "react";
import { useApp } from "../context/AppContext";
import { setConfigValue } from "../utils/apis";

interface CookieFields {
  sessdata: string;
  biliJct: string;
  dedeUserId: string;
}

function parseCookie(cookieStr: string): CookieFields {
  const fields: CookieFields = { sessdata: "", biliJct: "", dedeUserId: "" };
  if (!cookieStr) return fields;

  const parts = cookieStr.split(";").map((p) => p.trim());
  for (const part of parts) {
    const [key, ...valueParts] = part.split("=");
    const value = valueParts.join("=");
    if (key === "SESSDATA") fields.sessdata = value;
    else if (key === "bili_jct") fields.biliJct = value;
    else if (key === "DedeUserID") fields.dedeUserId = value;
  }
  return fields;
}

function buildCookie(fields: CookieFields): string {
  const parts: string[] = [];
  if (fields.sessdata) parts.push(`SESSDATA=${fields.sessdata}`);
  if (fields.biliJct) parts.push(`bili_jct=${fields.biliJct}`);
  if (fields.dedeUserId) parts.push(`DedeUserID=${fields.dedeUserId}`);
  return parts.join("; ");
}

export function BilibiliPage() {
  const { isConnected, showToast } = useApp();
  const [fields, setFields] = useState<CookieFields>({
    sessdata: "",
    biliJct: "",
    dedeUserId: "",
  });
  const [savedCookie, setSavedCookie] = useState("");
  const [saving, setSaving] = useState(false);

  // Load saved cookie on mount
  useEffect(() => {
    fetch("/api/config")
      .then((res) => res.json())
      .then((data) => {
        if (data.data?.bilibili_cookie) {
          setSavedCookie(data.data.bilibili_cookie);
          setFields(parseCookie(data.data.bilibili_cookie));
        }
      })
      .catch(() => {});
  }, []);

  const handleSave = async () => {
    const cookie = buildCookie(fields);
    if (!cookie) return;

    setSaving(true);
    try {
      await setConfigValue("bilibili.cookie", cookie);
      setSavedCookie(cookie);
      showToast("success", "登录成功！前往首页开始下载 Bilibili 视频");
    } catch (error) {
      console.error("Failed to save cookie:", error);
      showToast("error", "保存失败，请重试");
    } finally {
      setSaving(false);
    }
  };

  const handleClear = async () => {
    setSaving(true);
    try {
      await setConfigValue("bilibili.cookie", "");
      setFields({ sessdata: "", biliJct: "", dedeUserId: "" });
      setSavedCookie("");
    } catch (error) {
      console.error("Failed to clear cookie:", error);
    } finally {
      setSaving(false);
    }
  };

  const isLoggedIn = savedCookie.includes("SESSDATA");
  const hasAnyInput = fields.sessdata || fields.biliJct || fields.dedeUserId;

  return (
    <div className="max-w-3xl mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">Bilibili</h1>

      {/* Status */}
      <div className="mb-6 flex items-center gap-2">
        <span
          className={`inline-block w-2 h-2 rounded-full ${
            isLoggedIn ? "bg-green-500" : "bg-zinc-400"
          }`}
        />
        <span className="text-sm text-zinc-600 dark:text-zinc-400">
          {isLoggedIn ? "已登录" : "未登录"}
        </span>
      </div>

      {/* Instructions */}
      <div className="mb-6 p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <h3 className="text-sm font-medium mb-3">获取 Cookie 的方法</h3>
        <ol className="space-y-2 text-sm text-zinc-600 dark:text-zinc-400">
          <li>1. 在浏览器中打开 bilibili.com 并登录</li>
          <li>2. 按 F12 打开开发者工具</li>
          <li>3. 切换到「应用」(Application) 标签</li>
          <li>4. 在左侧找到 Cookies → bilibili.com</li>
          <li>5. 复制以下字段的值，粘贴到下方输入框</li>
        </ol>
      </div>

      {/* Cookie Input */}
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-4">
        <label className="block text-sm font-medium mb-2">
          Bilibili Cookie
        </label>
        <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-4">
          粘贴 Cookie 以下载会员或登录内容
        </p>

        <div className="space-y-3">
          {/* SESSDATA */}
          <div>
            <label className="block text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
              SESSDATA
            </label>
            <input
              type="text"
              value={fields.sessdata}
              onChange={(e) =>
                setFields((f) => ({ ...f, sessdata: e.target.value }))
              }
              placeholder="粘贴 SESSDATA 值"
              className="w-full px-3 py-2 text-sm bg-zinc-50 dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
              disabled={!isConnected || saving}
            />
          </div>

          {/* bili_jct */}
          <div>
            <label className="block text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
              bili_jct
            </label>
            <input
              type="text"
              value={fields.biliJct}
              onChange={(e) =>
                setFields((f) => ({ ...f, biliJct: e.target.value }))
              }
              placeholder="粘贴 bili_jct 值"
              className="w-full px-3 py-2 text-sm bg-zinc-50 dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
              disabled={!isConnected || saving}
            />
          </div>

          {/* DedeUserID */}
          <div>
            <label className="block text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
              DedeUserID
            </label>
            <input
              type="text"
              value={fields.dedeUserId}
              onChange={(e) =>
                setFields((f) => ({ ...f, dedeUserId: e.target.value }))
              }
              placeholder="粘贴 DedeUserID 值"
              className="w-full px-3 py-2 text-sm bg-zinc-50 dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
              disabled={!isConnected || saving}
            />
          </div>
        </div>

        {/* Buttons */}
        <div className="flex gap-2 mt-4">
          <button
            onClick={handleSave}
            disabled={!isConnected || saving || !hasAnyInput}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? "..." : "保存"}
          </button>
          {isLoggedIn && (
            <button
              onClick={handleClear}
              disabled={!isConnected || saving}
              className="px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 border border-red-300 dark:border-red-600 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              清除
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
