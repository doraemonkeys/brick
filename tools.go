package brick

import (
	"io/fs"
	"os"

	"github.com/doraemonkeys/doraemon"
)

func RandomLiveID() string {
	return doraemon.GenRandomString("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 25)
}
func WriteFilePerm(name string, data []byte) error {
	perm := fs.FileMode(0644)
	f, err := os.Stat(name)
	if err == nil {
		perm = f.Mode()
	}
	return os.WriteFile(name, data, perm)
}
