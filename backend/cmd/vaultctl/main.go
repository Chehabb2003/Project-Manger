package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"project-crypto/internal/storage"
	"project-crypto/internal/vault"
	"time"
)

func main() {
	// ---- create ----
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	createVaultPath := createCmd.String("vault", "./main.vlt", "path to vault file")
	createMongoURI := createCmd.String("mongo", "", "MongoDB URI (optional)")
	createDB := createCmd.String("db", "vaultdb", "Mongo database name")
	createColl := createCmd.String("coll", "blobs", "Mongo collection name")

	// ---- add ----
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	addVaultPath := addCmd.String("vault", "./main.vlt", "path to vault file")
	site := addCmd.String("site", "", "site name")
	user := addCmd.String("user", "", "username")
	pass := addCmd.String("pass", "", "password or gen:N to generate N chars")
	addMongoURI := addCmd.String("mongo", "", "MongoDB URI (optional)")
	addDB := addCmd.String("db", "vaultdb", "Mongo database name")
	addColl := addCmd.String("coll", "blobs", "Mongo collection name")

	// ---- get ----
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getVaultPath := getCmd.String("vault", "./main.vlt", "path to vault file")
	getID := getCmd.String("id", "", "item id")
	getMongoURI := getCmd.String("mongo", "", "MongoDB URI (optional)")
	getDB := getCmd.String("db", "vaultdb", "Mongo DB")
	getColl := getCmd.String("coll", "blobs", "Mongo collection")

	// ---- list ----
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listVaultPath := listCmd.String("vault", "./main.vlt", "path to vault file")
	listType := listCmd.String("type", "", "filter by type (e.g. login)")
	listMongoURI := listCmd.String("mongo", "", "MongoDB URI (optional)")
	listDB := listCmd.String("db", "vaultdb", "Mongo DB")
	listColl := listCmd.String("coll", "blobs", "Mongo collection")

	// ---- setpass ----
	setCmd := flag.NewFlagSet("setpass", flag.ExitOnError)
	setVaultPath := setCmd.String("vault", "./main.vlt", "path to vault file")
	setID := setCmd.String("id", "", "item id")
	setPass := setCmd.String("pass", "", "new password or gen:N")
	setMongoURI := setCmd.String("mongo", "", "MongoDB URI (optional)")
	setDB := setCmd.String("db", "vaultdb", "Mongo DB")
	setColl := setCmd.String("coll", "blobs", "Mongo collection")

	// ---- delete ----
	delCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	delVaultPath := delCmd.String("vault", "./main.vlt", "path to vault file")
	delID := delCmd.String("id", "", "item id")
	delMongoURI := delCmd.String("mongo", "", "MongoDB URI (optional)")
	delDB := delCmd.String("db", "vaultdb", "Mongo DB")
	delColl := delCmd.String("coll", "blobs", "Mongo collection")

	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "create":
		_ = createCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*createVaultPath, *createMongoURI, *createDB, *createColl)
		dieIf(err)
		dieIf(createVaultWithStore(*createVaultPath, blobStore, metaStore))

	case "add":
		_ = addCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*addVaultPath, *addMongoURI, *addDB, *addColl)
		dieIf(err)
		dieIf(addItemWithStore(*addVaultPath, *site, *user, *pass, blobStore, metaStore))

	case "get":
		_ = getCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*getVaultPath, *getMongoURI, *getDB, *getColl)
		dieIf(err)
		dieIf(cmdGet(*getVaultPath, *getID, blobStore, metaStore))

	case "list":
		_ = listCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*listVaultPath, *listMongoURI, *listDB, *listColl)
		dieIf(err)
		dieIf(cmdList(*listVaultPath, *listType, blobStore, metaStore))

	case "setpass":
		_ = setCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*setVaultPath, *setMongoURI, *setDB, *setColl)
		dieIf(err)
		dieIf(cmdSetPass(*setVaultPath, *setID, *setPass, blobStore, metaStore))

	case "delete":
		_ = delCmd.Parse(os.Args[2:])
		blobStore, metaStore, err := buildStore(*delVaultPath, *delMongoURI, *delDB, *delColl)
		dieIf(err)
		dieIf(cmdDelete(*delVaultPath, *delID, blobStore, metaStore))

	default:
		usage()
	}
}

// ============ Helper Functions ============

func usage() {
	fmt.Print(`vaultctl commands:

  create  --vault path [--mongo URI --db vaultdb --coll blobs]
  add     --vault path --site example.com --user alice --pass gen:20 [--mongo URI --db vaultdb --coll blobs]
  get     --vault path --id <ITEM_ID> [--mongo URI --db vaultdb --coll blobs]
  list    --vault path [--type login] [--mongo URI --db vaultdb --coll blobs]
  setpass --vault path --id <ITEM_ID> --pass <new|gen:N> [--mongo URI --db vaultdb --coll blobs]
  delete  --vault path --id <ITEM_ID> [--mongo URI --db vaultdb --coll blobs]

Examples:
  vaultctl create --vault ./main.vlt
  vaultctl add --vault ./main.vlt --site example.com --user ahmad --pass gen:16
  vaultctl get --vault ./main.vlt --id 1761753230653491299
`)
}

