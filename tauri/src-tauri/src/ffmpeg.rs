use ffmpeg_sidecar::command::FfmpegCommand;
use ffmpeg_sidecar::event::{FfmpegEvent, LogLevel};
use std::path::Path;

/// Check if ffmpeg sidecar is available
pub fn ffmpeg_available() -> bool {
    FfmpegCommand::new().print_command().spawn().is_ok()
}

/// Merge separate video and audio files into a single output file.
/// Uses stream copy (-c copy) for fast merging without re-encoding.
pub async fn merge_video_audio(
    video_path: &str,
    audio_path: &str,
    output_path: &str,
    delete_originals: bool,
) -> Result<(), String> {
    // Validate input files exist
    if !Path::new(video_path).exists() {
        return Err(format!("Video file not found: {}", video_path));
    }
    if !Path::new(audio_path).exists() {
        return Err(format!("Audio file not found: {}", audio_path));
    }

    // Create output directory if needed
    if let Some(parent) = Path::new(output_path).parent() {
        std::fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create output directory: {}", e))?;
    }

    // Run ffmpeg merge command
    // -i video_path -i audio_path -c copy -map 0:v -map 1:a output_path
    tokio::task::spawn_blocking({
        let video_path = video_path.to_string();
        let audio_path = audio_path.to_string();
        let output_path = output_path.to_string();

        move || {
            let mut cmd = FfmpegCommand::new();
            cmd.args(["-y"]) // Overwrite output
                .input(&video_path)
                .input(&audio_path)
                .args(["-map", "0:v"]) // Video from first input
                .args(["-map", "1:a"]) // Audio from second input
                .args(["-c", "copy"]) // Stream copy, no re-encoding
                .output(&output_path);

            let mut child = cmd.spawn().map_err(|e| format!("Failed to spawn ffmpeg: {}", e))?;

            // Collect events and check for errors
            let mut error_msg: Option<String> = None;

            for event in child.iter().expect("Failed to iterate ffmpeg events") {
                match event {
                    FfmpegEvent::Log(LogLevel::Error, msg) => {
                        eprintln!("[ffmpeg error] {}", msg);
                        error_msg = Some(msg);
                    }
                    FfmpegEvent::Log(LogLevel::Warning, msg) => {
                        eprintln!("[ffmpeg warning] {}", msg);
                    }
                    FfmpegEvent::Progress(progress) => {
                        // Could emit progress events here if needed
                        let _ = progress;
                    }
                    FfmpegEvent::Done => {
                        break;
                    }
                    _ => {}
                }
            }

            // Check if output file was created
            if !Path::new(&output_path).exists() {
                return Err(error_msg.unwrap_or_else(|| "FFmpeg failed to create output file".to_string()));
            }

            Ok(())
        }
    })
    .await
    .map_err(|e| format!("Task join error: {}", e))??;

    // Delete original files if requested
    if delete_originals {
        if let Err(e) = std::fs::remove_file(video_path) {
            eprintln!("[ffmpeg] Warning: could not remove video file: {}", e);
        }
        if let Err(e) = std::fs::remove_file(audio_path) {
            eprintln!("[ffmpeg] Warning: could not remove audio file: {}", e);
        }
    }

    Ok(())
}

/// Get the path to the bundled ffmpeg binary
pub fn get_ffmpeg_path() -> std::path::PathBuf {
    // ffmpeg-sidecar will automatically find the binary
    // For Tauri sidecar, it's in the app's resource directory
    ffmpeg_sidecar::paths::ffmpeg_path()
}
