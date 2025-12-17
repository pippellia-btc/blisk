# Blisk

Blisk is a local database for storing [blossom blobs](https://github.com/hzrd149/blossom) on disk. It is designed for efficient, scalable, and deduplicated blob storage while maintaining metadata in sqlite. It's short for Blobs on Disk.

## Installation

```
go get github.com/pippellia-btc/blisk
```

## Features

- **Highly concurrent:** Blobs are stored on disk in directories based on the first three characters of their hash (e.g., `/blobs/abc/abcdef123...`). This reduces contention and allows up to 4096 concurrent writes all while maintaining a consistent state.

- **Deduplication:** Multiple uploads of the same blob are stored only once. Only when all uploaders delete a blob is it removed from disk, saving storage space.

- **Idempotent operations:** Saving or deleting the same blob multiple times with the same uploader is a no-op.

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

This sharding strategy ensures low contention and efficient filesystem operations.


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

```go
file, err := store.Load(ctx, hash)
if err != nil {
    log.Print(err)
}
defer file.Close()
```

### Listing uploader blobs

```go
uploader := "alice" // normally this would be a npub

hashes, err := store.Hashes(ctx, uploader)
if err != nil {
    log.Print(err)
}
fmt.Printf("Uploader %s has blobs %v", uploader, hashes)
```

### Deleting a blob
```go
deleter := "alice" // normally this would be a npub

err := store.Delete(ctx, hash, deleter)
if err != nil {
    log.Print(err)
}
```
