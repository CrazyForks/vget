mod types;
mod direct;

pub use types::*;

use url::Url;

/// Extract media information from a URL
pub async fn extract_media(url_str: &str) -> Result<MediaInfo, ExtractError> {
    let url = Url::parse(url_str).map_err(|_| ExtractError::InvalidUrl(url_str.to_string()))?;

    // Check for direct file URLs first
    if direct::DirectExtractor::matches(&url) {
        return direct::DirectExtractor::extract(url_str).await;
    }

    // TODO: Add more extractors here
    // - Twitter
    // - Bilibili
    // - Xiaoyuzhou
    // - etc.

    Err(ExtractError::NoExtractor(url_str.to_string()))
}
