use super::{DownloadProgress, DownloadStatus};
use futures::StreamExt;
use reqwest::Client;
use std::collections::HashMap;
use std::path::Path;
use std::time::Instant;
use tauri::{Emitter, Window};
use tokio::fs::File;
use tokio::io::AsyncWriteExt;
use tokio::sync::watch::Receiver;

pub struct SimpleDownloader {
    client: Client,
}

impl SimpleDownloader {
    pub fn new() -> Self {
        Self {
            client: Client::builder()
                .user_agent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
                .build()
                .unwrap_or_default(),
        }
    }

    pub async fn download(
        &self,
        job_id: &str,
        url: &str,
        output_path: &str,
        window: &Window,
        cancel_rx: Receiver<bool>,
        headers: Option<HashMap<String, String>>,
    ) -> Result<(), String> {
        // Ensure parent directory exists
        if let Some(parent) = Path::new(output_path).parent() {
            tokio::fs::create_dir_all(parent)
                .await
                .map_err(|e| format!("Failed to create directory: {}", e))?;
        }

        // Start download with optional headers
        let mut request = self.client.get(url);

        if let Some(hdrs) = headers {
            for (key, value) in hdrs {
                request = request.header(&key, &value);
            }
        }

        let response = request
            .send()
            .await
            .map_err(|e| format!("Failed to fetch: {}", e))?;

        if !response.status().is_success() {
            return Err(format!("HTTP error: {}", response.status()));
        }

        let total = response.content_length();
        let mut downloaded: u64 = 0;
        let mut last_emit = Instant::now();
        let mut last_downloaded: u64 = 0;

        // Create file
        let mut file = File::create(output_path)
            .await
            .map_err(|e| format!("Failed to create file: {}", e))?;

        // Stream download
        let mut stream = response.bytes_stream();

        while let Some(chunk_result) = stream.next().await {
            // Check for cancellation
            if *cancel_rx.borrow() {
                drop(file);
                let _ = tokio::fs::remove_file(output_path).await;
                return Err("Download cancelled".to_string());
            }

            let chunk = chunk_result.map_err(|e| format!("Stream error: {}", e))?;
            downloaded += chunk.len() as u64;

            file.write_all(&chunk)
                .await
                .map_err(|e| format!("Write error: {}", e))?;

            // Emit progress every 100ms
            if last_emit.elapsed().as_millis() >= 100 {
                let elapsed = last_emit.elapsed().as_secs_f64();
                let speed = if elapsed > 0.0 {
                    ((downloaded - last_downloaded) as f64 / elapsed) as u64
                } else {
                    0
                };

                let percent = total.map(|t| (downloaded as f64 / t as f64) * 100.0).unwrap_or(0.0);

                let progress = DownloadProgress {
                    job_id: job_id.to_string(),
                    downloaded,
                    total,
                    speed,
                    percent,
                };

                let _ = window.emit("download-progress", &progress);

                last_emit = Instant::now();
                last_downloaded = downloaded;
            }
        }

        file.flush()
            .await
            .map_err(|e| format!("Flush error: {}", e))?;

        // Emit completion
        let progress = DownloadProgress {
            job_id: job_id.to_string(),
            downloaded,
            total,
            speed: 0,
            percent: 100.0,
        };

        let _ = window.emit("download-progress", &progress);
        let _ = window.emit(
            "download-complete",
            serde_json::json!({
                "jobId": job_id,
                "status": DownloadStatus::Completed,
                "outputPath": output_path,
            }),
        );

        Ok(())
    }

