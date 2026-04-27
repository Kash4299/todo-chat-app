# E2EE Encryption Plan — Messenger-style (Signal Protocol)

> Approach: Facebook Messenger (Signal Protocol + libsignal)
> Priority: P3 — implement sau khi core app ổn định (Sprint 14+)
> Prerequisite: Frontend phải hoàn thiện trước khi bắt đầu E2EE

---

## Tổng quan

Facebook Messenger dùng **Signal Protocol** cho E2EE (adopt từ 2016 cho Secret Conversations, default cho tất cả chats từ 2023). Signal Protocol là industry standard, được WhatsApp, Google Messages, Wire dùng.

**Nguyên tắc cốt lõi**: Server chỉ relay ciphertext. Private key sinh ra trên client, không bao giờ rời client.

---

## Signal Protocol — Cách hoạt động

### 1. Key Types

```
Mỗi user có các key sau (tất cả sinh ra trên client):

Identity Key Pair (IK)     — long-term, đại diện cho identity của user
Signed Prekey Pair (SPK)   — medium-term, rotate mỗi 7-30 ngày, signed bởi IK
One-Time Prekeys (OPKs)    — single-use, batch upload lên server (~100 keys)

Public keys → upload lên server
Private keys → chỉ tồn tại trên device, không bao giờ gửi đi
```

### 2. Initial Key Exchange — X3DH (Extended Triple Diffie-Hellman)

```
Alice muốn nhắn tin lần đầu cho Bob:

1. Alice fetch Bob's key bundle từ server:
   { IK_bob_public, SPK_bob_public, OPK_bob_public (một cái), SPK_signature }

2. Alice verify SPK_signature bằng IK_bob_public

3. Alice tính shared secret:
   DH1 = DH(IK_alice, SPK_bob)
   DH2 = DH(EK_alice, IK_bob)       ← EK là ephemeral key, gen mới cho session này
   DH3 = DH(EK_alice, SPK_bob)
   DH4 = DH(EK_alice, OPK_bob)      ← nếu OPK available
   master_secret = KDF(DH1 || DH2 || DH3 || DH4)

4. Alice dùng master_secret để encrypt message đầu tiên
5. Alice gửi: { IK_alice_public, EK_alice_public, OPK_id_used, ciphertext }

6. Bob nhận, tự tính lại master_secret với private keys của mình
   → Cùng ra một master_secret mà server không biết
```

### 3. Ongoing Messages — Double Ratchet Algorithm

```
Sau X3DH, mỗi message dùng key mới (forward secrecy + break-in recovery):

Symmetric Ratchet:  mỗi message → derive key mới từ key trước
Diffie-Hellman Ratchet: mỗi reply → exchange DH keys mới → reset chain

Nếu một message key bị compromise → chỉ message đó bị lộ
Nếu attacker compromise toàn bộ state → không đọc được messages trước đó (forward secrecy)
```

---

## Kiến trúc thay đổi

### Backend changes (Go)

**1. New table: `user_key_bundles`**
```sql
CREATE TABLE user_key_bundles (
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id        UUID NOT NULL DEFAULT uuid_generate_v4(),
    ik_public        BYTEA NOT NULL,          -- Identity Key public
    spk_public       BYTEA NOT NULL,          -- Signed Prekey public
    spk_signature    BYTEA NOT NULL,          -- SPK signed by IK
    spk_uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, device_id)
);
```

**2. New table: `user_one_time_prekeys`**
```sql
CREATE TABLE user_one_time_prekeys (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id  UUID NOT NULL,
    key_id     INT  NOT NULL,
    opk_public BYTEA NOT NULL,
    used_at    TIMESTAMPTZ,                   -- null = available
    UNIQUE (user_id, device_id, key_id)
);
```

**3. Message table changes**
```sql
-- Thêm vào messages table:
ALTER TABLE messages ADD COLUMN is_encrypted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN sender_device_id UUID;
-- content column: lưu ciphertext (base64) thay vì plaintext khi is_encrypted = TRUE
-- search_vector: disabled cho encrypted messages (server không thể search ciphertext)
```

