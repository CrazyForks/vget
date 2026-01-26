use base64::{engine::general_purpose::STANDARD, Engine};
use headless_chrome::{types::PrintToPdfOptions, Browser, LaunchOptions};
use pulldown_cmark::{CodeBlockKind, Event, HeadingLevel, Options, Parser, Tag, TagEnd};
use std::fs;
use std::path::Path;
use syntect::highlighting::ThemeSet;
use syntect::html::highlighted_html_for_string;
use syntect::parsing::SyntaxSet;

// Embed fonts at compile time
static INTER_REGULAR: &[u8] = include_bytes!("../resources/fonts/Inter-Regular.woff2");
static INTER_MEDIUM: &[u8] = include_bytes!("../resources/fonts/Inter-Medium.woff2");
static INTER_SEMIBOLD: &[u8] = include_bytes!("../resources/fonts/Inter-SemiBold.woff2");
static INTER_BOLD: &[u8] = include_bytes!("../resources/fonts/Inter-Bold.woff2");
static INTER_EXTRABOLD: &[u8] = include_bytes!("../resources/fonts/Inter-ExtraBold.woff2");
static JETBRAINS_MONO_REGULAR: &[u8] = include_bytes!("../resources/fonts/JetBrainsMono-Regular.woff2");
static JETBRAINS_MONO_MEDIUM: &[u8] = include_bytes!("../resources/fonts/JetBrainsMono-Medium.woff2");

/// Convert markdown file to PDF
pub fn convert_md_to_pdf(
    input_path: &str,
    output_path: &str,
    theme: &str,
    page_size: &str,
) -> Result<(), String> {
    // Read markdown file
    let markdown = fs::read_to_string(input_path)
        .map_err(|e| format!("Failed to read markdown file: {}", e))?;

    // Parse markdown to HTML with syntax highlighting
    let html_content = markdown_to_html(&markdown, theme);

    // Generate full HTML with styling
    let full_html = generate_styled_html(&html_content, theme);

    // Convert HTML to PDF using headless Chrome
    html_to_pdf(&full_html, output_path, page_size)?;

    Ok(())
}

