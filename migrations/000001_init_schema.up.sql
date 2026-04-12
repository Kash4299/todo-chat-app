-- Cài đặt extension để tự động generate UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==========================================
-- BẢNG 1: USERS (Người dùng)
-- ==========================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth0_id VARCHAR(255) UNIQUE NOT NULL, -- ID trả về từ Auth0
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- BẢNG 2: TASKS (Công việc / Phòng Chat)
-- ==========================================
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'TODO', -- TODO, IN_PROGRESS, DONE
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index để tìm kiếm nhanh các task do 1 user tạo hoặc theo trạng thái
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_status ON tasks(status);

-- ==========================================
-- BẢNG 3: TASK_MEMBERS (Thành viên trong Task/Room)
-- ==========================================
CREATE TABLE task_members (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'MEMBER', -- OWNER, MEMBER
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Composite Primary Key: Đảm bảo 1 user không bị add 2 lần vào 1 task
    PRIMARY KEY (task_id, user_id) 
);

-- Index để query nhanh: "Hiển thị tất cả các task mà user này đang tham gia"
CREATE INDEX idx_task_members_user_id ON task_members(user_id);

-- ==========================================
-- BẢNG 4: MESSAGES (Đã nâng cấp)
-- ==========================================
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Phân loại tin nhắn: TEXT (chữ), FILE (ảnh/tài liệu), SYSTEM (thông báo hệ thống)
    message_type VARCHAR(20) DEFAULT 'TEXT', 
    
    content TEXT, -- Có thể rỗng nếu chỉ gửi file
    
    -- Lưu trữ nhiều file cùng lúc bằng mảng JSON
    -- Ví dụ: [{"url": "s3.aws.com/file1.png", "name": "bug.png", "size": 1024, "mimetype": "image/png"}]
    attachments JSONB DEFAULT '[]'::jsonb, 
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- ======================================
    -- TỐI ƯU HÓA TÌM KIẾM (FULL-TEXT SEARCH)
    -- ======================================
    -- Cột này tự động sinh ra các "từ khóa" từ cột content để search
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content, ''))) STORED
);

-- Index 1: Dùng để load lịch sử chat nhanh nhất (Mặc định)
CREATE INDEX idx_messages_task_id_created_at ON messages(task_id, created_at DESC);

-- Index 2: GIN INDEX - Vũ khí bí mật để tìm kiếm văn bản siêu tốc
CREATE INDEX idx_messages_search ON messages USING GIN(search_vector);

-- Index 3: Index cho JSONB (Tùy chọn) - Đề phòng sau này bạn muốn lọc "Chỉ lấy tin nhắn có hình ảnh"
CREATE INDEX idx_messages_attachments ON messages USING GIN(attachments);
