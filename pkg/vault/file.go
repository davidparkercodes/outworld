package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const fileMagic = "OUTWORLD-VAULT-1\n"

var ErrWrongKey = errors.New("vault: cannot decrypt vault header (wrong key or corrupted file)")

type FileBackend struct {
	path    string
	aead    cipher.AEAD
	mu      sync.Mutex
	offset  int64
	frames  int
	dirty   bool
	release func() error
	fresh   []byte
}

type header struct {
	Version int    `json:"v"`
	Secret  []byte `json:"secret"`
}

func NewFileBackend(path string, key []byte) (*FileBackend, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("vault: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &FileBackend{path: path, aead: aead}, nil
}

func NewFileBackendWithSecret(path string, key, secret []byte) (*FileBackend, error) {
	if len(secret) != 32 {
		return nil, fmt.Errorf("vault: secret must be 32 bytes, got %d", len(secret))
	}
	f, err := NewFileBackend(path, key)
	if err != nil {
		return nil, err
	}
	f.fresh = append([]byte(nil), secret...)
	return f, nil
}

func OpenFile(path string, key []byte) (*Vault, error) {
	b, err := NewFileBackend(path, key)
	if err != nil {
		return nil, err
	}
	return Open(b)
}

func (f *FileBackend) aad(i int) []byte { return []byte("outworld-vault|" + strconv.Itoa(i)) }

func (f *FileBackend) seal(i int, plain []byte) []byte {
	nonce := make([]byte, f.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}
	ct := f.aead.Seal(nil, nonce, plain, f.aad(i))
	frame := make([]byte, 4, 4+len(nonce)+len(ct))
	binary.BigEndian.PutUint32(frame, uint32(len(nonce)+len(ct)))
	frame = append(frame, nonce...)
	return append(frame, ct...)
}

func (f *FileBackend) open(i int, body []byte) ([]byte, error) {
	ns := f.aead.NonceSize()
	if len(body) < ns {
		return nil, errors.New("short frame")
	}
	return f.aead.Open(nil, body[:ns], body[ns:], f.aad(i))
}

func (f *FileBackend) create() ([]byte, error) {
	secret := make([]byte, 32)
	if f.fresh != nil {
		copy(secret, f.fresh)
	} else if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	h, err := json.Marshal(header{Version: 1, Secret: secret})
	if err != nil {
		return nil, err
	}
	data := append([]byte(fileMagic), f.seal(0, h)...)
	if err := writeDurable(f.path, data); err != nil {
		return nil, err
	}
	f.offset = int64(len(data))
	f.frames = 1
	return secret, nil
}

func writeDurable(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("vault: create: %w", err)
	}
	name := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(name)
		}
	}()
	if err = tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("vault: create: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("vault: create: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("vault: sync: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("vault: create: %w", err)
	}
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("vault: create: %w", err)
	}
	return syncDir(dir)
}

func (f *FileBackend) Load() (secret []byte, recs []Record, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if singleProcess && f.release == nil {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
			return nil, nil, fmt.Errorf("vault: create dir: %w", err)
		}
		release, lerr := acquireExclusive(f.path)
		if lerr != nil {
			return nil, nil, lerr
		}
		f.release = release
		defer func() {
			if err != nil && f.release != nil {
				f.release()
				f.release = nil
			}
		}()
	}
	return f.load()
}

func (f *FileBackend) load() ([]byte, []Record, error) {
	fh, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
			return nil, nil, fmt.Errorf("vault: create dir: %w", err)
		}
		var unlock func()
		if unlock, err = lockPath(f.path); err != nil {
			return nil, nil, err
		}
		defer unlock()
		if _, statErr := os.Stat(f.path); errors.Is(statErr, os.ErrNotExist) {
			s, err := f.create()
			return s, nil, err
		}
		fh, err = os.Open(f.path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("vault: open: %w", err)
	}
	defer fh.Close()
	magic := make([]byte, len(fileMagic))
	if _, err := io.ReadFull(fh, magic); err != nil || string(magic) != fileMagic {
		return nil, nil, errors.New("vault: not an outworld vault file")
	}
	plains, offset, frames, err := f.readFrames(fh, int64(len(fileMagic)), 0)
	if err != nil {
		return nil, nil, err
	}
	if len(plains) == 0 {
		return nil, nil, errors.New("vault: missing header")
	}
	var h header
	if err := json.Unmarshal(plains[0], &h); err != nil || len(h.Secret) == 0 {
		return nil, nil, errors.New("vault: bad header")
	}
	recs, err := decodeRecords(plains[1:])
	if err != nil {
		return nil, nil, err
	}
	f.offset, f.frames = offset, frames
	return h.Secret, recs, nil
}