**4. New API endpoints**
```
POST   /users/me/keys              — Upload key bundle (IK + SPK + OPKs batch)
GET    /users/:id/keys             — Fetch key bundle của user khác (để init session)
POST   /users/me/keys/prekeys      — Replenish one-time prekeys khi server còn < 10
GET    /users/me/keys/prekeys/count — Client check xem cần replenish không
```

**5. Message endpoint changes**
- Message content không validate plaintext nữa khi `is_encrypted = true`
- Full-text search disabled cho encrypted channels
- Server không decrypt để moderate/search

### Frontend changes (React/TypeScript)

```
lib/signal/
├── keystore.ts          — Lưu private keys trong IndexedDB (encrypted bởi device password)
├── session-manager.ts   — Quản lý Signal sessions với từng contact
├── x3dh.ts              — Initial key exchange
├── double-ratchet.ts    — Ongoing message encryption/decryption
└── crypto-utils.ts      — Key generation, serialization

hooks/
├── useEncryptedChat.ts  — Hook wrap WebSocket, auto encrypt/decrypt messages
└── useKeySetup.ts       — Khởi tạo keys lần đầu, upload lên server
```

**Dùng library**: `@signalapp/libsignal-client` (official Signal library, WebAssembly build cho browser) — không tự implement crypto.

---

## Migration Strategy

Không thể bật E2EE cho existing messages. Approach:

```
Phase 1 — Opt-in per channel:
  - Tạo "Encrypted Channel" type mới (thêm vào channel.type ENUM)
  - Channels mới có thể chọn encrypted
  - Existing channels giữ nguyên plaintext

Phase 2 — Default cho DM:
  - Tất cả DM mới tự động encrypted
  - Existing DMs: không migrate, hiển thị "Messages before [date] are not encrypted"

Phase 3 — Default cho tất cả:
  - Group channels: phức tạp hơn (Sender Keys — mỗi member nhận copy key)
  - Xem xét sau Phase 2 ổn định
```

---

## Sender Keys cho Group Chat

DM thì 1-1. Group chat phức tạp hơn — Signal dùng **Sender Keys**:

```
Alice muốn gửi vào group (Alice, Bob, Carol):

1. Alice generate một Sender Key riêng cho group này
2. Alice gửi Sender Key cho Bob và Carol (qua X3DH/Double Ratchet 1-1)
3. Alice encrypt message một lần bằng Sender Key
4. Bob và Carol decrypt bằng Sender Key đã nhận

→ Server nhận một ciphertext duy nhất, fan-out đến tất cả members
→ Hiệu quả hơn encrypt N lần cho N members
```

---

## Điều không thể làm khi có E2EE

| Feature | Lý do |
|---------|-------|
| Full-text search messages | Server không thể đọc ciphertext |
| Message moderation | Server không biết nội dung |
| AI features (summarize, translate) | Cần plaintext |
| Server-side backup | Backup là ciphertext, vô dụng nếu mất key |

Messenger giải quyết search bằng **client-side search** — index lưu trên device.

---

## Sprint Plan (sau Sprint 13)

| Sprint | Task |
|--------|------|
| Sprint 14 | Schema migrations (key_bundles, one_time_prekeys), Key management API |
| Sprint 15 | Frontend: libsignal setup, keystore, X3DH implementation |
| Sprint 16 | Frontend: Double Ratchet integration, DM E2EE opt-in |
| Sprint 17 | Group E2EE (Sender Keys), migration UX |
| Sprint 18 | Security audit, penetration testing |

> **Quan trọng**: Sprint 18 (security audit) là bắt buộc trước khi ship E2EE ra production. Không audit = không ship.

---

## Libraries

| Layer | Library | Lý do |
|-------|---------|-------|
| Backend (Go) | Không cần signal lib — chỉ relay | Server không decrypt |
| Frontend (JS/TS) | `@signalapp/libsignal-client` | Official, battle-tested, WASM build |
| Key storage (browser) | IndexedDB + WebCrypto API | Private keys không rời browser |
| Key storage (mobile) | Secure Enclave / Android Keystore | Hardware-backed security |

**Không tự implement bất kỳ crypto primitive nào.** Luôn dùng library đã được audit.
