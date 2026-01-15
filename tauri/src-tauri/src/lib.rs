mod config;
mod extractor;

use config::{get_config as load_config, save_config as store_config, Config};
use extractor::{extract_media as do_extract, MediaInfo};
use tauri::Emitter;

// ============ CONFIG COMMANDS ============

#[tauri::command]
fn get_config() -> Result<Config, String> {
    load_config().map_err(|e| e.to_string())
}

#[tauri::command]
fn save_config(config: Config) -> Result<(), String> {
    store_config(&config).map_err(|e| e.to_string())
}

// ============ EXTRACTOR COMMANDS ============

#[tauri::command]
async fn extract_media(url: String) -> Result<MediaInfo, String> {
    do_extract(&url).await.map_err(|e| e.to_string())
}

// ============ DOWNLOAD COMMANDS ============

#[tauri::command]
async fn start_download(
    url: String,
    output_path: String,
    format_id: Option<String>,
    window: tauri::Window,
) -> Result<String, String> {
    let job_id = uuid::Uuid::new_v4().to_string();

    // TODO: Implement download with progress events
    let _ = window.emit("download-progress", serde_json::json!({
        "jobId": job_id,
        "progress": 0,
        "total": 100,
        "speed": 0
    }));

    Ok(job_id)
}

#[tauri::command]
fn cancel_download(_job_id: String) -> Result<(), String> {
    // TODO: Implement cancel
    Ok(())
}

// ============ TAURI SETUP ============

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .invoke_handler(tauri::generate_handler![
            // Config
            get_config,
            save_config,
            // Extractor
            extract_media,
            // Download
            start_download,
            cancel_download,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
