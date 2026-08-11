package keeper

import "log"

const (
	LevelDebug = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	LevelInfo  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	LevelError = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	LevelFatal = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	LevelNone  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	LevelAll   = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {

}

func InitLog() {
	log.SetFlags(LevelDebug)
}

type c struct {
	log.Logger
}
