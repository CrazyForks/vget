use crate::config::{get_config, save_config, BilibiliConfig};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::Manager;

// ============ TYPES ============

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SiteAuthStatus {
    pub status: String, // "logged_out", "checking", "logged_in"
    pub username: Option<String>,
    pub avatar: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QRSession {
    pub url: String,
    pub qrcode_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QRPollResult {
    pub status: i32,
    pub status_text: String,
    pub username: Option<String>,
}

// ============ HELPER FUNCTIONS ============

fn config_dir() -> PathBuf {
    dirs::home_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join(".config")
        .join("vget")
}

fn xhs_cookies_path() -> PathBuf {
    config_dir().join("xhs_cookies.json")
}

// ============ BILIBILI AUTH ============

#[tauri::command]
pub async fn bilibili_check_status() -> Result<SiteAuthStatus, String> {
    let config = tauri::async_runtime::spawn_blocking(|| {
        get_config().map_err(|e| e.to_string())
    })
    .await
    .map_err(|e| e.to_string())??;

    let cookie = config.bilibili.cookie.unwrap_or_default();
    if cookie.is_empty() {
        return Ok(SiteAuthStatus {
            status: "logged_out".to_string(),
            username: None,
            avatar: None,
        });
    }

    // Try to get user info from Bilibili API
    match fetch_bilibili_user_info(&cookie).await {
        Ok((username, avatar)) => Ok(SiteAuthStatus {
            status: "logged_in".to_string(),
            username: Some(username),
            avatar,
        }),
        Err(_) => {
            // Cookie might be invalid, but we still have it
            Ok(SiteAuthStatus {
                status: "logged_in".to_string(),
                username: None,
                avatar: None,
            })
        }
    }
}

async fn fetch_bilibili_user_info(cookie: &str) -> Result<(String, Option<String>), String> {
    let client = reqwest::Client::new();
    let resp = client
        .get("https://api.bilibili.com/x/web-interface/nav")
        .header("Cookie", cookie)
        .header(
            "User-Agent",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        )
        .send()
        .await
        .map_err(|e| e.to_string())?;

    let json: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;

    if json["code"].as_i64() == Some(0) {
        let data = &json["data"];
        let username = data["uname"].as_str().unwrap_or("User").to_string();
        let avatar = data["face"].as_str().map(|s| s.to_string());
        Ok((username, avatar))
    } else {
        Err("Not logged in".to_string())
    }
}

#[tauri::command]
pub async fn bilibili_qr_generate() -> Result<QRSession, String> {
    let client = reqwest::Client::new();
    let resp = client
        .get("https://passport.bilibili.com/x/passport-login/web/qrcode/generate")
        .header(
            "User-Agent",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        )
        .send()
        .await
        .map_err(|e| e.to_string())?;

    let json: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;

    if json["code"].as_i64() != Some(0) {
        return Err(json["message"]
            .as_str()
            .unwrap_or("Failed to generate QR code")
            .to_string());
    }

    let data = &json["data"];
    Ok(QRSession {
        url: data["url"].as_str().unwrap_or("").to_string(),
        qrcode_key: data["qrcode_key"].as_str().unwrap_or("").to_string(),
    })
}

#[tauri::command]
pub async fn bilibili_qr_poll(qrcode_key: String) -> Result<QRPollResult, String> {
    let client = reqwest::Client::new();
    let resp = client
        .get("https://passport.bilibili.com/x/passport-login/web/qrcode/poll")
        .query(&[("qrcode_key", &qrcode_key)])
        .header(
            "User-Agent",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        )
        .send()
        .await
        .map_err(|e| e.to_string())?;

    // Get cookies from response before parsing JSON
    let cookies: Vec<String> = resp
        .headers()
        .get_all("set-cookie")
        .iter()
        .filter_map(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .collect();

    let json: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;

    if json["code"].as_i64() != Some(0) {
        return Err(json["message"]
            .as_str()
            .unwrap_or("Failed to poll QR code")
            .to_string());
    }

    let data = &json["data"];
    let status = data["code"].as_i64().unwrap_or(-1) as i32;

    // Status codes:
    // 86101 - waiting for scan
    // 86090 - scanned, waiting for confirmation
    // 86038 - expired
    // 0 - confirmed/success

    let mut username = None;

    // If login confirmed, save cookies
    if status == 0 && !cookies.is_empty() {
        let cookie_str = extract_bilibili_cookies(&cookies);
        if !cookie_str.is_empty() {
            // Save cookie to config
            if let Ok(mut config) = get_config() {
                config.bilibili = BilibiliConfig {
                    cookie: Some(cookie_str.clone()),
                };
                let _ = save_config(&config);
            }

            // Try to get username
            if let Ok((name, _)) = fetch_bilibili_user_info(&cookie_str).await {
                username = Some(name);
            }
        }
    }

    let status_text = match status {
        86101 => "Waiting for scan",
        86090 => "Scanned, confirm on phone",
        86038 => "QR code expired",
        0 => "Login successful",
        _ => "Unknown status",
    };

    Ok(QRPollResult {
        status,
        status_text: status_text.to_string(),
        username,
    })
}

fn extract_bilibili_cookies(set_cookie_headers: &[String]) -> String {
    let mut sessdata = String::new();
    let mut bili_jct = String::new();
    let mut dede_user_id = String::new();

    for cookie in set_cookie_headers {
        let parts: Vec<&str> = cookie.split(';').collect();
        if let Some(kv) = parts.first() {
            let kv_parts: Vec<&str> = kv.splitn(2, '=').collect();
            if kv_parts.len() == 2 {
                let key = kv_parts[0].trim();
                let value = kv_parts[1].trim();
                match key {
                    "SESSDATA" => sessdata = value.to_string(),
                    "bili_jct" => bili_jct = value.to_string(),
                    "DedeUserID" => dede_user_id = value.to_string(),
                    _ => {}
                }
            }
        }
    }

    let mut parts = Vec::new();
    if !sessdata.is_empty() {
        parts.push(format!("SESSDATA={}", sessdata));
    }
    if !bili_jct.is_empty() {
        parts.push(format!("bili_jct={}", bili_jct));
    }
    if !dede_user_id.is_empty() {
        parts.push(format!("DedeUserID={}", dede_user_id));
    }
    parts.join("; ")
}

#[tauri::command]
pub async fn bilibili_save_cookie(cookie: String) -> Result<(), String> {
    tauri::async_runtime::spawn_blocking(move || {
        let mut config = get_config().map_err(|e| e.to_string())?;
        config.bilibili = BilibiliConfig {
            cookie: if cookie.is_empty() {
                None
            } else {
                Some(cookie)
            },
        };
        save_config(&config).map_err(|e| e.to_string())
    })
    .await
    .map_err(|e| e.to_string())?
}

#[tauri::command]
pub async fn bilibili_logout() -> Result<(), String> {
    tauri::async_runtime::spawn_blocking(|| {
        let mut config = get_config().map_err(|e| e.to_string())?;
        config.bilibili = BilibiliConfig { cookie: None };
        save_config(&config).map_err(|e| e.to_string())
    })
    .await
    .map_err(|e| e.to_string())?
}

// ============ XIAOHONGSHU AUTH ============

#[tauri::command]
pub async fn xhs_check_status() -> Result<SiteAuthStatus, String> {
    let cookies_path = xhs_cookies_path();

    if !cookies_path.exists() {
        return Ok(SiteAuthStatus {
            status: "logged_out".to_string(),
            username: None,
            avatar: None,
        });
    }

    // Check if cookies file has content
    match fs::read_to_string(&cookies_path) {
        Ok(content) => {
            if content.trim().is_empty() || content == "[]" {
                return Ok(SiteAuthStatus {
                    status: "logged_out".to_string(),
                    username: None,
                    avatar: None,
                });
            }
            // Has cookies, assume logged in
            Ok(SiteAuthStatus {
                status: "logged_in".to_string(),
                username: None, // XHS doesn't easily expose username
                avatar: None,
            })
        }
        Err(_) => Ok(SiteAuthStatus {
            status: "logged_out".to_string(),
            username: None,
            avatar: None,
        }),
    }
}

#[tauri::command]
pub async fn xhs_logout() -> Result<(), String> {
    let cookies_path = xhs_cookies_path();

    if cookies_path.exists() {
        fs::remove_file(&cookies_path).map_err(|e| e.to_string())?;
    }

    // Also clear browser user data if exists
    let browser_data = config_dir().join("browser");
    if browser_data.exists() {
        // Only remove XHS-related data, not the entire browser folder
        // This is safer than removing everything
        let _ = fs::remove_dir_all(browser_data.join("Default").join("Cookies"));
    }

    Ok(())
}

#[tauri::command]
pub async fn xhs_open_login_window(app: tauri::AppHandle) -> Result<(), String> {
    use tauri::WebviewUrl;
    use tauri::WebviewWindowBuilder;

    // Check if window already exists
    if let Some(window) = app.get_webview_window("xhs-login") {
        window.set_focus().map_err(|e| e.to_string())?;
        return Ok(());
    }

    // Create new webview window
    let window = WebviewWindowBuilder::new(
        &app,
        "xhs-login",
        WebviewUrl::External("https://www.xiaohongshu.com/explore".parse().unwrap()),
    )
    .title("Login to Xiaohongshu")
    .inner_size(450.0, 700.0)
    .center()
    .build()
    .map_err(|e| e.to_string())?;

    // Monitor for login by checking cookies periodically
    let app_handle = app.clone();
    let window_label = window.label().to_string();

    tauri::async_runtime::spawn(async move {
        use tokio::time::{sleep, Duration};

        loop {
            sleep(Duration::from_secs(3)).await;

            // Check if window still exists
            if app_handle.get_webview_window(&window_label).is_none() {
                break;
            }

            // Try to get cookies from the webview
            // Note: This is a simplified check - in production you'd use more robust cookie extraction
            if let Some(win) = app_handle.get_webview_window(&window_label) {
                // Execute JS to check if user is logged in
                let result = win.eval(
                    r#"
                    (function() {
                        // Check if user menu or profile element exists (indicates logged in)
                        const loggedInIndicator = document.querySelector('.user-info, .login-btn[style*="display: none"], .user-avatar');
                        if (loggedInIndicator) {
                            window.__VGET_LOGGED_IN__ = true;
                        }
                        // Also check localStorage for user data
                        const userData = localStorage.getItem('user') || localStorage.getItem('userInfo');
                        if (userData) {
                            window.__VGET_LOGGED_IN__ = true;
                        }
                    })();
                    "#
                );

                if result.is_ok() {
                    // Give it a moment for the JS to execute
                    sleep(Duration::from_millis(500)).await;

                    // For now, just keep window open and let user close it manually
                    // A more robust solution would use tauri's webview cookie API when available
                }
            }
        }
    });

    Ok(())
}
