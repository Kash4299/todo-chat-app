CREATE TABLE IF NOT EXISTS todos (
    id uuid default uuid_generate_v4() primary key,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    deleted_At timestamp with time zone,
    title varchar(255),
    status varchar(255),
    priority integer,
    description varchar(255),
    created_by uuid,
    assigned uuid 
);