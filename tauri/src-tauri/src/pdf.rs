use lopdf::{Document, Object, ObjectId};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Serialize, Deserialize)]
pub struct PdfInfo {
    pub path: String,
    pub pages: usize,
    pub title: Option<String>,
    pub author: Option<String>,
}

/// Get basic info about a PDF file
pub fn get_pdf_info(path: &str) -> Result<PdfInfo, String> {
    let doc = Document::load(path).map_err(|e| format!("Failed to load PDF: {}", e))?;

    let pages = doc.get_pages().len();

    // Try to get metadata from document info dictionary
    let mut title = None;
    let mut author = None;

    if let Ok(info_dict) = doc.trailer.get(b"Info") {
        if let Ok(info_ref) = info_dict.as_reference() {
            if let Ok(info_obj) = doc.get_object(info_ref) {
                if let Object::Dictionary(dict) = info_obj {
                    if let Ok(t) = dict.get(b"Title") {
                        if let Ok(s) = t.as_str() {
                            title = Some(String::from_utf8_lossy(s).to_string());
                        }
                    }
                    if let Ok(a) = dict.get(b"Author") {
                        if let Ok(s) = a.as_str() {
                            author = Some(String::from_utf8_lossy(s).to_string());
                        }
                    }
                }
            }
        }
    }

    Ok(PdfInfo {
        path: path.to_string(),
        pages,
        title,
        author,
    })
}

/// Merge multiple PDF files into one
pub fn merge_pdfs(input_paths: &[String], output_path: &str) -> Result<(), String> {
    if input_paths.is_empty() {
        return Err("No input files provided".to_string());
    }

    if input_paths.len() == 1 {
        // Just copy the file
        std::fs::copy(&input_paths[0], output_path)
            .map_err(|e| format!("Failed to copy PDF: {}", e))?;
        return Ok(());
    }

    // Load the first document as the base
    let mut base_doc =
        Document::load(&input_paths[0]).map_err(|e| format!("Failed to load first PDF: {}", e))?;

    // Merge subsequent documents
    for (idx, path) in input_paths.iter().enumerate().skip(1) {
        let doc = Document::load(path)
            .map_err(|e| format!("Failed to load PDF {}: {}", idx + 1, e))?;

        // Get the max object id from base document
        let mut max_id = base_doc.max_id;

        // Map old object ids to new ones
        let mut id_map: HashMap<ObjectId, ObjectId> = HashMap::new();

        // Copy all objects from the source document with new IDs
        for (old_id, object) in doc.objects.iter() {
            max_id += 1;
            let new_id = (max_id, 0);
            id_map.insert(*old_id, new_id);
            base_doc.objects.insert(new_id, object.clone());
        }

        // Update references in copied objects
        for (_old_id, new_id) in id_map.iter() {
            if let Some(obj) = base_doc.objects.get_mut(new_id) {
                update_references(obj, &id_map);
            }
        }

        // Get pages from the source document and add them to base
        let src_pages = doc.get_pages();
        for (_page_num, page_id) in src_pages {
            if let Some(new_page_id) = id_map.get(&page_id) {
                // Add the page to the base document's page tree
                let catalog = base_doc.catalog().map_err(|e| e.to_string())?;
                let pages_ref = catalog.get(b"Pages").map_err(|e| e.to_string())?;
                let pages_id = pages_ref.as_reference().map_err(|e| e.to_string())?;

                if let Ok(Object::Dictionary(ref mut pages_dict)) =
                    base_doc.get_object_mut(pages_id)
                {
                    if let Ok(kids) = pages_dict.get_mut(b"Kids") {
                        if let Object::Array(ref mut kids_array) = kids {
                            kids_array.push(Object::Reference(*new_page_id));
                        }
                    }
                    // Update page count
                    if let Ok(count) = pages_dict.get_mut(b"Count") {
                        if let Object::Integer(ref mut n) = count {
                            *n += 1;
                        }
                    }
                }
            }
        }

        base_doc.max_id = max_id;
    }

    // Save the merged document
    base_doc
        .save(output_path)
        .map_err(|e| format!("Failed to save merged PDF: {}", e))?;

    Ok(())
}

/// Update object references after copying
fn update_references(obj: &mut Object, id_map: &HashMap<ObjectId, ObjectId>) {
    match obj {
        Object::Reference(ref mut id) => {
            if let Some(new_id) = id_map.get(id) {
                *id = *new_id;
            }
        }
        Object::Array(arr) => {
            for item in arr.iter_mut() {
                update_references(item, id_map);
            }
        }
        Object::Dictionary(dict) => {
            for (_, value) in dict.iter_mut() {
                update_references(value, id_map);
            }
        }
        Object::Stream(stream) => {
            for (_, value) in stream.dict.iter_mut() {
                update_references(value, id_map);
            }
        }
        _ => {}
    }
}

