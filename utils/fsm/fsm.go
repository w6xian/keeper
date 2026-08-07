package fsm

type IFSM interface {
	Set(bucket, key string, value []byte) error
	Get(bucket, key string) ([]byte, error)
	Del(bucket, key string) error
	Close() error
	String() string
}

func NewFSM(t string, dbDir string) (IFSM, error) {
	switch t {
	case "badger":
		return NewBadger(dbDir)
	}
	return NewBolt(dbDir)
}
