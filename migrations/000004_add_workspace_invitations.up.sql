CREATE TABLE IF not exists workspace_invitations (                                                                                                                                                                                                       
        id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),                                                                                   
        workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,                                                                                  
        invited_by   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,         -- user tạo invite                  
        email        varchar(255) NOT NULL,  -- email của người được invite                                                                                                                                                                                                          
        token        varchar(64) NOT NULL UNIQUE,     -- crypto random string                                                                                                                         
        expires_at   TIMESTAMPTZ NOT NULL,
        used_at      TIMESTAMPTZ
);

-- Covers ListPendingByWorkspace (workspace_id only) and FindByWorkspaceAndEmail (both columns)
-- via leftmost-prefix rule — replaces the need for a separate workspace_id-only index.
CREATE INDEX idx_workspace_invitations_workspace_email ON workspace_invitations(workspace_id, email);
CREATE INDEX idx_workspace_invitations_email ON workspace_invitations(email);