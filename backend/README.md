# project-crypto — Minimal Zero-Knowledge Password Vault

A small learning project that stores **encrypted items** (logins) using a local file
or a MongoDB collection. Cleartext never leaves your machine: only ciphertext blobs
(and minimal metadata, if you enable it) are stored.

---

## Requirements

- Go 1.22+
- (Optional) MongoDB Atlas/Server connection string (SRV), e.g.  
  `mongodb+srv://<user>:<pass>@<cluster>/<db>?retryWrites=true&w=majority&ssl=true`

> **Master password:** You will be prompted on each command that needs access.  
> Keep it safe; if lost, items can’t be decrypted.

---

## Layout

cmd/
vaultctl/ # CLI tool (create/add/get/list/setpass/delete)
internal/
crypto/ # crypto primitives & helpers
storage/ # file & mongo blob/meta stores
vault/ # vault logic (header/KDF/KD/items)
main.vlt # your vault header (created at runtime)

sql
Copy code

---

## Quick Start

### Windows (CMD / PowerShell – *use one line per command*)
```bat
REM Create a local vault (files on disk)
go run .\cmd\vaultctl create --vault .\main.vlt

REM Add a login (local)
go run .\cmd\vaultctl add --vault .\main.vlt --site example.com --user USER --pass gen:XX

REM List items (local in-memory index; typically empty across separate runs)
go run .\cmd\vaultctl list --vault .\main.vlt

REM Get/decrypt one item
go run .\cmd\vaultctl get --vault .\main.vlt --id <ITEM_ID>


Linux / macOS
bash
Copy code
# Create a local vault (files on disk)
go run ./cmd/vaultctl create --vault ./main.vlt

# Add a login (local)
go run ./cmd/vaultctl add --vault ./main.vlt --site example.com --user ahmad --pass gen:16

# List items (local in-memory index; typically empty across separate runs)
go run ./cmd/vaultctl list --vault ./main.vlt

# Get/decrypt one item
go run ./cmd/vaultctl get --vault ./main.vlt --id <ITEM_ID>
Using MongoDB (Ciphertext blobs in Mongo + Metadata in Mongo)
Replace the URI with your Atlas URI. Keep ssl=true for SRV.

Note: You must allow your client IP in Atlas Network Access.

Windows (CMD / PowerShell)
bat
Copy code
REM Create vault; store blobs in Mongo (DB: vaultdb, coll: blobs)
go run .\cmd\vaultctl create --vault .\main.vlt --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

REM Add a login (ciphertext blob goes to Mongo)
go run .\cmd\vaultctl add --vault .\main.vlt --site example.com --user ahmad --pass gen:16 --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

REM List items (reads metadata from Mongo `vaultdb.meta`)
go run .\cmd\vaultctl list --vault .\main.vlt --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

REM Get/decrypt an item by id (downloads ciphertext blob from Mongo)
go run .\cmd\vaultctl get --vault .\main.vlt --id <ITEM_ID> --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs
Linux / macOS
bash
Copy code
# Create vault; store blobs in Mongo (DB: vaultdb, coll: blobs)
go run ./cmd/vaultctl create --vault ./main.vlt --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

# Add a login (ciphertext blob goes to Mongo)
go run ./cmd/vaultctl add --vault ./main.vlt --site example.com --user ahmad --pass gen:16 --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

# List items (reads metadata from Mongo `vaultdb.meta`)
go run ./cmd/vaultctl list --vault ./main.vlt --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs

# Get/decrypt an item by id
go run ./cmd/vaultctl get --vault ./main.vlt --id <ITEM_ID> --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs
Update, Delete, and Custom Passwords
Change only the password field of an item
bat
Copy code
REM Windows (choose your own password value)
go run .\cmd\vaultctl setpass --vault .\main.vlt --id <ITEM_ID> --pass MyStr0ngPass!! --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs
bash
Copy code
# Linux/macOS (auto-generate 24 chars instead)
go run ./cmd/vaultctl setpass --vault ./main.vlt --id <ITEM_ID> --pass gen:24 --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs
Delete an item
bat
Copy code
go run .\cmd\vaultctl delete --vault .\main.vlt --id <ITEM_ID> --mongo "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true" --db vaultdb --coll blobs
