package image

import (
	"os"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Image wraps a few methods about Image.
type Image interface { //nolint
	GetName() string
	GetSize() int64
	GetUser() string
	GetDistro() string
	GetID() string
	GetType() string
	GetHash() string
	UpdateHash() (string, error)

	Delete() error

	String() string
	Filepath() string
	Filename() string
}

// Load loads an Image.
func Load(name, user string) (Image, error) {
	if len(user) > 0 {
		return LoadUserImage(user, name)
	}
	return LoadSysImage(name)
}

// List lists all images which belong to a specific user, or system-wide type.
func List(user string) ([]Image, error) {
	if len(user) > 0 {
		return ListUserImages(user)
	}
	return ListSysImages()
}

// Exists whether the image file exists.
func Exists(img Image) (bool, error) {
	switch _, err := os.Stat(img.Filepath()); {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, errors.Trace(err)
	}
}
