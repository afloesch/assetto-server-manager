package archiver

// assetType defines a simple interface so new asset types, or breaking changes
// to existing asset types can easily be added or fixed
type assetType interface {
	// Name returns the asset folder name
	Name() string
	// Folder returns the asset folder path relative to ac server install directory
	Folder() string
	// Data returns the glob pattern to the asset json file relative to the asset folder
	Data() string
}

// Car type archive
type Car struct{}

// Name the asset folder name
func (c Car) Name() string {
	return "cars"
}

// Folder returns the asset folder path relative to server install directory
func (c Car) Folder() string {
	return "content/cars"
}

// Data returns the glob pattern to the asset json file relative to the asset folder
func (c Car) Data() string {
	return "ui/ui_car.json"
}

// Track type archive
type Track struct{}

// Name the asset folder name
func (t Track) Name() string {
	return "tracks"
}

// Folder returns the asset folder path relative to server install directory
func (t Track) Folder() string {
	return "content/tracks"
}

// Data returns the glob pattern to the asset json file relative to the asset folder
func (t Track) Data() string {
	return "ui/meta_data.json"
}
