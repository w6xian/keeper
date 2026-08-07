package fsm

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

const (
	CMDSET = "SET"
	CMDDEL = "DEL"
)

type badgerFSM struct {
	db *badger.DB
}

func (b badgerFSM) Set(bucket, key string, value []byte) error {
	if len(value) <= 0 {
		return nil
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (b badgerFSM) Get(bucket, key string) ([]byte, error) {
	var data []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		data, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})

	return data, err
}

func (b badgerFSM) Del(bucket, key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b badgerFSM) Close() error {
	if err := b.db.Close(); err != nil {
		return err
	}
	return nil
}

func NewBadger(dbDir string) (IFSM, error) {
	f := filepath.Join(dbDir, "cache.db")
	opts := badger.DefaultOptions(f)
	badgerDB, err := badger.Open(opts)
	if err != nil {
		err = os.WriteFile(filepath.Join(dbDir, "badger_open_error.log"), []byte(err.Error()), 0644)
		return nil, err
	}
	return &badgerFSM{
		db: badgerDB,
	}, nil
}
func (b badgerFSM) String() string {
	return "badger db"
}
