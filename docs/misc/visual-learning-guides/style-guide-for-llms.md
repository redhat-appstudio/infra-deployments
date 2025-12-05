---
title: Visual Learning Guides - Style Guide for LLMs
---

# 📋 Style Guide (For LLMs)

**This document is technical reference for AI assistants generating or updating visual learning guides.**

## File Structure

Each guide is a standalone HTML file with:
- Full `<!DOCTYPE html>` document
- Embedded CSS in `<style>` tags (no external stylesheets)
- Embedded JavaScript in `<script>` tags (no external scripts)
- No external dependencies

## Color Palette

```css
/* Primary gradient (headers, accents) */
background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

/* Card backgrounds */
background: white;
box-shadow: 0 4px 6px rgba(0,0,0,0.1);

/* Difficulty tags */
.beginner { background: #48bb78; }      /* Green */
.intermediate { background: #ed8936; }  /* Orange */
.advanced { background: #f56565; }      /* Red */

/* Category colors */
Success/Green: #48bb78, #10b981
Warning/Orange: #ed8936, #f59e0b
Info/Blue: #4299e1, #3b82f6
Purple accent: #9f7aea, #764ba2
Error/Red: #ef4444, #f56565
```

## Required Elements

Every guide should include:

1. **Back link** at the top:
```html
<a href="index.html" class="back-link">← Back to Visual Guide Index</a>
```

2. **Header** with gradient background:
```html
<div class="header">
    <h1>🎨 Guide Title</h1>
    <p>Brief description</p>
</div>
```

3. **Difficulty tags** on interactive elements:
```html
<span class="difficulty beginner">Beginner</span>
<span class="difficulty intermediate">Intermediate</span>
```

4. **Card-based layout** for navigation/content:
```html
<div class="card">
    <div class="card-icon">🧩</div>
    <h2>Section Title</h2>
    <p>Description text</p>
</div>
```

## Typography

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
```

## Code Blocks

**IMPORTANT:** Always include `white-space: pre-wrap` to preserve line breaks:

```css
.code-block {
    background: #2d3748;
    color: #68d391;
    padding: 20px;
    border-radius: 8px;
    font-family: 'Courier New', monospace;
    font-size: 0.9em;
    line-height: 1.6;
    white-space: pre-wrap;  /* Required for line breaks! */
    overflow-x: auto;
}
```

## File Paths & Directory Structures

**Don't use code blocks for file trees** - they render poorly. Instead, use **card layouts**:

```html
<!-- Good: Card layout for file locations -->
<div style="background: #f0fff4; border-radius: 10px; padding: 20px; border-left: 4px solid #48bb78;">
    <h3 style="color: #2f855a;">📁 Directory Name</h3>
    <code style="background: #2d3748; color: #68d391; padding: 8px 12px; border-radius: 5px;">
        path/to/file.yaml
    </code>
    <span style="color: #48bb78; font-weight: bold;">← Description</span>
</div>
```

Use **colored boxes** with:
- Blue (#4299e1) for base/shared resources
- Green (#48bb78) for development  
- Orange (#ed8936) for staging
- Red (#f56565) for production

## Responsive Design

All guides should work on mobile. Use:
```css
@media (max-width: 768px) {
    /* Mobile adjustments */
}
```

## Index Card Template

When adding a new guide, add this to `index.html`:
```html
<a href="visual-NEW-GUIDE.html" class="card">
    <div class="card-icon">🆕</div>
    <h2>Guide Title</h2>
    <p>Description of what this guide covers and why it's useful.</p>
    <span class="difficulty intermediate">Intermediate</span>
    <span class="card-tag">Tag</span>
</a>
```