/// Parse markdown to HTML using pulldown-cmark with syntax highlighting
fn markdown_to_html(markdown: &str, theme: &str) -> String {
    let mut options = Options::empty();
    options.insert(Options::ENABLE_TABLES);
    options.insert(Options::ENABLE_FOOTNOTES);
    options.insert(Options::ENABLE_STRIKETHROUGH);
    options.insert(Options::ENABLE_TASKLISTS);
    options.insert(Options::ENABLE_HEADING_ATTRIBUTES);

    let parser = Parser::new_ext(markdown, options);

    // Load syntax highlighting
    let ss = SyntaxSet::load_defaults_newlines();
    let ts = ThemeSet::load_defaults();
    let syntax_theme = if theme == "dark" {
        &ts.themes["base16-ocean.dark"]
    } else {
        &ts.themes["InspiredGitHub"]
    };

    let mut html_output = String::new();
    let mut in_code_block = false;
    let mut in_table_head = false;
    let mut code_lang = String::new();
    let mut code_content = String::new();

    for event in parser {
        match event {
            Event::Start(Tag::CodeBlock(kind)) => {
                in_code_block = true;
                code_content.clear();
                code_lang = match kind {
                    CodeBlockKind::Fenced(lang) => lang.to_string(),
                    CodeBlockKind::Indented => String::new(),
                };
            }
            Event::End(TagEnd::CodeBlock) => {
                in_code_block = false;
                // Try to find syntax for the language
                let syntax = if !code_lang.is_empty() {
                    ss.find_syntax_by_token(&code_lang)
                } else {
                    None
                }
                .unwrap_or_else(|| ss.find_syntax_plain_text());

                // Generate highlighted HTML
                match highlighted_html_for_string(&code_content, &ss, syntax, syntax_theme) {
                    Ok(highlighted) => {
                        html_output.push_str(&highlighted);
                    }
                    Err(_) => {
                        // Fallback to plain code block
                        html_output.push_str("<pre><code>");
                        html_output.push_str(&html_escape(&code_content));
                        html_output.push_str("</code></pre>\n");
                    }
                }
            }
            Event::Text(text) if in_code_block => {
                code_content.push_str(&text);
            }
            Event::Start(Tag::Table(alignments)) => {
                html_output.push_str("<table>\n");
                // Store alignments for later use (simplified - we just open the table)
                let _ = alignments;
            }
            Event::End(TagEnd::Table) => {
                html_output.push_str("</tbody>\n</table>\n");
            }
            Event::Start(Tag::TableHead) => {
                in_table_head = true;
                html_output.push_str("<thead>\n");
            }
            Event::End(TagEnd::TableHead) => {
                in_table_head = false;
                html_output.push_str("</thead>\n<tbody>\n");
            }
            Event::Start(Tag::TableRow) => {
                html_output.push_str("<tr>\n");
            }
            Event::End(TagEnd::TableRow) => {
                html_output.push_str("</tr>\n");
            }
            Event::Start(Tag::TableCell) => {
                if in_table_head {
                    html_output.push_str("<th>");
                } else {
                    html_output.push_str("<td>");
                }
            }
            Event::End(TagEnd::TableCell) => {
                if in_table_head {
                    html_output.push_str("</th>\n");
                } else {
                    html_output.push_str("</td>\n");
                }
            }
            Event::Start(Tag::Heading { level, .. }) => {
                let level_num = heading_level_to_u8(level);
                html_output.push_str(&format!("<h{}>", level_num));
            }
            Event::End(TagEnd::Heading(level)) => {
                let level_num = heading_level_to_u8(level);
                html_output.push_str(&format!("</h{}>\n", level_num));
            }
            Event::Start(Tag::Paragraph) => {
                html_output.push_str("<p>");
            }
            Event::End(TagEnd::Paragraph) => {
                html_output.push_str("</p>\n");
            }
            Event::Start(Tag::List(None)) => {
                html_output.push_str("<ul>\n");
            }
            Event::Start(Tag::List(Some(start))) => {
                html_output.push_str(&format!("<ol start=\"{}\">\n", start));
            }
            Event::End(TagEnd::List(ordered)) => {
                if ordered {
                    html_output.push_str("</ol>\n");
                } else {
                    html_output.push_str("</ul>\n");
                }
            }
            Event::Start(Tag::Item) => {
                html_output.push_str("<li>");
            }
            Event::End(TagEnd::Item) => {
                html_output.push_str("</li>\n");
            }
            Event::Start(Tag::BlockQuote(_)) => {
                html_output.push_str("<blockquote>\n");
            }
            Event::End(TagEnd::BlockQuote(_)) => {
                html_output.push_str("</blockquote>\n");
            }
            Event::Start(Tag::Emphasis) => {
                html_output.push_str("<em>");
            }
            Event::End(TagEnd::Emphasis) => {
                html_output.push_str("</em>");
            }
            Event::Start(Tag::Strong) => {
                html_output.push_str("<strong>");
            }
            Event::End(TagEnd::Strong) => {
                html_output.push_str("</strong>");
            }
            Event::Start(Tag::Strikethrough) => {
                html_output.push_str("<del>");
            }
            Event::End(TagEnd::Strikethrough) => {
                html_output.push_str("</del>");
            }
            Event::Start(Tag::Link { dest_url, title, .. }) => {
                html_output.push_str(&format!(
                    "<a href=\"{}\" title=\"{}\">",
                    html_escape(&dest_url),
                    html_escape(&title)
                ));
            }
            Event::End(TagEnd::Link) => {
                html_output.push_str("</a>");
            }
            Event::Start(Tag::Image { dest_url, title, .. }) => {
                html_output.push_str(&format!(
                    "<img src=\"{}\" alt=\"",
                    html_escape(&dest_url)
                ));
                // The alt text will come as a Text event
                let _ = title;
            }
            Event::End(TagEnd::Image) => {
                html_output.push_str("\" />");
            }
            Event::Code(code) => {
                html_output.push_str("<code>");
                html_output.push_str(&html_escape(&code));
                html_output.push_str("</code>");
            }
            Event::Text(text) => {
                html_output.push_str(&html_escape(&text));
            }
            Event::SoftBreak => {
                html_output.push('\n');
            }
            Event::HardBreak => {
                html_output.push_str("<br />\n");
            }
            Event::Rule => {
                html_output.push_str("<hr />\n");
            }
            Event::TaskListMarker(checked) => {
                if checked {
                    html_output.push_str("<input type=\"checkbox\" checked disabled /> ");
                } else {
                    html_output.push_str("<input type=\"checkbox\" disabled /> ");
                }
            }
            _ => {}
        }
    }

    html_output
}