func (f *FileBackend) readFrames(fh *os.File, offset int64, frames int) ([][]byte, int64, int, error) {
	if _, err := fh.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, 0, err
	}
	var out [][]byte
	lenBuf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(fh, lenBuf); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return out, offset, frames, nil
			}
			return nil, 0, 0, err
		}
		n := binary.BigEndian.Uint32(lenBuf)
		if n > 1<<24 {
			return nil, 0, 0, errors.New("vault: corrupt frame length")
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(fh, body); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return out, offset, frames, nil
			}
			return nil, 0, 0, err
		}
		plain, err := f.open(frames, body)
		if err != nil {
			if frames == 0 {
				return nil, 0, 0, ErrWrongKey
			}
			if torn, terr := tornTail(fh, offset); terr != nil {
				return nil, 0, 0, terr
			} else if torn {
				return out, offset, frames, nil
			}
			return nil, 0, 0, fmt.Errorf("vault: frame %d failed authentication", frames)
		}
		out = append(out, plain)
		frames++
		offset += int64(4 + n)
	}
}

func tornTail(fh *os.File, from int64) (bool, error) {
	if _, err := fh.Seek(from, io.SeekStart); err != nil {
		return false, err
	}
	lenBuf := make([]byte, 4)
	for {
		k, err := io.ReadFull(fh, lenBuf)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return allZero(lenBuf[:k]), nil
		}
		if err != nil {
			return false, err
		}
		n := binary.BigEndian.Uint32(lenBuf)
		if n > 1<<24 {
			return false, nil
		}
		body := make([]byte, n)
		k, err = io.ReadFull(fh, body)
		if !allZero(body[:k]) {
			return false, nil
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
	}
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

func decodeRecords(plains [][]byte) ([]Record, error) {
	recs := make([]Record, 0, len(plains))
	for _, p := range plains {
		var r Record
		if err := json.Unmarshal(p, &r); err != nil {
			return nil, fmt.Errorf("vault: bad record: %w", err)
		}
		recs = append(recs, r)
	}
	return recs, nil
}

func (f *FileBackend) Update(fn func([]Record) ([]Record, error)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	unlock, err := lockPath(f.path)
	if err != nil {
		return err
	}
	defer unlock()
	fh, err := os.OpenFile(f.path, os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("vault: open for update: %w", err)
	}
	defer fh.Close()
	plains, offset, frames, err := f.readFrames(fh, f.offset, f.frames)
	if err != nil {
		return err
	}
	newer, err := decodeRecords(plains)
	if err != nil {
		return err
	}
	f.offset, f.frames = offset, frames
	add, err := fn(newer)
	if err != nil {
		return err
	}
	if len(add) == 0 {
		if f.dirty {
			if err := syncFile(fh); err != nil {
				return fmt.Errorf("vault: sync: %w", err)
			}
			f.dirty = false
		}
		return nil
	}
	if _, err := fh.Seek(f.offset, io.SeekStart); err != nil {
		return err
	}
	if err := fh.Truncate(f.offset); err != nil {
		return err
	}
	var buf []byte
	for i, r := range add {
		p, err := json.Marshal(r)
		if err != nil {
			return err
		}
		buf = append(buf, f.seal(f.frames+i, p)...)
	}
	f.dirty = true
	if _, err := fh.Write(buf); err != nil {
		return fmt.Errorf("vault: write: %w", err)
	}
	if err := syncFile(fh); err != nil {
		return fmt.Errorf("vault: sync: %w", err)
	}
	f.dirty = false
	f.offset += int64(len(buf))
	f.frames += len(add)
	return nil
}

var syncFile = (*os.File).Sync

func (f *FileBackend) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.release == nil {
		return nil
	}
	err := f.release()
	f.release = nil
	return err
}

func ParseKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty key")
	}
	if len(s) == 64 {
		if k, err := hex.DecodeString(s); err == nil {
			return k, nil
		}
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if k, err := enc.DecodeString(s); err == nil && len(k) == 32 {
			return k, nil
		}
	}
	return nil, errors.New("key must be 32 bytes encoded as 64 hex chars or base64")
}

func LoadKey(envName, keyFile string) ([]byte, error) {
	if envName != "" {
		if v := os.Getenv(envName); strings.TrimSpace(v) != "" {
			k, err := ParseKey(v)
			if err != nil {
				return nil, fmt.Errorf("vault key from $%s: %w", envName, err)
			}
			return k, nil
		}
	}
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("vault key file: %w", err)
		}
		k, err := ParseKey(string(data))
		if err != nil {
			return nil, fmt.Errorf("vault key file %s: %w", keyFile, err)
		}
		return k, nil
	}
	where := "a key file"
	if envName != "" {
		where = "$" + envName + " or a key file"
	}
	return nil, fmt.Errorf("vault key required: set %s (generate one with `outworld keygen`); refusing to run without encryption", where)
}

func GenerateKey() (string, error) {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return "", err
	}
	return hex.EncodeToString(k), nil
}
