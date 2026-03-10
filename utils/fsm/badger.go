package fsm

import (
	"github.com/dgraph-io/badger/v4"
)

const (
	CMDSET = "SET"
	CMDDEL = "DEL"
)

type badgerFSM struct {
	db *badger.DB
}

func (b badgerFSM) Set(key string, value []byte) error {
	if len(value) <= 0 {
		return nil
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (b badgerFSM) Get(key string) ([]byte, error) {
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

func (b badgerFSM) Del(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func NewBadger(badgerDB *badger.DB) IFSM {
	return &badgerFSM{
		db: badgerDB,
	}
}