/// Escape HTML special characters
fn html_escape(text: &str) -> String {
    text.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&#39;")
}

/// Convert HeadingLevel enum to u8
fn heading_level_to_u8(level: HeadingLevel) -> u8 {
    match level {
        HeadingLevel::H1 => 1,
        HeadingLevel::H2 => 2,
        HeadingLevel::H3 => 3,
        HeadingLevel::H4 => 4,
        HeadingLevel::H5 => 5,
        HeadingLevel::H6 => 6,
    }
}

/// Generate @font-face CSS rules with embedded base64 fonts
fn generate_font_css() -> String {
    format!(
        r#"
/* Embedded Fonts */
@font-face {{
    font-family: 'Inter';
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'Inter';
    font-style: normal;
    font-weight: 500;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'Inter';
    font-style: normal;
    font-weight: 600;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'Inter';
    font-style: normal;
    font-weight: 700;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'Inter';
    font-style: normal;
    font-weight: 800;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'JetBrains Mono';
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
@font-face {{
    font-family: 'JetBrains Mono';
    font-style: normal;
    font-weight: 500;
    font-display: swap;
    src: url(data:font/woff2;base64,{}) format('woff2');
}}
"#,
        STANDARD.encode(INTER_REGULAR),
        STANDARD.encode(INTER_MEDIUM),
        STANDARD.encode(INTER_SEMIBOLD),
        STANDARD.encode(INTER_BOLD),
        STANDARD.encode(INTER_EXTRABOLD),
        STANDARD.encode(JETBRAINS_MONO_REGULAR),
        STANDARD.encode(JETBRAINS_MONO_MEDIUM),
    )
}

/// Generate full HTML document with CSS styling
fn generate_styled_html(content: &str, theme: &str) -> String {
    let font_css = generate_font_css();
    let theme_css = get_theme_css(theme);

    format!(
        r#"<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
{font_css}
{theme_css}
    </style>
</head>
<body>
    <article class="markdown-body">
{content}
    </article>
</body>
</html>"#
    )
}

/// Get CSS based on theme
fn get_theme_css(theme: &str) -> &'static str {
    match theme {
        "dark" => DARK_THEME_CSS,
        _ => LIGHT_THEME_CSS,
    }
}

/// Convert HTML to PDF using headless Chrome
fn html_to_pdf(html: &str, output_path: &str, page_size: &str) -> Result<(), String> {
    // Write HTML to a temp file (more reliable than data URLs for large content)
    let temp_dir = std::env::temp_dir();
    let temp_html_path = temp_dir.join(format!("md2pdf_{}.html", std::process::id()));
    fs::write(&temp_html_path, html)
        .map_err(|e| format!("Failed to write temp HTML file: {}", e))?;

    let browser = Browser::new(
        LaunchOptions::default_builder()
            .headless(true)
            .sandbox(false)
            .build()
            .map_err(|e| format!("Failed to build launch options: {}", e))?,
    )
    .map_err(|e| format!("Failed to launch browser: {}", e))?;

    let tab = browser
        .new_tab()
        .map_err(|e| format!("Failed to create new tab: {}", e))?;

    // Navigate to the temp HTML file
    let file_url = format!("file://{}", temp_html_path.display());

    tab.navigate_to(&file_url)
        .map_err(|e| format!("Failed to navigate: {}", e))?;

    tab.wait_until_navigated()
        .map_err(|e| format!("Failed to wait for navigation: {}", e))?;

    // Wait for page to fully render (important for embedded fonts to load)
    std::thread::sleep(std::time::Duration::from_millis(500));

    // Get page dimensions based on page size (in inches)
    let (paper_width, paper_height) = match page_size {
        "Letter" => (8.5, 11.0),
        _ => (8.27, 11.69), // A4 default
    };

    // Generate PDF with custom options
    let options = PrintToPdfOptions {
        landscape: Some(false),
        display_header_footer: Some(false),
        print_background: Some(true),
        scale: Some(1.0),
        paper_width: Some(paper_width),
        paper_height: Some(paper_height),
        margin_top: Some(0.5),
        margin_bottom: Some(0.5),
        margin_left: Some(0.5),
        margin_right: Some(0.5),
        prefer_css_page_size: Some(false),
        ..Default::default()
    };

    let pdf_bytes = tab
        .print_to_pdf(Some(options))
        .map_err(|e| format!("Failed to generate PDF: {}", e))?;

    // Ensure output directory exists
    if let Some(parent) = Path::new(output_path).parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create output directory: {}", e))?;
    }

    // Write PDF to file
    fs::write(output_path, pdf_bytes).map_err(|e| format!("Failed to write PDF: {}", e))?;

    // Clean up temp file
    let _ = fs::remove_file(&temp_html_path);

    Ok(())
}

