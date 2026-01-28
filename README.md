# Blisk

Blisk is a local database for storing [blossom blobs](https://github.com/hzrd149/blossom) on disk. It is designed for efficient, scalable, and deduplicated blob storage while maintaining metadata in sqlite. It's short for Blobs on Disk.

[![Go Report Card](https://goreportcard.com/badge/github.com/pippellia-btc/blisk)](https://goreportcard.com/report/github.com/pippellia-btc/blisk)
[![Go Reference](https://pkg.go.dev/badge/github.com/pippellia-btc/blisk.svg)](https://pkg.go.dev/github.com/pippellia-btc/blisk)


## Installation

```
go get github.com/pippellia-btc/blisk
```

## Features

- **Highly concurrent:** Blobs are stored on disk in directories based on the first three characters of their hash (e.g., `/blobs/abc/abcdef123...`). This reduces contention and allows up to 4096 concurrent writes all while maintaining a consistent state.

- **Deduplication:** Multiple uploads of the same blob are stored only once. Only when all uploaders delete a blob is it removed from disk, saving storage space.

- **Idempotent operations:** Saving or deleting the same blob multiple times is a no-op if performed by the same entity.

## Limitations

Blisk `Store` is safe to be used by multiple goroutines within a single program. Multi-program writes are not safe and highly discouraged.

## Directory Structure

Blisk stores blobs on disk in a deterministic, sharded structure:

```
blobs/
├─ 000/
├─ 001/
├─ 002/
│ ...
├─ fff/
```

A blob with hash `abcdef123...` is stored in: `blobs/abc/abcdef123...`
This sharding strategy ensures low contention and allows for up to 4096 concurrent writes.

## Usage

### Creating a store

```go
store, err := blisk.New("/path/to/store")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### Saving a blob

Saving a blob to disk only if not already present, adding its hash to the uploads of the specified uploader.

```go
blob := []byte("hello world")
uploader := "alice" // normally this would be a npub

meta, err := store.Save(ctx, blob, uploader)
if err != nil {
    log.Print(err)
}
fmt.Printf("Saved blob with hash %v", meta.Hash)
```

### Loading a blob

Loading the blob's file associated with the provided hash.

```go
file, err := store.Load(ctx, hash)
if err != nil {
    log.Print(err)
}
defer file.Close()
```

### Fetching metadata

Fetching blob's metadata without loading it into memory.

```go
meta, err := store.Info(ctx, hash)
if err != nil {
    log.Print(err)
}
fmt.Printf("blob %s: size %d, mime %s", hash, meta.Size, meta.MIME)
```

### Listing uploads

Listing all hashes uploaded by an entity, for example a nostr pubkey.

```go
uploader := "alice" // normally this would be a npub

hashes, err := store.Hashes(ctx, uploader)
if err != nil {
    log.Print(err)
}
fmt.Printf("Uploader %s has blobs %v", uploader, hashes)
```

### Deleting a blob

Deleting a blob from the uploads of an entity. If no entity reference the blob, it is then deleted from disk.

```go
deleter := "alice" // normally this would be a npub

err := store.Delete(ctx, hash, deleter)
if err != nil {
    log.Print(err)
}
```