/// Convert images to a single PDF using printpdf
pub fn images_to_pdf(image_paths: &[String], output_path: &str) -> Result<(), String> {
    use printpdf::*;

    if image_paths.is_empty() {
        return Err("No image files provided".to_string());
    }

    // Create a new PDF document
    let mut doc = PdfDocument::new("Images to PDF");
    let mut warnings = Vec::new();

    for image_path in image_paths.iter() {
        // Read the image file
        let image_bytes = std::fs::read(image_path)
            .map_err(|e| format!("Failed to read image {}: {}", image_path, e))?;

        // Decode image to get dimensions using the image crate
        let img = ::image::load_from_memory(&image_bytes)
            .map_err(|e| format!("Failed to decode image {}: {}", image_path, e))?;

        let (img_width, img_height) = ::image::GenericImageView::dimensions(&img);

        // Calculate page size (A4 max, scale if needed)
        let max_width_mm: f32 = 210.0;
        let max_height_mm: f32 = 297.0;

        // Convert pixels to mm (assuming 96 DPI)
        let dpi: f32 = 96.0;
        let img_width_mm = (img_width as f32 / dpi) * 25.4;
        let img_height_mm = (img_height as f32 / dpi) * 25.4;

        // Scale to fit within A4 while maintaining aspect ratio
        let scale = (max_width_mm / img_width_mm)
            .min(max_height_mm / img_height_mm)
            .min(1.0);
        let final_width_mm = img_width_mm * scale;
        let final_height_mm = img_height_mm * scale;

        // Decode image for printpdf
        let raw_image = RawImage::decode_from_bytes(&image_bytes, &mut warnings)
            .map_err(|e| format!("Failed to decode image for PDF: {}", e))?;

        // Add image to document resources
        let image_id = doc.add_image(&raw_image);

        // Create page with image
        let page = PdfPage::new(
            Mm(final_width_mm),
            Mm(final_height_mm),
            vec![Op::UseXobject {
                id: image_id.into(),
                transform: XObjectTransform {
                    translate_x: Some(Pt(0.0)),
                    translate_y: Some(Pt(0.0)),
                    scale_x: Some(scale),
                    scale_y: Some(scale),
                    ..Default::default()
                },
            }],
        );

        doc.pages.push(page);
    }

    // Save the PDF
    let pdf_bytes = doc.save(&PdfSaveOptions::default(), &mut warnings);
    std::fs::write(output_path, pdf_bytes)
        .map_err(|e| format!("Failed to save PDF: {}", e))?;

    Ok(())
}

/// Result of watermark removal attempt
#[derive(Debug, Serialize, Deserialize)]
pub struct WatermarkRemovalResult {
    pub success: bool,
    pub items_removed: usize,
    pub message: String,
}