    /// Download video and audio separately, then merge with ffmpeg
    pub async fn download_and_merge(
        &self,
        job_id: &str,
        video_url: &str,
        audio_url: &str,
        output_path: &str,
        window: &Window,
        cancel_rx: Receiver<bool>,
        headers: Option<HashMap<String, String>>,
    ) -> Result<(), String> {
        use crate::ffmpeg::merge_video_audio;

        // Create temp file paths
        let output = Path::new(output_path);
        let parent = output.parent().unwrap_or(Path::new("."));
        let stem = output.file_stem().and_then(|s| s.to_str()).unwrap_or("video");

        let video_temp = parent.join(format!(".{}_video.m4s", stem));
        let audio_temp = parent.join(format!(".{}_audio.m4s", stem));

        let video_temp_str = video_temp.to_string_lossy().to_string();
        let audio_temp_str = audio_temp.to_string_lossy().to_string();

        // Download video (0-45% of progress)
        eprintln!("[download] Downloading video from: {}", video_url);
        self.download_file_with_progress(
            video_url,
            &video_temp_str,
            headers.clone(),
            job_id,
            window,
            0.0,  // start percent
            45.0, // end percent
            "downloading video",
        )
        .await
        .map_err(|e| {
            eprintln!("[download] Video download failed: {}", e);
            e
        })?;
        eprintln!("[download] Video downloaded to: {}", video_temp_str);

        // Check cancellation
        if *cancel_rx.borrow() {
            let _ = tokio::fs::remove_file(&video_temp).await;
            return Err("Download cancelled".to_string());
        }

        // Download audio (45-90% of progress)
        eprintln!("[download] Downloading audio from: {}", audio_url);
        self.download_file_with_progress(
            audio_url,
            &audio_temp_str,
            headers,
            job_id,
            window,
            45.0, // start percent
            90.0, // end percent
            "downloading audio",
        )
        .await
        .map_err(|e| {
            eprintln!("[download] Audio download failed: {}", e);
            // Clean up video temp file
            let _ = std::fs::remove_file(&video_temp);
            e
        })?;
        eprintln!("[download] Audio downloaded to: {}", audio_temp_str);

        // Check cancellation
        if *cancel_rx.borrow() {
            let _ = tokio::fs::remove_file(&video_temp).await;
            let _ = tokio::fs::remove_file(&audio_temp).await;
            return Err("Download cancelled".to_string());
        }

        // Merge with ffmpeg
        let _ = window.emit(
            "download-progress",
            serde_json::json!({
                "job_id": job_id,
                "downloaded": 0,
                "total": null,
                "speed": 0,
                "percent": 90.0,
                "stage": "merging video and audio"
            }),
        );

        eprintln!("[download] Merging video and audio to: {}", output_path);
        merge_video_audio(&video_temp_str, &audio_temp_str, output_path, true)
            .await
            .map_err(|e| {
                eprintln!("[download] Merge failed: {}", e);
                format!("Failed to merge: {}", e)
            })?;
        eprintln!("[download] Merge completed successfully");

        // Emit completion
        let _ = window.emit(
            "download-progress",
            serde_json::json!({
                "job_id": job_id,
                "downloaded": 0,
                "total": null,
                "speed": 0,
                "percent": 100.0,
                "stage": "completed"
            }),
        );

        let _ = window.emit(
            "download-complete",
            serde_json::json!({
                "jobId": job_id,
                "status": "completed",
                "outputPath": output_path,
            }),
        );

        Ok(())
    }

    /// Download a file with streaming and progress reporting
    async fn download_file_with_progress(
        &self,
        url: &str,
        output_path: &str,
        headers: Option<HashMap<String, String>>,
        job_id: &str,
        window: &Window,
        start_percent: f64,
        end_percent: f64,
        _stage: &str,
    ) -> Result<(), String> {
        // Ensure parent directory exists
        if let Some(parent) = Path::new(output_path).parent() {
            tokio::fs::create_dir_all(parent)
                .await
                .map_err(|e| format!("Failed to create directory: {}", e))?;
        }

        let mut request = self.client.get(url);

        if let Some(hdrs) = headers {
            for (key, value) in hdrs {
                request = request.header(&key, &value);
            }
        }

        let response = request
            .send()
            .await
            .map_err(|e| format!("Failed to fetch {}: {}", url, e))?;

        if !response.status().is_success() {
            return Err(format!("HTTP error for {}: {}", url, response.status()));
        }

        let total = response.content_length();
        let mut downloaded: u64 = 0;
        let mut last_emit = Instant::now();
        let mut last_downloaded: u64 = 0;
        let percent_range = end_percent - start_percent;

        // Stream download to file
        let mut file = File::create(output_path)
            .await
            .map_err(|e| format!("Failed to create file: {}", e))?;

        let mut stream = response.bytes_stream();

        while let Some(chunk_result) = stream.next().await {
            let chunk = chunk_result.map_err(|e| format!("Stream error: {}", e))?;
            downloaded += chunk.len() as u64;

            file.write_all(&chunk)
                .await
                .map_err(|e| format!("Write error: {}", e))?;

            // Emit progress every 100ms
            if last_emit.elapsed().as_millis() >= 100 {
                let elapsed = last_emit.elapsed().as_secs_f64();
                let speed = if elapsed > 0.0 {
                    ((downloaded - last_downloaded) as f64 / elapsed) as u64
                } else {
                    0
                };

                // Calculate percent within the allocated range
                let file_percent = total
                    .map(|t| (downloaded as f64 / t as f64) * 100.0)
                    .unwrap_or(50.0);
                let overall_percent = start_percent + (file_percent / 100.0) * percent_range;

                let progress = DownloadProgress {
                    job_id: job_id.to_string(),
                    downloaded,
                    total,
                    speed,
                    percent: overall_percent,
                };

                let _ = window.emit("download-progress", &progress);

                last_emit = Instant::now();
                last_downloaded = downloaded;
            }
        }

        file.flush()
            .await
            .map_err(|e| format!("Flush error: {}", e))?;

        Ok(())
    }
}

impl Default for SimpleDownloader {
    fn default() -> Self {
        Self::new()
    }
}
