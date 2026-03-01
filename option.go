package keeper

import "github.com/w6xian/keeper/internal/fsm"

type DogOption func(*Dog)

type DoorOption func(*Door)

type IWatcher interface {
}

func WithDogName(name string) DogOption {
	return func(d *Dog) {
		d.Name = name
	}
}

func WithDoorName(name string) DoorOption {
	return func(d *Door) {
		d.Name = name
	}
}

func WithFSMStore(fsmStore fsm.IFSM) DoorOption {
	return func(d *Door) {
		d.fsmStore = fsmStore
	}
}

func WithDogWatcher(watcher IWatcher) DogOption {
	return func(d *Dog) {
		d.Watcher = watcher
	}
}