// Light theme CSS - Clean document styling
const LIGHT_THEME_CSS: &str = r#"
@page {
    margin: 0;
    size: auto;
}

*, *::before, *::after {
    box-sizing: border-box;
}

html {
    font-size: 16px;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
    text-rendering: optimizeLegibility;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

body {
    font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif,
        "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei",
        "Hiragino Kaku Gothic Pro", "Yu Gothic",
        "Apple SD Gothic Neo", "Malgun Gothic",
        "Apple Color Emoji", "Segoe UI Emoji";
    font-size: 1rem;
    font-weight: 400;
    line-height: 1.6;
    color: #111827;
    background-color: #ffffff;
    margin: 0;
    padding: 0;
    word-wrap: break-word;
    font-feature-settings: "kern" 1, "liga" 1, "calt" 1;
}

.markdown-body {
    max-width: 100%;
    margin: 0 auto;
    padding: 48px 56px;
}

/* ==================== Typography ==================== */

h1, h2, h3, h4, h5, h6 {
    font-weight: 600;
    line-height: 1.35;
    color: #111827;
    margin-top: 1.8em;
    margin-bottom: 0.6em;
    page-break-after: avoid;
    page-break-inside: avoid;
}

h1:first-child, h2:first-child, h3:first-child,
h4:first-child, h5:first-child, h6:first-child {
    margin-top: 0;
}

h1 {
    font-size: 2.2rem;
    padding-bottom: 0.3em;
    border-bottom: 1px solid #e5e7eb;
}

h2 {
    font-size: 1.6rem;
    padding-bottom: 0.2em;
    border-bottom: 1px solid #e5e7eb;
}

h3 {
    font-size: 1.3rem;
}

h4 {
    font-size: 1.1rem;
}

h5 {
    font-size: 1rem;
}

h6 {
    font-size: 0.95rem;
    color: #4b5563;
}

/* Paragraphs */
p {
    margin-top: 0;
    margin-bottom: 1em;
}

/* Links */
a {
    color: #0969da;
    text-decoration: none;
    border-bottom: 1px solid rgba(9, 105, 218, 0.2);
}

a:hover {
    border-bottom-color: rgba(9, 105, 218, 0.5);
}

/* ==================== Code ==================== */

code {
    font-family: "JetBrains Mono", "Fira Code", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace;
    font-size: 0.9em;
    font-weight: 500;
    padding: 0.2em 0.4em;
    background-color: #f6f8fa;
    border-radius: 4px;
    color: #24292f;
    border: 1px solid #e5e7eb;
}

/* Code blocks - syntect generates pre with inline styles */
pre {
    font-family: "JetBrains Mono", "Fira Code", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace;
    font-size: 0.88rem;
    line-height: 1.6;
    padding: 1em 1.2em;
    overflow-x: auto;
    background-color: #f6f8fa !important;
    border-radius: 6px;
    border: 1px solid #e5e7eb;
    margin-top: 0;
    margin-bottom: 1.2em;
    page-break-inside: avoid;
}

pre code {
    font-size: inherit;
    font-weight: 400;
    padding: 0;
    background: transparent !important;
    border: none;
    border-radius: 0;
    color: inherit;
    white-space: pre;
}

/* ==================== Blockquotes ==================== */

blockquote {
    margin: 0 0 1em 0;
    padding: 0.2em 1em;
    color: #4b5563;
    border-left: 4px solid #d0d7de;
}

blockquote p {
    margin-bottom: 0.6em;
}

blockquote p:last-child {
    margin-bottom: 0;
}

blockquote code {
    font-style: normal;
}

/* ==================== Lists ==================== */

ul, ol {
    margin-top: 0;
    margin-bottom: 1em;
    padding-left: 2em;
}

ul ul, ol ol, ul ol, ol ul {
    margin-bottom: 0;
    margin-top: 0.4em;
}

li {
    margin-bottom: 0.35em;
    line-height: 1.6;
}

li > p {
    margin-bottom: 0.5em;
}

li > p:last-child {
    margin-bottom: 0;
}

/* Task lists */
li input[type="checkbox"] {
    margin-right: 0.5em;
    margin-left: -0.1em;
    vertical-align: middle;
    position: relative;
    top: -1px;
    width: 14px;
    height: 14px;
    accent-color: #0969da;
}

/* ==================== Tables ==================== */

table {
    border-collapse: collapse;
    margin-top: 0;
    margin-bottom: 1.2em;
    width: 100%;
    page-break-inside: avoid;
}

th, td {
    padding: 0.6em 0.9em;
    text-align: left;
    border: 1px solid #d0d7de;
}

th {
    font-weight: 600;
    font-size: 0.9em;
    color: #374151;
    background-color: #f6f8fa;
}

td {
    color: #1f2937;
}

tbody tr:nth-child(even) {
    background-color: #fbfcfe;
}

/* ==================== Other Elements ==================== */

hr {
    height: 0;
    padding: 0;
    margin: 1.8em 0;
    border: 0;
    border-top: 1px solid #e5e7eb;
    background: transparent;
}

/* Images */
img {
    max-width: 100%;
    height: auto;
    display: block;
    margin: 1.2em auto;
    border-radius: 6px;
    border: 1px solid #e5e7eb;
}

/* Strikethrough */
del {
    color: #6b7280;
    text-decoration: line-through;
}

/* Strong and emphasis */
strong {
    font-weight: 600;
    color: #111827;
}

em {
    font-style: italic;
    color: #374151;
}

/* Definition lists */
dt {
    font-weight: 600;
    margin-top: 1.1em;
    color: #111827;
}

dd {
    margin-left: 1.6em;
    margin-bottom: 0.5em;
    color: #4b5563;
}

/* Footnotes */
.footnote-definition {
    font-size: 0.9rem;
    margin-top: 2em;
    padding-top: 1em;
    border-top: 1px solid #e5e7eb;
    color: #4b5563;
}

/* Keyboard shortcut styling */
kbd {
    font-family: inherit;
    font-size: 0.85em;
    padding: 0.15em 0.35em;
    background-color: #f3f4f6;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    color: #111827;
}

/* ==================== Print Optimizations ==================== */

@media print {
    html {
        font-size: 15px;
    }

    body {
        background: white;
    }

    .markdown-body {
        padding: 40px 48px;
    }

    pre, blockquote, table, img, h1, h2, h3, h4, h5, h6 {
        page-break-inside: avoid;
    }

    h1, h2, h3, h4, h5, h6 {
        page-break-after: avoid;
    }

    p, li {
        orphans: 3;
        widows: 3;
    }

    table {
        border: 1px solid #d0d7de;
    }

    img {
        border: 1px solid #e5e7eb;
    }

    pre {
        border: 1px solid #d0d7de;
    }
}
"#;

// Dark theme CSS - Clean document styling
const DARK_THEME_CSS: &str = r#"
@page {
    margin: 0;
    size: auto;
}

*, *::before, *::after {
    box-sizing: border-box;
}

html {
    font-size: 16px;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
    text-rendering: optimizeLegibility;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

body {
    font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif,
        "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei",
        "Hiragino Kaku Gothic Pro", "Yu Gothic",
        "Apple SD Gothic Neo", "Malgun Gothic",
        "Apple Color Emoji", "Segoe UI Emoji";
    font-size: 1rem;
    font-weight: 400;
    line-height: 1.6;
    color: #e2e8f0;
    background-color: #0f172a;
    margin: 0;
    padding: 0;
    word-wrap: break-word;
    font-feature-settings: "kern" 1, "liga" 1, "calt" 1;
}

.markdown-body {
    max-width: 100%;
    margin: 0 auto;
    padding: 48px 56px;
}

/* ==================== Typography ==================== */

h1, h2, h3, h4, h5, h6 {
    font-weight: 600;
    line-height: 1.35;
    color: #f8fafc;
    margin-top: 1.8em;
    margin-bottom: 0.6em;
    page-break-after: avoid;
    page-break-inside: avoid;
}

h1:first-child, h2:first-child, h3:first-child,
h4:first-child, h5:first-child, h6:first-child {
    margin-top: 0;
}

h1 {
    font-size: 2.2rem;
    padding-bottom: 0.3em;
    border-bottom: 1px solid #334155;
}

h2 {
    font-size: 1.6rem;
    padding-bottom: 0.2em;
    border-bottom: 1px solid #334155;
}

h3 {
    font-size: 1.3rem;
}

h4 {
    font-size: 1.1rem;
    color: #e2e8f0;
}

h5 {
    font-size: 1rem;
    color: #cbd5e1;
}

h6 {
    font-size: 0.95rem;
    color: #94a3b8;
}

/* Paragraphs */
p {
    margin-top: 0;
    margin-bottom: 1em;
}

/* Links */
a {
    color: #60a5fa;
    text-decoration: none;
    border-bottom: 1px solid rgba(96, 165, 250, 0.25);
}

a:hover {
    border-bottom-color: rgba(96, 165, 250, 0.55);
}

/* ==================== Code ==================== */

code {
    font-family: "JetBrains Mono", "Fira Code", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace;
    font-size: 0.9em;
    font-weight: 500;
    padding: 0.2em 0.4em;
    background-color: #111827;
    border-radius: 4px;
    color: #e2e8f0;
    border: 1px solid #334155;
}

/* Code blocks - syntect generates pre with inline styles */
pre {
    font-family: "JetBrains Mono", "Fira Code", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace;
    font-size: 0.88rem;
    line-height: 1.6;
    padding: 1em 1.2em;
    overflow-x: auto;
    background-color: #111827 !important;
    border-radius: 6px;
    border: 1px solid #334155;
    margin-top: 0;
    margin-bottom: 1.2em;
    page-break-inside: avoid;
}

pre code {
    font-size: inherit;
    font-weight: 400;
    padding: 0;
    background: transparent !important;
    border: none;
    border-radius: 0;
    color: inherit;
    white-space: pre;
}

/* ==================== Blockquotes ==================== */

blockquote {
    margin: 0 0 1em 0;
    padding: 0.2em 1em;
    color: #94a3b8;
    border-left: 4px solid #334155;
}

blockquote p {
    margin-bottom: 0.6em;
}

blockquote p:last-child {
    margin-bottom: 0;
}

blockquote code {
    font-style: normal;
}

/* ==================== Lists ==================== */

ul, ol {
    margin-top: 0;
    margin-bottom: 1em;
    padding-left: 2em;
}

ul ul, ol ol, ul ol, ol ul {
    margin-bottom: 0;
    margin-top: 0.4em;
}

li {
    margin-bottom: 0.35em;
    line-height: 1.6;
}

li > p {
    margin-bottom: 0.5em;
}

li > p:last-child {
    margin-bottom: 0;
}

/* Task lists */
li input[type="checkbox"] {
    margin-right: 0.5em;
    margin-left: -0.1em;
    vertical-align: middle;
    position: relative;
    top: -1px;
    width: 14px;
    height: 14px;
    accent-color: #60a5fa;
}

/* ==================== Tables ==================== */

table {
    border-collapse: collapse;
    margin-top: 0;
    margin-bottom: 1.2em;
    width: 100%;
    page-break-inside: avoid;
}

th, td {
    padding: 0.6em 0.9em;
    text-align: left;
    border: 1px solid #334155;
}

th {
    font-weight: 600;
    font-size: 0.9em;
    color: #cbd5e1;
    background-color: #111827;
}

td {
    color: #e2e8f0;
}

tbody tr:nth-child(even) {
    background-color: #0b1220;
}

/* ==================== Other Elements ==================== */

hr {
    height: 0;
    padding: 0;
    margin: 1.8em 0;
    border: 0;
    border-top: 1px solid #334155;
    background: transparent;
}

/* Images */
img {
    max-width: 100%;
    height: auto;
    display: block;
    margin: 1.2em auto;
    border-radius: 6px;
    border: 1px solid #334155;
}

/* Strikethrough */
del {
    color: #94a3b8;
    text-decoration: line-through;
}

/* Strong and emphasis */
strong {
    font-weight: 600;
    color: #f8fafc;
}

em {
    font-style: italic;
    color: #cbd5e1;
}

/* Definition lists */
dt {
    font-weight: 600;
    margin-top: 1.1em;
    color: #f8fafc;
}

dd {
    margin-left: 1.6em;
    margin-bottom: 0.5em;
    color: #94a3b8;
}

/* Footnotes */
.footnote-definition {
    font-size: 0.9rem;
    margin-top: 2em;
    padding-top: 1em;
    border-top: 1px solid #334155;
    color: #94a3b8;
}

/* Keyboard shortcut styling */
kbd {
    font-family: inherit;
    font-size: 0.85em;
    padding: 0.15em 0.35em;
    background-color: #111827;
    border: 1px solid #475569;
    border-radius: 4px;
    color: #e2e8f0;
}

/* ==================== Print Optimizations ==================== */

@media print {
    html {
        font-size: 15px;
    }

    body {
        background: #0f172a;
    }

    .markdown-body {
        padding: 40px 48px;
    }

    pre, blockquote, table, img, h1, h2, h3, h4, h5, h6 {
        page-break-inside: avoid;
    }

    h1, h2, h3, h4, h5, h6 {
        page-break-after: avoid;
    }

    p, li {
        orphans: 3;
        widows: 3;
    }

    table {
        border: 1px solid #334155;
    }

    img {
        border: 1px solid #334155;
    }

    pre {
        border: 1px solid #334155;
    }
}
"#;
