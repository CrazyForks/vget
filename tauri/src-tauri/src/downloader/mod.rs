mod simple;

pub use simple::*;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DownloadProgress {
    pub job_id: String,
    pub downloaded: u64,
    pub total: Option<u64>,
    pub speed: u64,
    pub percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum DownloadStatus {
    Pending,
    Downloading,
    Completed,
    Failed,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DownloadJob {
    pub id: String,
    pub url: String,
    pub output_path: String,
    pub status: DownloadStatus,
    pub progress: Option<DownloadProgress>,
    pub error: Option<String>,
}

/// Global download manager to track active downloads
pub struct DownloadManager {
    jobs: Arc<RwLock<HashMap<String, DownloadJob>>>,
    cancellation_tokens: Arc<RwLock<HashMap<String, tokio::sync::watch::Sender<bool>>>>,
}

impl DownloadManager {
    pub fn new() -> Self {
        Self {
            jobs: Arc::new(RwLock::new(HashMap::new())),
            cancellation_tokens: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn add_job(&self, job: DownloadJob) -> tokio::sync::watch::Receiver<bool> {
        let (tx, rx) = tokio::sync::watch::channel(false);
        let job_id = job.id.clone();

        self.jobs.write().await.insert(job_id.clone(), job);
        self.cancellation_tokens.write().await.insert(job_id, tx);

        rx
    }

    pub async fn update_job(&self, job_id: &str, status: DownloadStatus, progress: Option<DownloadProgress>, error: Option<String>) {
        if let Some(job) = self.jobs.write().await.get_mut(job_id) {
            job.status = status;
            job.progress = progress;
            job.error = error;
        }
    }

    pub async fn cancel_job(&self, job_id: &str) -> Result<(), String> {
        if let Some(tx) = self.cancellation_tokens.read().await.get(job_id) {
            tx.send(true).map_err(|e| e.to_string())?;
        }
        self.update_job(job_id, DownloadStatus::Cancelled, None, None).await;
        Ok(())
    }

    pub async fn get_job(&self, job_id: &str) -> Option<DownloadJob> {
        self.jobs.read().await.get(job_id).cloned()
    }

    pub async fn remove_job(&self, job_id: &str) {
        self.jobs.write().await.remove(job_id);
        self.cancellation_tokens.write().await.remove(job_id);
    }
}

impl Default for DownloadManager {
    fn default() -> Self {
        Self::new()
    }
}
