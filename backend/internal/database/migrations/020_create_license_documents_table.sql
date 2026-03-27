-- Migration: Create license_documents table for linking SharePoint docs to license records
-- Phase 3: Document Linking — files stay on SharePoint, portal stores URLs

CREATE TABLE IF NOT EXISTS license_documents (
    id SERIAL PRIMARY KEY,
    license_id INTEGER NOT NULL REFERENCES state_licenses(id) ON DELETE CASCADE,
    document_name VARCHAR(255) NOT NULL,
    document_url TEXT NOT NULL,
    document_type VARCHAR(50) DEFAULT 'other',
    sharepoint_item_id VARCHAR(255),
    folder_path TEXT,
    linked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    linked_by INTEGER REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_license_documents_license_id ON license_documents(license_id);

COMMENT ON TABLE license_documents IS 'Links SharePoint document URLs to license records. Files remain on SharePoint — only URLs stored here.';
