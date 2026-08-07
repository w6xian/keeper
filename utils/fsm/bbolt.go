package fsm

import (
	"fmt"
	"path/filepath"
	"strings"

	bolt "go.etcd.io/bbolt"
)

func NewBolt(dbDir string) (IFSM, error) {
	f := filepath.Join(dbDir, "bolt.db")
	db, err := bolt.Open(f, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &bboltFSM{
		db: db,
	}, nil
}

type bboltFSM struct {
	db     *bolt.DB
	bucket []byte
}

func (b bboltFSM) Set(bucket, key string, value []byte) error {
	if len(value) <= 0 {
		return nil
	}
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return nil
	}
	return b.db.Update(func(txn *bolt.Tx) error {
		bk, err := txn.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return bk.Put([]byte(key), value)
	})
}
func (b bboltFSM) Get(bucket, key string) ([]byte, error) {
	var data []byte
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return nil, nil
	}
	err := b.db.View(func(txn *bolt.Tx) error {
		bk := txn.Bucket([]byte(bucket))
		if bk == nil {
			return nil
		}
		d := bk.Get([]byte(key))
		if d == nil {
			return nil
		}
		data = make([]byte, len(d))
		copy(data, d)
		return nil
	})
	return data, err
}
func (b bboltFSM) Del(bucket, key string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return nil
	}
	return b.db.Update(func(txn *bolt.Tx) error {
		bk := txn.Bucket([]byte(bucket))
		if bk == nil {
			return nil
		}
		return bk.Delete([]byte(key))
	})
}

func (b bboltFSM) Close() error {
	if err := b.db.Close(); err != nil {
		return err
	}
	return nil
}
func (b bboltFSM) String() string {
	return fmt.Sprintf("bolt db: %s", b.db.Path())
}
