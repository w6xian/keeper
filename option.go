package keeper

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
func WithDogWatcher(watcher IWatcher) DogOption {
	return func(d *Dog) {
		d.Watcher = watcher
	}
}
