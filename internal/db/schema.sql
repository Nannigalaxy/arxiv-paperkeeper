-- Papers fetched from arXiv
CREATE TABLE IF NOT EXISTS papers (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    abstract TEXT,
    authors TEXT,
    categories TEXT,
    published_at DATETIME,
    updated_at DATETIME,
    pdf_url TEXT,
    arxiv_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User's library (saved papers)
CREATE TABLE IF NOT EXISTS library (
    paper_id TEXT PRIMARY KEY,
    is_read BOOLEAN DEFAULT 0,
    saved_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (paper_id) REFERENCES papers(id) ON DELETE CASCADE
);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

-- Paper-Tag relationship (many-to-many)
CREATE TABLE IF NOT EXISTS paper_tags (
    paper_id TEXT,
    tag_id INTEGER,
    PRIMARY KEY (paper_id, tag_id),
    FOREIGN KEY (paper_id) REFERENCES papers(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_papers_published ON papers(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_papers_categories ON papers(categories);
CREATE INDEX IF NOT EXISTS idx_library_saved ON library(saved_at DESC);
CREATE INDEX IF NOT EXISTS idx_paper_tags_paper ON paper_tags(paper_id);
CREATE INDEX IF NOT EXISTS idx_paper_tags_tag ON paper_tags(tag_id);