/// Attempt to remove watermarks from a PDF
/// This works for overlay-type watermarks but not for embedded ones
pub fn remove_watermark(input_path: &str, output_path: &str) -> Result<WatermarkRemovalResult, String> {
    let mut doc = Document::load(input_path)
        .map_err(|e| format!("Failed to load PDF: {}", e))?;

    let mut items_removed = 0;

    // Common watermark-related names to look for
    let watermark_indicators: Vec<&[u8]> = vec![
        b"Watermark",
        b"watermark",
        b"WATERMARK",
        b"WM",
        b"wm",
        b"Overlay",
        b"overlay",
        b"Background",
        b"Draft",
        b"DRAFT",
        b"Confidential",
        b"CONFIDENTIAL",
        b"Sample",
        b"SAMPLE",
        b"Copy",
        b"COPY",
    ];

    // Collect object IDs that might be watermarks
    let mut objects_to_remove: Vec<ObjectId> = Vec::new();

    // Check all objects for watermark indicators
    for (obj_id, obj) in doc.objects.iter() {
        if let Object::Dictionary(dict) = obj {
            // Check for watermark-named objects
            if let Ok(name) = dict.get(b"Name") {
                if let Object::Name(n) = name {
                    for indicator in &watermark_indicators {
                        if n.windows(indicator.len()).any(|w| w == *indicator) {
                            objects_to_remove.push(*obj_id);
                            break;
                        }
                    }
                }
            }

            // Check Subtype for watermark
            if let Ok(subtype) = dict.get(b"Subtype") {
                if let Object::Name(n) = subtype {
                    if n == b"Watermark" {
                        objects_to_remove.push(*obj_id);
                    }
                }
            }

            // Check Type for watermark annotation
            if let Ok(type_obj) = dict.get(b"Type") {
                if let Object::Name(n) = type_obj {
                    if n == b"Annot" {
                        if let Ok(subtype) = dict.get(b"Subtype") {
                            if let Object::Name(st) = subtype {
                                if st == b"Watermark" {
                                    objects_to_remove.push(*obj_id);
                                }
                            }
                        }
                    }
                }
            }
        }

        // Check streams for watermark content
        if let Object::Stream(stream) = obj {
            if let Ok(name) = stream.dict.get(b"Name") {
                if let Object::Name(n) = name {
                    for indicator in &watermark_indicators {
                        if n.windows(indicator.len()).any(|w| w == *indicator) {
                            objects_to_remove.push(*obj_id);
                            break;
                        }
                    }
                }
            }
        }
    }

    // Remove identified watermark objects
    for obj_id in &objects_to_remove {
        doc.objects.remove(obj_id);
        items_removed += 1;
    }

    // Also try to clean up page annotations that might be watermarks
    let pages: Vec<ObjectId> = doc.get_pages().values().cloned().collect();

    for page_id in pages {
        // First, collect annotation info
        let annots_to_process: Option<(Vec<Object>, Vec<ObjectId>)> = {
            if let Ok(Object::Dictionary(page_dict)) = doc.get_object(page_id) {
                if let Ok(annots) = page_dict.get(b"Annots") {
                    if let Object::Array(annot_array) = annots {
                        let mut watermark_annots = Vec::new();
                        for annot_ref in annot_array {
                            if let Object::Reference(annot_id) = annot_ref {
                                if objects_to_remove.contains(annot_id) {
                                    watermark_annots.push(*annot_id);
                                } else if let Ok(annot_obj) = doc.get_object(*annot_id) {
                                    if let Object::Dictionary(annot_dict) = annot_obj {
                                        if let Ok(subtype) = annot_dict.get(b"Subtype") {
                                            if let Object::Name(n) = subtype {
                                                if n == b"Watermark" {
                                                    watermark_annots.push(*annot_id);
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                        if !watermark_annots.is_empty() {
                            Some((annot_array.clone(), watermark_annots))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                } else {
                    None
                }
            } else {
                None
            }
        };

        // Now modify the page if needed
        if let Some((annot_array, watermark_annots)) = annots_to_process {
            let new_annots: Vec<Object> = annot_array
                .into_iter()
                .filter(|annot_ref| {
                    if let Object::Reference(annot_id) = annot_ref {
                        !watermark_annots.contains(annot_id)
                    } else {
                        true
                    }
                })
                .collect();

            items_removed += watermark_annots.len();

            if let Ok(Object::Dictionary(ref mut page_dict)) = doc.get_object_mut(page_id) {
                page_dict.set(b"Annots", Object::Array(new_annots));
            }
        }
    }

    // Save the modified document
    doc.save(output_path)
        .map_err(|e| format!("Failed to save PDF: {}", e))?;

    let message = if items_removed > 0 {
        format!("Found and removed {} potential watermark element(s). Please check the output file.", items_removed)
    } else {
        "No obvious watermark elements were found. The watermark may be embedded in the page content, which cannot be easily removed.".to_string()
    };

    Ok(WatermarkRemovalResult {
        success: items_removed > 0,
        items_removed,
        message,
    })
}

/// Delete specific pages from a PDF
pub fn delete_pages(input_path: &str, output_path: &str, pages_to_delete: &[u32]) -> Result<(), String> {
    let mut doc = Document::load(input_path)
        .map_err(|e| format!("Failed to load PDF: {}", e))?;

    let total_pages = doc.get_pages().len();

    // Validate page numbers
    for &page in pages_to_delete {
        if page == 0 || page as usize > total_pages {
            return Err(format!(
                "Invalid page number: {}. PDF has {} pages (1-indexed).",
                page, total_pages
            ));
        }
    }

    // Check we're not deleting all pages
    if pages_to_delete.len() >= total_pages {
        return Err("Cannot delete all pages from PDF".to_string());
    }

    // Delete pages in reverse order to maintain correct indices
    let mut sorted_pages: Vec<u32> = pages_to_delete.to_vec();
    sorted_pages.sort_by(|a, b| b.cmp(a)); // Reverse sort

    for page_num in sorted_pages {
        doc.delete_pages(&[page_num]);
    }

    // Save the modified document
    doc.save(output_path)
        .map_err(|e| format!("Failed to save PDF: {}", e))?;

    Ok(())
}

/// Print a PDF file using the system printer
pub fn print_pdf(path: &str) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    {
        std::process::Command::new("lpr")
            .arg(path)
            .status()
            .map_err(|e| format!("Failed to print: {}", e))?;
    }

    #[cfg(target_os = "windows")]
    {
        std::process::Command::new("cmd")
            .args(["/C", "print", path])
            .status()
            .map_err(|e| format!("Failed to print: {}", e))?;
    }

    #[cfg(target_os = "linux")]
    {
        std::process::Command::new("lpr")
            .arg(path)
            .status()
            .map_err(|e| format!("Failed to print: {}", e))?;
    }

    Ok(())
}

/// Open PDF with system default application
pub fn open_pdf_external(path: &str) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    {
        std::process::Command::new("open")
            .arg(path)
            .status()
            .map_err(|e| format!("Failed to open PDF: {}", e))?;
    }

    #[cfg(target_os = "windows")]
    {
        std::process::Command::new("cmd")
            .args(["/C", "start", "", path])
            .status()
            .map_err(|e| format!("Failed to open PDF: {}", e))?;
    }

    #[cfg(target_os = "linux")]
    {
        std::process::Command::new("xdg-open")
            .arg(path)
            .status()
            .map_err(|e| format!("Failed to open PDF: {}", e))?;
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_get_pdf_info() {
        // This would require a test PDF file
    }
}