func buildStore(vaultPath, mongoURI, db, coll string) (storage.BlobStore, storage.MetaStore, error) {
	if mongoURI == "" {
		fs := storage.NewFileBlobStore("." + filepathBase(vaultPath) + ".blobs")
		return fs, nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	blobStore, err := storage.NewMongoBlobStore(ctx, mongoURI, db, coll)
	if err != nil {
		return nil, nil, err
	}

	metaStore, err := storage.NewMongoMetaStore(ctx, mongoURI, db, "meta")
	if err != nil {
		return nil, nil, err
	}

	return blobStore, metaStore, nil
}

func createVaultWithStore(path string, blobs storage.BlobStore, meta storage.MetaStore) error {
	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Create(ctx, master); err != nil {
		return err
	}
	fmt.Println("Vault created:", path)
	return nil
}

func addItemWithStore(path, site, user, pass string, blobs storage.BlobStore, meta storage.MetaStore) error {
	if site == "" || user == "" || pass == "" {
		return errors.New("site/user/pass required")
	}
	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Unlock(ctx, master); err != nil {
		return err
	}
	defer vlt.Lock()

	if len(pass) > 4 && pass[:4] == "gen:" {
		var n int
		_, _ = fmt.Sscanf(pass, "gen:%d", &n)
		if n <= 0 {
			n = 20
		}
		pass = genPassword(n)
	}

	item := vault.Item{
		Type: "login",
		Fields: map[string]string{
			"site":     site,
			"username": user,
			"password": pass,
		},
	}

	id, err := vlt.AddItem(ctx, item)
	if err != nil {
		return err
	}
	fmt.Println("Added item id:", id)

	items, _ := vlt.List(ctx, vault.Query{})
	b, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdGet(path, id string, blobs storage.BlobStore, meta storage.MetaStore) error {
	if id == "" {
		return errors.New("--id required")
	}
	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Unlock(ctx, master); err != nil {
		return err
	}
	defer vlt.Lock()

	it, err := vlt.GetItem(ctx, id)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(it, "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdList(path, typ string, blobs storage.BlobStore, meta storage.MetaStore) error {
	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Unlock(ctx, master); err != nil {
		return err
	}
	defer vlt.Lock()

	metas, err := vlt.List(ctx, vault.Query{Type: typ})
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(metas, "", "  ")
	fmt.Println(string(b))
	return nil
}

// ============ Utilities ============

func promptSecret(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	br := bufio.NewReader(os.Stdin)
	master, err := br.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(master) > 0 && master[len(master)-1] == '\n' {
		master = master[:len(master)-1]
	}
	return master, nil
}

func genPassword(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = alphabet[i%len(alphabet)]
		}
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func filepathBase(p string) string {
	if p == "" {
		return p
	}
	last := -1
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			last = i
		}
	}
	if last == -1 {
		return p
	}
	if last == len(p)-1 {
		return ""
	}
	return p[last+1:]
}

// Change the password field of an item (keeps other fields intact).
func cmdSetPass(path, id, pass string, blobs storage.BlobStore, meta storage.MetaStore) error {
	if id == "" {
		return errors.New("--id required")
	}
	if pass == "" {
		return errors.New("--pass required (or gen:N)")
	}

	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Unlock(ctx, master); err != nil {
		return err
	}
	defer vlt.Lock()

	// fetch the current item
	curr, err := vlt.GetItem(ctx, id)
	if err != nil {
		return err
	}

	// auto-generate if requested (e.g., gen:24)
	if len(pass) > 4 && pass[:4] == "gen:" {
		var n int
		_, _ = fmt.Sscanf(pass, "gen:%d", &n)
		if n <= 0 {
			n = 20
		}
		pass = genPassword(n)
	}

	if curr.Fields == nil {
		curr.Fields = map[string]string{}
	}
	curr.Fields["password"] = pass

	// keep same type; only update fields
	upd := vault.Item{Type: curr.Type, Fields: curr.Fields}
	if err := vlt.UpdateItem(ctx, id, upd); err != nil {
		return err
	}

	fmt.Println("Password updated for id:", id)
	return nil
}

// Delete an item (removes meta, blob and KD entry).
func cmdDelete(path, id string, blobs storage.BlobStore, meta storage.MetaStore) error {
	if id == "" {
		return errors.New("--id required")
	}

	master, err := promptSecret("Master password: ")
	if err != nil {
		return err
	}
	defer zero(master)

	vlt := vault.NewWithStores(path, blobs, meta)
	ctx := context.Background()
	if err := vlt.Unlock(ctx, master); err != nil {
		return err
	}
	defer vlt.Lock()

	if err := vlt.DeleteItem(ctx, id); err != nil {
		return err
	}
	fmt.Println("Deleted item id:", id)
	return nil
}
