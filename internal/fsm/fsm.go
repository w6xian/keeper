package fsm

type IFSM interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Del(key string) error
}
