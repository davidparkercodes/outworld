# Using it as a library

```go
v, _ := vault.OpenFile("my.vault", key)
dict, _ := detect.LoadDictionary("people.csv")
eng, _ := outworld.New(outworld.Options{Vault: v, Policy: outworld.DefaultPolicy(), Dictionary: dict})

s := eng.NewSession("global")
masked, err := s.Mask(ctx, text)       // err = detection failure
if err := s.Err(); err != nil { ... }   // leak check (errors.Is(err, outworld.ErrLeak))
answer := s.Rehydrate(modelOutput)
r := s.NewStreamRehydrator(false)       // Push(chunk) / Flush() for streams
```

Packages: `pkg/detect` (detectors, dictionary, NER client, overlap resolution), `pkg/vault` (vault + `Backend` interface + encrypted file backend), `pkg/surrogate` (generators and name lists), `pkg/dateshift`, `pkg/entity`, `pkg/outworld` (the engine).

## Vault backends

`vault.Backend` is a small interface: `Load()` returns the vault secret and existing records; `Update(fn)` must give `fn` exclusive access, pass it records other writers added since the last call, and durably persist what `fn` returns. The bundled file backend implements this with `flock` and an append-only encrypted log, so on Linux, macOS and the BSDs several Outworld processes can share one vault file. On other platforms (Windows among them) a vault is single-process: the open vault holds an exclusive `<vault>.lock` file until it is closed, and a second process fails to open it with an error that names that file. A lock file left over from a crash must be deleted by hand, once no other Outworld process is running. Other stores can plug in by implementing the same interface.
